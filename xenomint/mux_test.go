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

package xenomint

import (
	"database/sql"
	"fmt"
	"math/rand"
	"os"
	"path"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/CovenantSQL/CovenantSQL/conf"
	con "github.com/CovenantSQL/CovenantSQL/consistent"
	ca "github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	"github.com/CovenantSQL/CovenantSQL/types"
)

type nodeRPCInfo struct {
	node   proto.Node
	server *rpc.Server
}

func setupBenchmarkMuxParallel(b *testing.B) (
	bp, miner *nodeRPCInfo, ms *MuxService, r []*MuxQueryRequest,
) {
	var (
		priv       *ca.PrivateKey
		nis        []proto.Node
		dht        *route.DHTService
		bpSv, mnSv *rpc.Server
		err        error
	)
	// Use testing private key to create several nodes
	if priv, err = kms.GetLocalPrivateKey(); err != nil {
		b.Fatalf("failed to setup bench environment: %v", err)
	}
	if nis, err = createNodesWithPublicKey(priv.PubKey(), testingNonceDifficulty, 3); err != nil {
		b.Fatalf("failed to setup bench environment: %v", err)
	} else if l := len(nis); l != 3 {
		b.Fatalf("failed to setup bench environment: unexpected length %d", l)
	}
	// Setup block producer RPC and register server address
	bpSv = rpc.NewServer()
	if err = bpSv.InitRPCServer(
		"localhost:0", testingPrivateKeyFile, testingMasterKey,
	); err != nil {
		b.Fatalf("failed to setup bench environment: %v", err)
	}
	nis[0].Addr = bpSv.Listener.Addr().String()
	nis[0].Role = proto.Leader
	// Setup miner RPC and register server address
	mnSv = rpc.NewServer()
	if err = mnSv.InitRPCServer(
		"localhost:0", testingPrivateKeyFile, testingMasterKey,
	); err != nil {
		b.Fatalf("failed to setup bench environment: %v", err)
	}
	nis[1].Addr = mnSv.Listener.Addr().String()
	nis[1].Role = proto.Miner
	// Setup client
	nis[2].Role = proto.Client
	// Setup global config
	conf.GConf = &conf.Config{
		IsTestMode:          true,
		GenerateKeyPair:     false,
		MinNodeIDDifficulty: testingNonceDifficulty,
		BP: &conf.BPInfo{
			PublicKey: priv.PubKey(),
			NodeID:    nis[0].ID,
			Nonce:     nis[0].Nonce,
		},
		KnownNodes: nis,
	}
	// Register DHT service, this will also initialize the public key store
	if dht, err = route.NewDHTService(
		testingPublicKeyStoreFile, &con.KMSStorage{}, true,
	); err != nil {
		b.Fatalf("failed to setup bench environment: %v", err)
	} else if err = bpSv.RegisterService(route.DHTRPCName, dht); err != nil {
		b.Fatalf("failed to setup bench environment: %v", err)
	}
	kms.SetLocalNodeIDNonce(nis[2].ID.ToRawNodeID().CloneBytes(), &nis[2].Nonce)
	for i := range nis {
		route.SetNodeAddrCache(nis[i].ID.ToRawNodeID(), nis[i].Addr)
		kms.SetNode(&nis[i])
	}
	// Register mux service
	if ms, err = NewMuxService(benchmarkRPCName, mnSv); err != nil {
		b.Fatalf("failed to setup bench environment: %v", err)
	}

	// Setup query requests
	var (
		sel = `SELECT v1, v2, v3 FROM bench WHERE k=?`
		ins = `INSERT INTO bench VALUES (?, ?, ?, ?)`
		src = make([][]interface{}, benchmarkNewKeyLength)
	)
	r = make([]*MuxQueryRequest, benchmarkMaxKey)
	// Read query key space [0, n-1]
	for i := 0; i < benchmarkReservedKeyLength; i++ {
		var req = buildRequest(types.ReadQuery, []types.Query{
			buildQuery(sel, i+benchmarkReservedKeyOffset),
		})
		if err = req.Sign(priv); err != nil {
			b.Fatalf("failed to setup bench environment: %v", err)
		}
		r[i] = &MuxQueryRequest{
			DatabaseID: benchmarkDatabaseID,
			Request:    req,
		}
	}
	// Write query key space [n, 2n-1]
	for i := range src {
		var vals [benchmarkVNum][benchmarkVLen]byte
		src[i] = make([]interface{}, benchmarkVNum+1)
		src[i][0] = i + benchmarkNewKeyOffset
		for j := range vals {
			rand.Read(vals[j][:])
			src[i][j+1] = string(vals[j][:])
		}
	}
	for i := 0; i < benchmarkNewKeyLength; i++ {
		var req = buildRequest(types.WriteQuery, []types.Query{
			buildQuery(ins, src[i]...),
		})
		if err = req.Sign(priv); err != nil {
			b.Fatalf("failed to setup bench environment: %v", err)
		}
		r[i+benchmarkNewKeyOffset] = &MuxQueryRequest{
			DatabaseID: benchmarkDatabaseID,
			Request:    req,
		}
	}

	bp = &nodeRPCInfo{
		node:   nis[0],
		server: bpSv,
	}
	miner = &nodeRPCInfo{
		node:   nis[1],
		server: mnSv,
	}

	go bpSv.Serve()
	go mnSv.Serve()
	//ca.BypassSignature = true
	return
}

