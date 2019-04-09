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

package mux

import (
	"net"
	"path/filepath"
	"sync"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

const (
	concurrency = conf.MaxRPCMuxPoolPhysicalConnection + 1
	packetCount = 100
)

var (
	baseDir        = utils.GetProjectSrcDir()
	testWorkingDir = FJ(baseDir, "./test/")
	logDir         = FJ(testWorkingDir, "./log/")
)

var FJ = filepath.Join

// withTCPDialer overwrites the node-oriented dialer of rpc package, skips the naconn resolver
// and implies a default resolving method: addr := string(remote node ID).
func withTCPDialer() func() {
	dial := rpc.Dial
	dialEx := rpc.DialEx

	rpc.Dial = func(remote proto.NodeID) (net.Conn, error) {
		return net.Dial("tcp", string(remote))
	}

	rpc.DialEx = func(remote proto.NodeID, isAnonymous bool) (net.Conn, error) {
		return net.Dial("tcp", string(remote))
	}

	// recover func
	return func() {
		rpc.Dial = dial
		rpc.DialEx = dialEx
	}
}

func BenchmarkSessionPool_Get(b *testing.B) {
	Convey("session pool", b, func(c C) {
		log.SetLevel(log.FatalLevel)
		p := &SessionPool{
			sessions: make(map[proto.NodeID]*Session),
		}
		defer withTCPDialer()()
		defer func() { _ = p.Close() }()

		l, err := net.Listen("tcp", ":0")
		if err != nil {
			log.Fatal(err)
		}
		server, err := NewServerWithService(ServiceMap{"Test": NewTestService()})
		server.SetListener(l)
		go server.WithAcceptConnFunc(rpc.AcceptRawConn).Serve()
		wg := &sync.WaitGroup{}
		b.ResetTimer()

		wg.Add(concurrency)
		for i := 0; i < concurrency; i++ {
			go func(c2 C, n int) {
				defer wg.Done()
				client, err := p.Get(proto.NodeID(l.Addr().String()))
				defer func() { _ = client.Close() }()
				if err != nil {
					log.Errorf("get session failed: %s", err)
					return
				}
				c2.So(err, ShouldBeNil)
				err = client.Call("Test.IncCounter", &TestReq{Step: 1}, &TestRep{})
				c2.So(err, ShouldBeNil)
			}(c, b.N)
		}
		wg.Wait()
	})
}

func TestNewSessionPool(t *testing.T) {
	Convey("session pool", t, func(c C) {
		log.SetLevel(log.FatalLevel)

		// setup pool
		p := &SessionPool{
			sessions: make(map[proto.NodeID]*Session),
		}
		defer withTCPDialer()()

		// setup server
		l, err := net.Listen("tcp", ":0")
		if err != nil {
			log.Fatal(err)
		}
		server, err := NewServerWithService(ServiceMap{"Test": NewTestService()})
		server.SetListener(l)
		go server.WithAcceptConnFunc(rpc.AcceptRawConn).Serve()

		wg := &sync.WaitGroup{}
		wg.Add(concurrency)
		for i := 0; i < concurrency; i++ {
			go func(c2 C, n int) {
				defer wg.Done()
				client, err := p.Get(proto.NodeID(l.Addr().String()))
				defer func() { _ = client.Close() }()
				if err != nil {
					log.Errorf("get session failed: %s", err)
					return
				}
				c2.So(err, ShouldBeNil)
				err = client.Call("Test.IncCounter", &TestReq{Step: 1}, &TestRep{})
				c2.So(err, ShouldBeNil)
			}(c, packetCount)
		}

		wg.Wait()
		So(p.Len(), ShouldEqual, conf.MaxRPCMuxPoolPhysicalConnection)

		// setup server
		l2, err := net.Listen("tcp", ":0")
		if err != nil {
			log.Fatal(err)
		}
		server2, err := NewServerWithService(ServiceMap{"Test": NewTestService()})
		server2.SetListener(l2)
		go server2.WithAcceptConnFunc(rpc.AcceptRawConn).Serve()

		_, err = p.Get(proto.NodeID(l2.Addr().String()))
		So(err, ShouldBeNil)
		So(p.Len(), ShouldEqual, conf.MaxRPCMuxPoolPhysicalConnection+1)

		wg2 := &sync.WaitGroup{}
		wg2.Add(concurrency)
		for i := 0; i < concurrency; i++ {
			go func(c2 C, n int) {
				// Open a new stream
				// Stream implements net.Conn
				defer wg2.Done()
				client, err := p.Get(proto.NodeID(l2.Addr().String()))
				defer func() { _ = client.Close() }()
				if err != nil {
					log.Errorf("get session failed: %s", err)
					return
				}
				c2.So(err, ShouldBeNil)
				err = client.Call("Test.IncCounter", &TestReq{Step: 1}, &TestRep{})
				c2.So(err, ShouldBeNil)
			}(c, packetCount)
		}

		wg2.Wait()
		So(p.Len(), ShouldEqual, conf.MaxRPCMuxPoolPhysicalConnection*2)

		p.Remove(proto.NodeID(l.Addr().String()))
		So(p.Len(), ShouldEqual, conf.MaxRPCMuxPoolPhysicalConnection)

		_ = p.Close()
		So(p.Len(), ShouldEqual, 0)
	})

	Convey("session pool get instance", t, func(c C) {
		So(GetSessionPoolInstance(), ShouldNotBeNil)
		So(GetSessionPoolInstance() == GetSessionPoolInstance(), ShouldBeTrue)
	})
}
