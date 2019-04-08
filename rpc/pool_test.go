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
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

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
	originLevel := log.GetLevel()
	defer log.SetLevel(originLevel)
	log.SetLevel(log.FatalLevel)
	nodes, stop, err := setupEnvironment(1, AcceptNAConn)
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
		nodes, stop, err := setupEnvironment(1, AcceptNAConn)
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

		Convey("Test pool functionality", func() {
			var resp AddResp
			// Test broken pipe
			cli3, err := pool.GetEx(target, false)
			So(err, ShouldBeNil)
			err = cli3.Call("Count.Add", &AddReq{Delta: 1}, &resp)
			So(err, ShouldBeNil)
			_ = cli3.(*pooledClient).Client.Close() // close underlying client, should produce an error upon any following Call
			err = cli3.Call("Count.Add", &AddReq{Delta: 1}, &resp)
			So(err, ShouldNotBeNil)
			_ = cli3.Close() // should not return a broken client to pool
			So(pool.Len(), ShouldEqual, 0)
		})

		Convey("Test caller functionality", func() {
			pool := &ClientPool{}
			defer func() { _ = pool.Close() }()
			caller := NewPersistentCallerWithPool(pool, target)
			defer caller.Close()

			// Set a bad client with raw tcp
			rawconn, err := net.Dial("tcp", nodes[0].Addr)
			So(err, ShouldBeNil)
			caller.client = NewClient(rawconn)
			err = caller.Call("Count.Add", &AddReq{Delta: 1}, &AddResp{}) // should try reconnect, and return err
			So(err, ShouldNotBeNil)
			err = caller.Call("Count.Add", &AddReq{Delta: 1}, &AddResp{}) // should be no err
			So(err, ShouldBeNil)
		})
	})

	Convey("Setup servers", t, func(c C) {
		nodes, stop, err := setupEnvironment(10, AcceptNAConn)
		So(err, ShouldBeNil)
		defer stop()

		var (
			totalQuest    = 1000 // of each server
			callerCount   = 10   // of each server
			callerConcurr = 10   // of each caller
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

func TestRecordRPCCost(t *testing.T) {
	Convey("Bug: bad critical section for multiple values", t, func(c C) {
		var (
			start      = time.Now()
			rounds     = 1000
			concurrent = 10
			wg         = &sync.WaitGroup{}
			body       = func(i int) {
				defer func() {
					c.So(recover(), ShouldBeNil)
					wg.Done()
				}()
				recordRPCCost(start, fmt.Sprintf("M%d", i), nil)
			}
		)
		for i := 0; i < rounds; i++ {
			for j := 0; j < concurrent; j++ {
				wg.Add(1)
				go body(i)
			}
			wg.Wait()
		}
	})
}