func teardownBenchmarkMuxParallel(b *testing.B, bpSv, mnSv *rpc.Server) {
	//ca.BypassSignature = false
	mnSv.Stop()
	bpSv.Stop()
}

func setupSubBenchmarkMuxParallel(b *testing.B, ms *MuxService) (c *Chain) {
	// Setup chain state
	var (
		fl   = path.Join(testingDataDir, strings.Replace(b.Name(), "/", "-", -1))
		err  error
		stmt *sql.Stmt
	)
	if c, err = NewChain(fmt.Sprint("file:", fl)); err != nil {
		b.Fatalf("failed to setup bench environment: %v", err)
	}
	if _, err = c.state.strg.Writer().Exec(
		`CREATE TABLE "bench" ("k" INT, "v1" TEXT, "v2" TEXT, "v3" TEXT, PRIMARY KEY("k"))`,
	); err != nil {
		b.Fatalf("failed to setup bench environment: %v", err)
	}
	if stmt, err = c.state.strg.Writer().Prepare(
		`INSERT INTO "bench" VALUES (?, ?, ?, ?)`,
	); err != nil {
		b.Fatalf("failed to setup bench environment: %v", err)
	}
	for i := 0; i < benchmarkReservedKeyLength; i++ {
		var (
			vals [benchmarkVNum][benchmarkVLen]byte
			args [benchmarkVNum + 1]interface{}
		)
		args[0] = i + benchmarkReservedKeyOffset
		for i := range vals {
			rand.Read(vals[i][:])
			args[i+1] = string(vals[i][:])
		}
		if _, err = stmt.Exec(args[:]...); err != nil {
			b.Fatalf("failed to setup bench environment: %v", err)
		}
	}
	ms.register(benchmarkDatabaseID, c)

	allKeyPermKeygen.reset()
	newKeyPermKeygen.reset()

	b.ResetTimer()
	return
}

func teardownSubBenchmarkMuxParallel(b *testing.B, ms *MuxService) {
	b.StopTimer()

	var (
		fl  = path.Join(testingDataDir, strings.Replace(b.Name(), "/", "-", -1))
		err error
		c   *Chain
	)
	// Stop RPC server
	if c, err = ms.route(benchmarkDatabaseID); err != nil {
		b.Fatalf("failed to teardown bench environment: %v", err)
	}
	ms.unregister(benchmarkDatabaseID)
	// Close chain
	if err = c.Stop(); err != nil {
		b.Fatalf("failed to teardown bench environment: %v", err)
	}
	if err = os.Remove(fl); err != nil {
		b.Fatalf("failed to teardown bench environment: %v", err)
	}
	if err = os.Remove(fmt.Sprint(fl, "-shm")); err != nil && !os.IsNotExist(err) {
		b.Fatalf("failed to teardown bench environment: %v", err)
	}
	if err = os.Remove(fmt.Sprint(fl, "-wal")); err != nil && !os.IsNotExist(err) {
		b.Fatalf("failed to teardown bench environment: %v", err)
	}
}

func BenchmarkMuxParallel(b *testing.B) {
	var bp, s, ms, r = setupBenchmarkMuxParallel(b)
	defer teardownBenchmarkMuxParallel(b, bp.server, s.server)
	var benchmarks = []struct {
		name string
		kg   keygen
	}{
		{
			name: "Write",
			kg:   newKeyPermKeygen,
		}, {
			name: "MixRW",
			kg:   allKeyPermKeygen,
		},
	}
	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			var (
				counter int32

				c = setupSubBenchmarkMuxParallel(b, ms)
			)
			defer teardownSubBenchmarkMuxParallel(b, ms)
			b.RunParallel(func(pb *testing.PB) {
				var (
					err    error
					method = fmt.Sprintf("%s.%s", benchmarkRPCName, "Query")
					caller = rpc.NewPersistentCaller(s.node.ID)
				)
				for pb.Next() {
					if err = caller.Call(
						method, &r[bm.kg.next()], &MuxQueryResponse{},
					); err != nil {
						b.Fatalf("failed to execute: %v", err)
					}
					if atomic.AddInt32(&counter, 1)%benchmarkQueriesPerBlock == 0 {
						if err = c.state.commit(); err != nil {
							b.Fatalf("failed to commit block: %v", err)
						}
					}
				}
			})
		})
	}
}
