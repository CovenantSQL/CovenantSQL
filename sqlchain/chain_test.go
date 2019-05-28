/*
 * Copyright 2018 The CovenantSQL Authors.
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

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/consistent"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	rpc "github.com/CovenantSQL/CovenantSQL/rpc/mux"
	"github.com/CovenantSQL/CovenantSQL/types"
)

var (
	testPeersNumber                 = 5
	testPeriod                      = 1 * time.Second
	testTick                        = 100 * time.Millisecond
	testQueryTTL             int32  = 10
	testDatabaseID                  = proto.DatabaseID(hash.THashH([]byte{'d', 'b'}).String())
	testPeriodNumber         int32  = 10
	testClientNumberPerChain        = 3
	testUpdatePeriod         uint64 = 2
)

type chainParams struct {
	dbfile string
	server *rpc.Server
	mux    *MuxService
	config *Config
	chain  *Chain
}

func TestIndexKey(t *testing.T) {
	for i := 0; i < 10; i++ {
		b1, err := createRandomBlock(genesisHash, false)

		if err != nil {
			t.Fatalf("error occurred: %v", err)
		}

		b2, err := createRandomBlock(genesisHash, false)

		if err != nil {
			t.Fatalf("error occurred: %v", err)
		}

		// Test partial order
		bi1 := newBlockNode(rand.Int31(), b1, nil)
		bi2 := newBlockNode(rand.Int31(), b2, nil)
		k1 := bi1.indexKey()
		k2 := bi2.indexKey()

		if c1, c2 := bytes.Compare(k1, k2) < 0, bi1.height < bi2.height; c1 != c2 {
			t.Fatalf("unexpected compare result: heights=%d,%d keys=%s,%s",
				bi1.height, bi2.height, hex.EncodeToString(k1), hex.EncodeToString(k2))
		}

		if c1, c2 := bytes.Compare(k1, k2) > 0, bi1.height > bi2.height; c1 != c2 {
			t.Fatalf("unexpected compare result: heights=%d,%d keys=%s,%s",
				bi1.height, bi2.height, hex.EncodeToString(k1), hex.EncodeToString(k2))
		}
	}
}

func TestMultiChain(t *testing.T) {
	//log.SetLevel(log.InfoLevel)
	// Create genesis block
	genesis, err := createRandomBlock(genesisHash, true)

	if err != nil {
		t.Fatalf("error occurred: %v", err)
	}

	// Create peer list: `testPeersNumber` miners + 1 block producer
	nis, peers, err := createTestPeers(testPeersNumber + 1)

	if err != nil {
		t.Fatalf("error occurred: %v", err)
	}

	for i, p := range peers.Servers {
		t.Logf("peer #%d: %s", i, p)
	}

	// Create config info from created nodes
	bpinfo := &conf.BPInfo{
		PublicKey: testPubKey,
		NodeID:    peers.Servers[testPeersNumber],
		Nonce:     nis[testPeersNumber].Nonce,
	}
	knownnodes := make([]proto.Node, 0, testPeersNumber+1)

	for i, v := range peers.Servers {
		knownnodes = append(knownnodes, proto.Node{
			ID: v,
			Role: func() proto.ServerRole {
				if i < testPeersNumber {
					return proto.Miner
				}
				return proto.Leader
			}(),
			Addr:      "",
			PublicKey: testPubKey,
			Nonce:     nis[i].Nonce,
		})
	}

	// Rip BP from peer list
	peers.Servers = peers.Servers[:testPeersNumber]

	// Create sql-chain instances
	chains := make([]*chainParams, testPeersNumber)

	for i := range chains {
		// Combine data file path
		dbfile := path.Join(testDataDir, fmt.Sprintf("%s-%02d", t.Name(), i))

		// Create new RPC server
		server := rpc.NewServer()

		if err = server.InitRPCServer("127.0.0.1:0", testPrivKeyFile, testMasterKey); err != nil {
			t.Fatalf("error occurred: %v", err)
		}

		go server.Serve()
		defer server.Stop()

		// Create multiplexing service from RPC server
		mux, err := NewMuxService(route.SQLChainRPCName, server)

		if err != nil {
			t.Fatalf("error occurred: %v", err)
		}

		// Create chain instance
		config := &Config{
			DatabaseID:      testDatabaseID,
			ChainFilePrefix: dbfile,
			DataFile:        dbfile,
			Genesis:         genesis,
			Period:          testPeriod,
			Tick:            testTick,
			MuxService:      mux,
			Server:          peers.Servers[i],
			Peers:           peers,
			QueryTTL:        testQueryTTL,
			UpdatePeriod:    testUpdatePeriod,
		}
		chain, err := NewChain(config)

		if err != nil {
			t.Fatalf("error occurred: %v", err)
		}

		// Set chain parameters
		chains[i] = &chainParams{
			dbfile: dbfile,
			server: server,
			mux:    mux,
			config: config,
			chain:  chain,
		}

	}

	// Create a master BP for RPC test
	bpsvr := rpc.NewServer()

	if err = bpsvr.InitRPCServer("127.0.0.1:0", testPrivKeyFile, testMasterKey); err != nil {
		return
	}

	go bpsvr.Serve()
	defer bpsvr.Stop()

	// Create global config and initialize route table
	knownnodes[testPeersNumber].Addr = bpsvr.Listener.Addr().String()

	for i, v := range chains {
		knownnodes[i].Addr = v.server.Listener.Addr().String()
	}

	conf.GConf = &conf.Config{
		UseTestMasterKey: true,
		GenerateKeyPair:  false,
		WorkingRoot:      testDataDir,
		PubKeyStoreFile:  "public.keystore",
		PrivateKeyFile:   "private.key",
		DHTFileName:      "dht.db",
		ListenAddr:       bpsvr.Listener.Addr().String(),
		ThisNodeID:       bpinfo.NodeID,
		ValidDNSKeys: map[string]string{
			"koPbw9wmYZ7ggcjnQ6ayHyhHaDNMYELKTqT+qRGrZpWSccr/lBcrm10Z1PuQHB3Azhii+sb0PYFkH1ruxLhe5g==": "cloudflare.com",
			"mdsswUyr3DPW132mOi8V9xESWE8jTo0dxCjjnopKl+GqJxpVXckHAeF+KkxLbxILfDLUT0rAK9iUzy1L53eKGQ==": "cloudflare.com",
		},
		MinNodeIDDifficulty: 2,
		DNSSeed: conf.DNSSeed{
			EnforcedDNSSEC: false,
			DNSServers: []string{
				"1.1.1.1",
				"202.46.34.74",
				"202.46.34.75",
				"202.46.34.76",
			},
		},
		BP:         bpinfo,
		KnownNodes: knownnodes,
	}

	// Start BP
	if dht, err := route.NewDHTService(testDHTStoreFile, new(consistent.KMSStorage), true); err != nil {
		t.Fatalf("error occurred: %v", err)
	} else if err = bpsvr.RegisterService(route.DHTRPCName, dht); err != nil {
		t.Fatalf("error occurred: %v", err)
	}

	for _, n := range conf.GConf.KnownNodes {
		rawNodeID := n.ID.ToRawNodeID()
		if err = route.SetNodeAddrCache(rawNodeID, n.Addr); err != nil {
			t.Fatalf("error occurred: %v", err)
		}
		node := &proto.Node{
			ID:        n.ID,
			Addr:      n.Addr,
			PublicKey: n.PublicKey,
			Nonce:     n.Nonce,
			Role:      n.Role,
		}

		if err = kms.SetNode(node); err != nil {
			t.Fatalf("error occurred: %v", err)
		}

		if n.ID == conf.GConf.ThisNodeID {
			kms.SetLocalNodeIDNonce(rawNodeID.CloneBytes(), &n.Nonce)
		}
	}

	// Test chain data reloading before exit
	for _, v := range chains {
		defer func(p *chainParams) {
			if chain, err := NewChain(p.config); err != nil {
				t.Errorf("error occurred: %v", err)
			} else {
				t.Logf("load chain from file %s: head = %s height = %d",
					p.dbfile, chain.rt.getHead().Head, chain.rt.getHead().Height)
			}
		}(v)
	}

	// Start all chain instances
	for _, v := range chains {
		if err = v.chain.Start(); err != nil {
			t.Fatalf("error occurred: %v", err)
		}
		defer func(c *Chain) {
			// Stop chain main process before exit
			_ = c.Stop()
		}(v.chain)
	}

	// Should be able to fetch all acks in all peers
	for _, v := range chains {
		defer func(c *Chain) {
			var ch = c.rt.getHead().Height
			for i := int32(0); i <= ch; i++ {
				var node *blockNode
				if node = c.rt.getHead().node.ancestor(i); node == nil {
					t.Logf("block at height %d not found in peer %s, continue",
						i, c.rt.getPeerInfoString())
					continue
				}
				block := node.load()
				if block == nil {
					var err error
					if block, err = c.FetchBlock(node.height); err != nil || block == nil {
						t.Errorf("failed to load block %v at height %d in peer %s: %v",
							block.BlockHash(), i, c.rt.getPeerInfoString(), err)
						continue
					}
				}
				t.Logf("checking block %v at height %d in peer %s",
					block.BlockHash(), i, c.rt.getPeerInfoString())
			}
		}(v.chain)
	}

	// Create table
	cli, err := newRandomNode(chains[0].chain, true)
	if err != nil {
		t.Fatalf("error occurred: %v", err)
	}
	req, err := cli.buildQuery(types.WriteQuery, []types.Query{
		buildQuery(`CREATE TABLE t1 (k INT, v TEXT, PRIMARY KEY(k))`),
		buildQuery(`INSERT INTO t1 (k, v) VALUES (?, ?), (?, ?), (?, ?), (?, ?), (?, ?)`,
			1, "v1", 2, "v2", 3, "v3", 4, "v4", 5, "v5",
		),
	})
	if err != nil {
		t.Fatalf("error occurred: %v", err)
	}
	for i, v := range chains {
		cli, err := newRandomNode(v.chain, i == 0)
		if err != nil {
			t.Fatalf("error occurred: %v", err)
		}
		err = cli.sendQuery(req)
		if err != nil {
			t.Fatalf("error occurred: %v", err)
		}
	}

	// Create some random clients to push new queries
	for i, v := range chains {
		sC := make(chan struct{})
		wg := &sync.WaitGroup{}

		for j := 0; j < testClientNumberPerChain; j++ {
			cli, err := newRandomNode(v.chain, i == 0)

			if err != nil {
				t.Fatalf("error occurred: %v", err)
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
						var err error
						// Send a random query
						err = cli.query(types.ReadQuery, []types.Query{
							buildQuery(`SELECT v FROM t1 WHERE k=?`, rand.Intn(5)),
						})
						if err != nil {
							t.Errorf("error occurred: %v", err)
						}
					}
				}
			}(v.chain, cli)
		}

		defer func() {
			// Quit client goroutines
			close(sC)
			wg.Wait()
		}()
	}

	time.Sleep(time.Duration(testPeriodNumber) * testPeriod)
}
