/*
 * Copyright 2019 The CovenantSQL Authors.
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

package naconn

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/utils"
)

var (
	tempDir string

	workingRoot = utils.GetProjectSrcDir()
	confFile    = filepath.Join(workingRoot, "test/node_c/config.yaml")
	privateKey  = filepath.Join(workingRoot, "test/node_c/private.key")
)

type simpleResolver struct {
	nodes sync.Map // *proto.RawNodeID -> *proto.Node
}

func (r *simpleResolver) registerNode(node *proto.Node) {
	key := *(node.ID.ToRawNodeID())
	r.nodes.Store(key, node)
}

func (r *simpleResolver) Resolve(id *proto.RawNodeID) (addr string, err error) {
	var node *proto.Node
	if node, err = r.ResolveEx(id); err != nil {
		return
	}
	addr = node.Addr
	return
}

func (r *simpleResolver) ResolveEx(id *proto.RawNodeID) (*proto.Node, error) {
	if node, ok := r.nodes.Load(*id); ok {
		return node.(*proto.Node), nil
	}
	return nil, fmt.Errorf("not found")
}

func TestNAConn(t *testing.T) {
	Convey("Test simple NAConn", t, func(c C) {
		l, err := net.Listen("tcp", "localhost:0")
		So(err, ShouldBeNil)
		defer func() { _ = l.Close() }()
		// Register node
		resolver := &simpleResolver{}
		nodeinfo := thisNode()
		So(nodeinfo, ShouldNotBeNil)
		resolver.registerNode(&proto.Node{
			Addr:      l.Addr().String(),
			ID:        nodeinfo.ID,
			PublicKey: nodeinfo.PublicKey,
			Nonce:     nodeinfo.Nonce,
		})
		// Register resolver
		RegisterResolver(resolver)
		// Serve
		rounds := 100
		message := [1024]byte{}
		rand.Read(message[:])
		wg := &sync.WaitGroup{}
		wg.Add(1)
		go func(c C) {
			defer wg.Done()
			for i := 0; i < rounds; i++ {
				conn, err := l.Accept()
				c.So(err, ShouldBeNil)
				wg.Add(1)
				go func(c C, conn net.Conn) {
					defer wg.Done()
					naconn, err := Accept(conn)
					c.So(err, ShouldBeNil)
					defer func() { _ = naconn.Close() }()
					t.Logf("accept conn %s <- %s", naconn.LocalAddr(), naconn.RemoteAddr())
					buffer, err := ioutil.ReadAll(naconn)
					c.So(err, ShouldBeNil)
					c.So(buffer, ShouldResemble, message[:])
				}(c, conn)
			}
		}(c)
		// Send request
		for i := 0; i < rounds; i++ {
			wg.Add(1)
			go func(c C) {
				defer wg.Done()
				var (
					conn net.Conn
					err  error
				)
				if rand.Int()%2 == 0 {
					conn, err = Dial(nodeinfo.ID)
				} else {
					conn, err = DialEx(nodeinfo.ID, true)
				}
				c.So(err, ShouldBeNil)
				defer func() { _ = conn.Close() }()
				t.Logf("dial conn %s -> %s", conn.LocalAddr(), conn.RemoteAddr())
				n, err := conn.Write(message[:])
				c.So(err, ShouldBeNil)
				c.So(n, ShouldEqual, len(message))
			}(c)
		}
		wg.Wait()
	})
}

func thisNode() *proto.Node {
	if conf.GConf != nil {
		for _, node := range conf.GConf.KnownNodes {
			if node.ID == conf.GConf.ThisNodeID {
				return &node
			}
		}
	}
	return nil
}

func setup() {
	rand.Seed(time.Now().UnixNano())

	var err error
	if tempDir, err = ioutil.TempDir("", "covenantsql"); err != nil {
		panic(err)
	}
	if conf.GConf, err = conf.LoadConfig(confFile); err != nil {
		panic(err)
	}
	if err = kms.InitLocalKeyPair(privateKey, []byte{}); err != nil {
		panic(err)
	}
	route.InitKMS(filepath.Join(tempDir, "public.keystore"))
}

func teardown() {
	if err := os.RemoveAll(tempDir); err != nil {
		panic(err)
	}
}

func TestMain(m *testing.M) {
	os.Exit(func() int {
		setup()
		defer teardown()
		return m.Run()
	}())
}
