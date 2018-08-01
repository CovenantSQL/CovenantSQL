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
	"math/rand"
	"path"
	"sync"
	"testing"
	"time"

	"gitlab.com/thunderdb/ThunderDB/proto"
	"gitlab.com/thunderdb/ThunderDB/route"
	"gitlab.com/thunderdb/ThunderDB/rpc"
)

var (
	testPeersNumber                           = 5
	testPeriod                                = 1 * time.Second
	testTick                                  = 100 * time.Millisecond
	testQueryTTL             int32            = 10
	testDatabaseID           proto.DatabaseID = "tdb-test"
	testChainService                          = "SQLC"
	testPeriodNumber         int32            = 10
	testClientNumberPerChain                  = 3
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
		bi1 := newBlockNode(rand.Int31(), b1, nil)
		bi2 := newBlockNode(rand.Int31(), b2, nil)
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

	for i, p := range peers.Servers {
		t.Logf("Peer #%d: %s", i, p.ID)
	}

	// Create sql-chain instances
	chains := make([]*Chain, testPeersNumber)

	for i := range chains {
		dataFile := path.Join(testDataDir, fmt.Sprintf("%s-%02d", t.Name(), i))

		defer func(c *Chain, db string) {
			// Try to reload chain
			if nc, err := LoadChain(&Config{
				DatabaseID: testDatabaseID,
				DataFile:   db,
				Period:     testPeriod,
				Tick:       testTick,
				MuxService: NewMuxService(testChainService, rpc.NewServer()),
				Server:     peers.Servers[i],
				Peers:      peers,
				QueryTTL:   testQueryTTL,
			}); err != nil {
				t.Errorf("Error occurred: %v", err)
			} else {
				t.Logf("Load chain from file %s: head = %s height = %d",
					db, nc.rt.getHead().Head, nc.rt.getHead().Height)
			}
		}(chains[i], dataFile)
	}

	for i := range chains {
		// Create RPC server
		server := rpc.NewServer()

		if err = server.InitRPCServer("127.0.0.1:0", testPrivKeyFile, testMasterKey); err != nil {
			t.Fatalf("Error occurred: %v", err)
		}

		go server.Serve()
		defer server.Stop()
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

		defer func(c *Chain) {
			// Stop chain main process
			c.Stop()
		}(chains[i])
	}

	// Create some random clients to push new queries
	for i := range chains {
		sC := make(chan struct{})
		wg := &sync.WaitGroup{}
		wk := &nodeProfile{
			NodeID:     peers.Servers[i].ID,
			PrivateKey: testPrivKey,
			PublicKey:  testPubKey,
		}

		for j := 0; j < testClientNumberPerChain; j++ {
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
						// Send a random query
						resp, err := createRandomQueryResponse(p, wk)

						if err != nil {
							t.Errorf("Error occurred: %v", err)
						} else if err = c.VerifyAndPushResponsedQuery(resp); err != nil {
							t.Errorf("Error occurred: %v", err)
						}

						time.Sleep(time.Duration(rand.Int63n(500)+1) * time.Millisecond)
						ack, err := createRandomQueryAckWithResponse(resp, p)

						if err != nil {
							t.Errorf("Error occurred: %v", err)
						} else if err = c.VerifyAndPushAckedQuery(ack); err != nil {
							t.Errorf("Error occurred: %v", err)
						}
					}
				}
			}(chains[i], cli)
		}

		defer func() {
			// Quit client goroutines
			close(sC)
			wg.Wait()
		}()
	}

	time.Sleep(time.Duration(testPeriodNumber) * testPeriod)
	return
}
