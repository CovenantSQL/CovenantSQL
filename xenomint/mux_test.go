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
	"fmt"
	"math/rand"
	"sync/atomic"
	"testing"

	"github.com/pkg/errors"
	. "github.com/smartystreets/goconvey/convey"

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

func setupMuxParallel(priv *ca.PrivateKey) (
	bp, miner *nodeRPCInfo, ms *MuxService, err error,
) {
	var (
		nis        []proto.Node
		dht        *route.DHTService
		bpSv, mnSv *rpc.Server
	)
	if nis, err = createNodesWithPublicKey(priv.PubKey(), testingNonceDifficulty, 3); err != nil {
		err = errors.Wrap(err, "failed to setup bench environment: ")
		return
	} else if l := len(nis); l != 3 {
		err = errors.Wrapf(err, "failed to setup bench environment: unexpected length %d", l)
		return
	}
	// Setup block producer RPC and register server address
	bpSv = rpc.NewServer()
	if err = bpSv.InitRPCServer(
		"localhost:0", testingPrivateKeyFile, testingMasterKey,
	); err != nil {
		err = errors.Wrap(err, "failed to setup bench environment: ")
		return
	}
	nis[0].Addr = bpSv.Listener.Addr().String()
	nis[0].Role = proto.Leader
	// Setup miner RPC and register server address
	mnSv = rpc.NewServer()
	if err = mnSv.InitRPCServer(
		"localhost:0", testingPrivateKeyFile, testingMasterKey,
	); err != nil {
		err = errors.Wrap(err, "failed to setup bench environment: ")
		return
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
		err = errors.Wrap(err, "failed to setup bench environment: ")
		return
	} else if err = bpSv.RegisterService(route.DHTRPCName, dht); err != nil {
		err = errors.Wrap(err, "failed to setup bench environment: ")
		return
	}
	kms.SetLocalNodeIDNonce(nis[2].ID.ToRawNodeID().CloneBytes(), &nis[2].Nonce)
	for i := range nis {
		route.SetNodeAddrCache(nis[i].ID.ToRawNodeID(), nis[i].Addr)
		kms.SetNode(&nis[i])
	}
	// Register mux service
	if ms, err = NewMuxService(benchmarkRPCName, mnSv); err != nil {
		err = errors.Wrap(err, "failed to setup bench environment: ")
		return
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

func setupBenchmarkMuxParallel(b *testing.B) (
	bp, miner *nodeRPCInfo, ms *MuxService, r []*MuxQueryRequest,
) {
	var err error
	var priv *ca.PrivateKey
	// Use testing private key to create several nodes
	if priv, err = kms.GetLocalPrivateKey(); err != nil {
		b.Fatalf("failed to setup bench environment: %v", err)
	}

	bp, miner, ms, err = setupMuxParallel(priv)
	if err != nil {
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

	return
}

func teardownBenchmarkMuxParallel(bpSv, mnSv *rpc.Server) {
	//ca.BypassSignature = false
	mnSv.Stop()
	bpSv.Stop()
}

func BenchmarkMuxParallel(b *testing.B) {
	var bp, s, ms, r = setupBenchmarkMuxParallel(b)
	defer teardownBenchmarkMuxParallel(bp.server, s.server)
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
				c       *Chain
			)

			c = setupBenchmarkChain(b)
			ms.register(benchmarkDatabaseID, c)
			b.ResetTimer()
			defer func() {
				b.StopTimer()
				ms.unregister(benchmarkDatabaseID)
				teardownBenchmarkChain(b, c)
			}()

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

func TestMuxService(t *testing.T) {
	Convey("test xenomint MuxService", t, func() {
		var (
			err       error
			priv      *ca.PrivateKey
			bp, miner *nodeRPCInfo
			ms        *MuxService
			c         *Chain

			sel = `SELECT v1, v2, v3 FROM bench WHERE k=?`
			rr  *types.Request

			method = fmt.Sprintf("%s.%s", benchmarkRPCName, "Query")
		)
		// Use testing private key to create several nodes
		priv, err = kms.GetLocalPrivateKey()
		So(err, ShouldBeNil)

		bp, miner, ms, err = setupMuxParallel(priv)
		So(err, ShouldBeNil)
		defer teardownBenchmarkMuxParallel(bp.server, miner.server)

		var caller = rpc.NewPersistentCaller(miner.node.ID)

		c, err = setupChain(t.Name())
		So(err, ShouldBeNil)

		ms.register(benchmarkDatabaseID, c)
		defer func() {
			ms.unregister(benchmarkDatabaseID)
			teardownChain(t.Name(), c)
		}()

		// Setup query requests
		rr = buildRequest(types.ReadQuery, []types.Query{
			buildQuery(sel, 0),
		})
		err = rr.Sign(priv)
		So(err, ShouldBeNil)

		r := &MuxQueryRequest{
			DatabaseID: benchmarkDatabaseID,
			Request:    rr,
		}
		err = caller.Call(
			method, &r, &MuxQueryResponse{},
		)
		So(err, ShouldBeNil)
		err = c.state.commit()
		So(err, ShouldBeNil)

	})
}
