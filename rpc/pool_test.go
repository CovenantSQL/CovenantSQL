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
	"net"
	"path/filepath"
	"sync"
	"testing"

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	. "github.com/smartystreets/goconvey/convey"
	mux "github.com/xtaci/smux"
)

const (
	localAddr   = "127.0.0.1:4444"
	localAddr2  = "127.0.0.1:4445"
	concurrency = conf.MaxRPCPoolPhysicalConnection + 1
	packetCount = 100
)

var (
	baseDir        = utils.GetProjectSrcDir()
	testWorkingDir = FJ(baseDir, "./test/")
	logDir         = FJ(testWorkingDir, "./log/")
)

var FJ = filepath.Join

func server(c C, localAddr string, n int) error {
	// Accept a TCP connection
	listener, err := net.Listen("tcp", localAddr)
	go func() {
		for i := 0; i < concurrency; i++ {
			go func() {
				conn, err := listener.Accept()
				c.So(err, ShouldBeNil)

				// Setup server side of mux
				log.Println("creating server session")
				session, err := mux.Server(conn, nil)
				c.So(err, ShouldBeNil)

				for i := 0; i < concurrency; i++ {
					// Accept a stream
					//c.So(err, ShouldBeNil)
					// Stream implements net.Conn
					// Listen for a message
					//c.So(string(buf1), ShouldEqual, "ping")
					log.Println("accepting stream")
					stream, err := session.AcceptStream()
					if err == nil {
						buf1 := make([]byte, 4)
						for i := 0; i < n; {
							n, err := stream.Read(buf1)
							if n == 4 && err == nil {
								i++
								c.So(string(buf1), ShouldEqual, "ping")
							}
						}
						log.Debugf("buf#%d read done", i)
					}
				}
			}()
		}
	}()
	return err
}

func BenchmarkSessionPool_Get(b *testing.B) {
	Convey("session pool", b, func(c C) {
		log.SetLevel(log.FatalLevel)
		p := newSessionPool(func(nodeID proto.NodeID) (net.Conn, error) {
			log.Debugf("creating new connection to %s", nodeID)
			return net.Dial("tcp", string(nodeID))
		})

		wg := &sync.WaitGroup{}

		server(c, localAddr, b.N)
		b.ResetTimer()
		wg.Add(concurrency)
		for i := 0; i < concurrency; i++ {
			go func(c2 C, n int) {
				// Open a new stream
				// Stream implements net.Conn
				defer wg.Done()
				stream, err := p.Get(proto.NodeID(localAddr))
				c2.So(err, ShouldBeNil)
				for i := 0; i < n; {
					n, err := stream.Write([]byte("ping"))
					if n == 4 && err == nil {
						i++
					}
				}
			}(c, b.N)
		}
		wg.Wait()
	})
}

func TestNewSessionPool(t *testing.T) {
	Convey("session pool", t, func(c C) {
		log.SetLevel(log.FatalLevel)
		p := newSessionPool(func(nodeID proto.NodeID) (net.Conn, error) {
			log.Debugf("creating new connection to %s", nodeID)
			return net.Dial("tcp", string(nodeID))
		})

		wg := &sync.WaitGroup{}

		server(c, localAddr, packetCount)
		p.Get(proto.NodeID(localAddr))

		wg.Add(concurrency)
		for i := 0; i < concurrency; i++ {
			go func(c2 C, n int) {
				// Open a new stream
				// Stream implements net.Conn
				defer wg.Done()
				stream, err := p.Get(proto.NodeID(localAddr))
				if err != nil {
					log.Errorf("get session failed: %s", err)
					return
				}
				c2.So(err, ShouldBeNil)
				for i := 0; i < n; {
					n, err := stream.Write([]byte("ping"))
					if n == 4 && err == nil {
						i++
					}
				}
			}(c, packetCount)
		}

		wg.Wait()
		So(p.Len(), ShouldEqual, conf.MaxRPCPoolPhysicalConnection)

		server(c, localAddr2, packetCount)
		_, err := p.Get(proto.NodeID(localAddr2))
		So(err, ShouldBeNil)
		So(p.Len(), ShouldEqual, conf.MaxRPCPoolPhysicalConnection+1)

		wg2 := &sync.WaitGroup{}
		wg2.Add(concurrency)
		for i := 0; i < concurrency; i++ {
			go func(c2 C, n int) {
				// Open a new stream
				// Stream implements net.Conn
				defer wg2.Done()
				stream, err := p.Get(proto.NodeID(localAddr2))
				if err != nil {
					log.Errorf("get session failed: %s", err)
					return
				}
				c2.So(err, ShouldBeNil)
				for i := 0; i < n; {
					n, err := stream.Write([]byte("ping"))
					if n == 4 && err == nil {
						i++
					}
				}
			}(c, packetCount)
		}

		wg2.Wait()
		So(p.Len(), ShouldEqual, conf.MaxRPCPoolPhysicalConnection*2)

		p.Remove(proto.NodeID(localAddr2))
		So(p.Len(), ShouldEqual, conf.MaxRPCPoolPhysicalConnection)

		p.Close()
		So(p.Len(), ShouldEqual, 0)
	})

	Convey("session pool get instance", t, func(c C) {
		So(GetSessionPoolInstance(), ShouldNotBeNil)
		So(GetSessionPoolInstance() == GetSessionPoolInstance(), ShouldBeTrue)
	})
}
