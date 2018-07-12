/*
 * Copyright 2018 The ThunderDB Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package sqlchain

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math/rand"
	"path"
	"sync"
	"testing"
	"time"

	bolt "github.com/coreos/bbolt"
	"gitlab.com/thunderdb/ThunderDB/crypto/kms"
	"gitlab.com/thunderdb/ThunderDB/kayak"
	"gitlab.com/thunderdb/ThunderDB/proto"
	"gitlab.com/thunderdb/ThunderDB/route"
	"gitlab.com/thunderdb/ThunderDB/rpc"
	ct "gitlab.com/thunderdb/ThunderDB/sqlchain/types"
)

var (
	testPeersNumber                   = 5
	testPeriod                        = 1 * time.Second
	testTick                          = 100 * time.Millisecond
	testQueryTTL     int32            = 10
	testDatabaseID   proto.DatabaseID = "tdb-test"
	testChainService                  = "sql-chain.thunderdb.rpc"
)

func TestIndexKey(t *testing.T) {
	for i := 0; i < 10; i++ {
		b1, err := createRandomBlock(genesisHash, false)

		if err != nil {
			t.Fatalf("Error occurred: %v", err)
		}

		b2, err := createRandomBlock(genesisHash, false)

		if err != nil {
			t.Fatalf("Error occurred: %v", err)
		}

		// Test partial order
		bi1 := newBlockNode(b1, nil)
		bi2 := newBlockNode(b2, nil)
		bi1.height = rand.Int31()
		bi2.height = rand.Int31()
		k1 := bi1.indexKey()
		k2 := bi2.indexKey()

		if c1, c2 := bytes.Compare(k1, k2) < 0, bi1.height < bi2.height; c1 != c2 {
			t.Fatalf("Unexpected compare result: heights=%d,%d keys=%s,%s",
				bi1.height, bi2.height, hex.EncodeToString(k1), hex.EncodeToString(k2))
		}

		if c1, c2 := bytes.Compare(k1, k2) > 0, bi1.height > bi2.height; c1 != c2 {
			t.Fatalf("Unexpected compare result: heights=%d,%d keys=%s,%s",
				bi1.height, bi2.height, hex.EncodeToString(k1), hex.EncodeToString(k2))
		}
	}
}

func TestChain(t *testing.T) {
	fl, err := ioutil.TempFile("", "chain")

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	fl.Close()

	// Create new chain
	genesis, err := createRandomBlock(genesisHash, true)

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	pub, err := kms.GetLocalPublicKey()

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	priv, err := kms.GetLocalPrivateKey()

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	servers := [...]*kayak.Server{
		&kayak.Server{ID: "X1"},
		&kayak.Server{ID: "X2"},
		&kayak.Server{ID: "X3"},
		&kayak.Server{ID: "X4"},
		&kayak.Server{ID: "X5"},
	}

	peers := &kayak.Peers{
		Term:    0,
		Leader:  servers[0],
		Servers: servers[:],
		PubKey:  pub,
	}

	if err = peers.Sign(priv); err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	chain, err := NewChain(&Config{
		DatabaseID: "tdb",
		DataFile:   fl.Name(),
		Genesis:    genesis,
		Period:     testPeriod,
		Tick:       testTick,
		QueryTTL:   testQueryTTL,
		MuxService: NewMuxService("sqlchain", rpc.NewServer()),
		Server:     servers[0],
		Peers:      peers,
	})

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	t.Logf("Create new chain: genesis = %s, inittime = %s, period = %.9f secs",
		genesis.SignedHeader.BlockHash,
		chain.rt.chainInitTime.Format(time.RFC3339Nano),
		chain.rt.period.Seconds())

	// Push blocks
	for {
		t.Logf("Chain state: head = %s, height = %d, turn = %d, nextturnstart = %s, ismyturn = %t",
			chain.st.Head, chain.st.Height, chain.rt.nextTurn,
			chain.rt.chainInitTime.Add(
				chain.rt.period*time.Duration(chain.rt.nextTurn)).Format(time.RFC3339Nano),
			chain.rt.isMyTurn())
		acks, err := createRandomQueries(10)

		if err != nil {
			t.Fatalf("Error occurred: %v", err)
		}

		for _, ack := range acks {
			if err = chain.VerifyAndPushAckedQuery(ack); err != nil {
				t.Fatalf("Error occurred: %v", err)
			}
		}

		// Run main cycle
		var now time.Time
		var d time.Duration

		for {
			now, d = chain.rt.nextTick()

			t.Logf("Wake up at: now = %s, d = %.9f secs",
				now.Format(time.RFC3339Nano), d.Seconds())

			if d > 0 {
				time.Sleep(d)
			} else {
				chain.runCurrentTurn(now)
				break
			}
		}

		// Advise block if it's not my turn
		if !chain.rt.isMyTurn() {
			block, err := createRandomBlockWithQueries(
				genesis.SignedHeader.BlockHash, chain.st.Head, acks)

			if err != nil {
				t.Fatalf("Error occurred: %v", err)
			}

			if err = chain.CheckAndPushNewBlock(block); err != nil {
				t.Fatalf("Error occurred: %v, block = %+v", err, block)
			}

			t.Logf("Pushed new block: height = %d, %s <- %s",
				chain.st.Height,
				block.SignedHeader.ParentHash,
				block.SignedHeader.BlockHash)
		} else {
			var enc []byte
			var block ct.Block

			if err = chain.db.View(func(tx *bolt.Tx) (err error) {
				enc = tx.Bucket(metaBucket[:]).Bucket(metaBlockIndexBucket).Get(
					chain.st.node.indexKey())
				return
			}); err != nil {
				t.Fatalf("Error occurred: %v", err)
			}

			if err = block.UnmarshalBinary(enc); err != nil {
				t.Fatalf("Error occurred: %v", err)
			}

			t.Logf("Produced new block: height = %d, %s <- %s",
				chain.st.Height,
				block.SignedHeader.ParentHash,
				block.SignedHeader.BlockHash)
		}

		if chain.st.Height >= testHeight {
			break
		}
	}

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	// Reload chain from DB file and rebuild memory cache
	chain.db.Close()
	chain, err = LoadChain(&Config{
		DataFile:   fl.Name(),
		Genesis:    genesis,
		Period:     testPeriod,
		Tick:       testTick,
		QueryTTL:   testQueryTTL,
		MuxService: NewMuxService("sqlchain", rpc.NewServer()),
		Server: &kayak.Server{
			ID: proto.NodeID("X1"),
		},
		Peers: peers,
	})

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}
}

func TestMultiChain(t *testing.T) {
	// Create genesis block
	genesis, err := createRandomBlock(genesisHash, true)

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	// Create peer list
	nis, peers, err := createTestPeers(testPeersNumber)

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	// Create sql-chain instances
	chains := make([]*Chain, testPeersNumber)

	for i := range chains {
		// Create RPC server
		server := rpc.NewServer()

		if err = server.InitRPCServer("127.0.0.1:0", testPrivKeyFile, testMasterKey); err != nil {
			t.Fatalf("Error occurred: %v", err)
		}

		go server.Serve()
		defer func() {
			server.Listener.Close()
			server.Stop()
		}()
		mux := NewMuxService(testChainService, server)

		// Register address
		if err = route.SetNodeAddrCache(
			&proto.RawNodeID{Hash: nis[i].Hash},
			server.Listener.Addr().String(),
		); err != nil {
			t.Fatalf("Error occurred: %v", err)
		}

		// Create sql-chain instance
		dataFile := path.Join(testDataDir, fmt.Sprintf("%s-%02d", t.Name(), i))
		chains[i], err = NewChain(&Config{
			DatabaseID: testDatabaseID,
			DataFile:   dataFile,
			Genesis:    genesis,
			Period:     testPeriod,
			Tick:       testTick,
			MuxService: mux,
			Server:     peers.Servers[i],
			Peers:      peers,
			QueryTTL:   testQueryTTL,
		})

		if err != nil {
			t.Fatalf("Error occurred: %v", err)
		}

		if err = chains[i].Start(); err != nil {
			t.Fatalf("Error occurred: %v", err)
		}

		// Create some random clients to push new queries
		sC := make(chan struct{})
		wg := &sync.WaitGroup{}
		wk := &nodeProfile{
			NodeID:     peers.Servers[i].ID,
			PrivateKey: testPrivKey,
			PublicKey:  testPubKey,
		}

		for x := 0; x < 10; x++ {
			cli, err := newRandomNode()

			if err != nil {
				t.Fatalf("Error occurred: %v", err)
			}

			wg.Add(1)
			go func(c *Chain, p *nodeProfile) {
				defer wg.Done()
			foreverLoop:
				for {
					select {
					case <-sC:
						break foreverLoop
					default:
						time.Sleep(time.Duration(rand.Int63n(100)+1) * time.Millisecond)
						// Send a random query
						ack, err := createRandomQueryAck(p, wk)

						if err != nil {
							t.Errorf("Error occurred: %v", err)
						} else if err = c.VerifyAndPushAckedQuery(ack); err != nil {
							t.Errorf("Error occurred: %v", err)
						}
					}
				}
			}(chains[i], cli)
		}

		defer func(c *Chain) {
			// Quit client goroutines
			close(sC)
			wg.Wait()
			// Stop chain main process
			c.Stop()
		}(chains[i])
	}

	time.Sleep(10 * testPeriod)
	return
}
