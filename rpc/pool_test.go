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

package rpc

import (
	"sync"
	"sync/atomic"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/CovenantSQL/CovenantSQL/proto"
)

func setupServer(node *proto.Node) (server *Server, err error) {
	if server, err = NewServerWithService(
		ServiceMap{"Count": &CountService{host: node.ID}},
	); err != nil {
		return nil, err
	}
	if err = server.InitRPCServer(":0", privateKey, []byte{}); err != nil {
		return nil, err
	}
	// register to resolver
	node.Addr = server.Listener.Addr().String()
	defaultResolver.registerNode(node)
	return
}

func setupServers(nodes []*proto.Node) (stop func(), err error) {
	servers := make([]*Server, len(nodes))
	for i, v := range nodes {
		if servers[i], err = setupServer(v); err != nil {
			return
		}
	}

	wg := &sync.WaitGroup{}
	for _, v := range servers {
		wg.Add(1)
		go func(server *Server) {
			defer wg.Done()
			server.Serve()
		}(v)
	}

	return func() {
		for _, v := range nodes {
			defaultResolver.deleteNode(*(v.ID.ToRawNodeID()))
		}
		for _, v := range servers {
			v.Stop()
		}
		wg.Wait()
	}, nil
}

func setupEnvironment(n int) ([]*proto.Node, func(), error) {
	nodes, err := createLocalNodes(10, n)
	if err != nil {
		return nil, nil, err
	}
	stop, err := setupServers(nodes)
	if err != nil {
		return nil, nil, err
	}
	return nodes, stop, nil
}

func benchCaller(b *testing.B, caller *Caller, remote proto.NodeID) {
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := caller.CallNode(remote, "Count.Add", &AddReq{Delta: 1}, &AddResp{})
			if err != nil {
				b.Error("call node failed: ", err)
			}
		}
	})
}

func benchPCaller(b *testing.B, caller *PersistentCaller) {
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := caller.Call("Count.Add", &AddReq{Delta: 1}, &AddResp{})
			if err != nil {
				b.Error("call node failed: ", err)
			}
		}
	})
}

func testCaller(c C, wg *sync.WaitGroup, pool NOClientPool, remote proto.NodeID, con int, quest int) {
	defer wg.Done()
	var finished int32
	caller := NewCallerWithPool(pool)
	iwg := &sync.WaitGroup{}
	defer iwg.Wait()
	for i := 0; i < con; i++ {
		iwg.Add(1)
		go func(c C) {
			defer iwg.Done()
			for atomic.AddInt32(&finished, 1) < int32(quest) {
				err := caller.CallNode(remote, "Count.Add", &AddReq{Delta: 1}, &AddResp{})
				c.So(err, ShouldBeNil)
			}
		}(c)
	}
}

func testPCaller(c C, wg *sync.WaitGroup, pool NOClientPool, remote proto.NodeID, con int, quest int) {
	defer wg.Done()
	var finished int32
	caller := NewPersistentCallerWithPool(pool, remote)
	defer caller.Close()
	iwg := &sync.WaitGroup{}
	defer iwg.Wait()
	for i := 0; i < con; i++ {
		iwg.Add(1)
		go func(c C) {
			defer iwg.Done()
			for atomic.AddInt32(&finished, 1) < int32(quest) {
				err := caller.Call("Count.Add", &AddReq{Delta: 1}, &AddResp{})
				c.So(err, ShouldBeNil)
			}
		}(c)
	}
}

func BenchmarkRPCComponents(b *testing.B) {
	nodes, stop, err := setupEnvironment(1)
	if err != nil {
		b.Fatal("failed to setup servers")
	}
	defer stop()

	b.Run("CallerWithPool", func(b *testing.B) {
		pool := &ClientPool{}
		defer func() { _ = pool.Close() }()
		benchCaller(b, NewCallerWithPool(pool), nodes[0].ID)
	})
	b.Run("CallerWithoutPool", func(b *testing.B) {
		benchCaller(b, NewCallerWithPool(&nilPool{}), nodes[0].ID)
	})
	b.Run("PCaller", func(b *testing.B) {
		pool := &ClientPool{}
		defer func() { _ = pool.Close() }()
		benchPCaller(b, NewPersistentCallerWithPool(pool, nodes[0].ID))
	})
}

func TestRPCComponents(t *testing.T) {
	Convey("Setup a single server for pool test", t, func(c C) {
		nodes, stop, err := setupEnvironment(1)
		So(err, ShouldBeNil)
		defer stop()

		target := nodes[0].ID
		pool := &ClientPool{}
		defer func() { _ = pool.Close() }()

		So(pool.Len(), ShouldEqual, 0)
		cli1, err := pool.GetEx(target, true)
		So(err, ShouldBeNil)
		_ = cli1.Close()
		So(pool.Len(), ShouldEqual, 0)

		cli2, err := pool.GetEx(target, false)
		So(err, ShouldBeNil)
		_ = cli2.Close()
		So(pool.Len(), ShouldEqual, 1)

		So(pool.Len(), ShouldEqual, 1)
		pool.Remove(target)
		So(pool.Len(), ShouldEqual, 0)

		/*
			// Test broken pipe
			cli3, err := pool.GetEx(target, false)
			So(err, ShouldBeNil)
			err = cli3.Call("Count.Add", &AddReq{Delta: 1}, &AddResp{})
			So(err, ShouldBeNil)
			stop() // shutdown server, should produce a broken pipe error on cli3
			err = cli3.Call("Count.Add", &AddReq{Delta: 1}, &AddResp{})
			So(err, ShouldNotBeNil)
			_ = cli3.Close() // should not return a broken client to pool
			So(pool.Len(), ShouldEqual, 0)
		*/
	})

	Convey("Setup servers", t, func(c C) {
		nodes, stop, err := setupEnvironment(10)
		So(err, ShouldBeNil)
		defer stop()

		var (
			totalQuest    = 10000 // of each server
			callerCount   = 10    // of each server
			callerConcurr = 10    // of each caller
		)

		Convey("Test RCP call with Caller", func(c C) {
			pool := &ClientPool{}
			defer func() {
				_ = pool.Close()
				So(pool.Len(), ShouldEqual, 0)
			}()

			wg := &sync.WaitGroup{}
			for _, v := range nodes {
				for i := 0; i < callerCount; i++ {
					wg.Add(1)
					go testCaller(
						c, wg, pool, v.ID, callerConcurr, totalQuest/(callerCount*callerConcurr))
				}
			}
			wg.Wait()
			So(pool.Len(), ShouldBeGreaterThan, 0)
		})

		Convey("Test RCP call with PCaller", func(c C) {
			pool := &ClientPool{}
			defer func() {
				_ = pool.Close()
				So(pool.Len(), ShouldEqual, 0)
			}()

			wg := &sync.WaitGroup{}
			for _, v := range nodes {
				for i := 0; i < callerCount; i++ {
					wg.Add(1)
					go testPCaller(
						c, wg, pool, v.ID, callerConcurr, totalQuest/(callerCount*callerConcurr))
				}
			}
			wg.Wait()
			So(pool.Len(), ShouldBeGreaterThan, 0)
		})
	})
}
