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
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/naconn"
	"github.com/CovenantSQL/CovenantSQL/pow/cpuminer"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

var (
	tempDir string

	workingRoot = utils.GetProjectSrcDir()
	confFile    = filepath.Join(workingRoot, "test/node_c/config.yaml")
	privateKey  = filepath.Join(workingRoot, "test/node_c/private.key")
)

type simpleResolver struct {
	nodes sync.Map // proto.RawNodeID -> *proto.Node
}

func (r *simpleResolver) registerNode(node *proto.Node) {
	key := *(node.ID.ToRawNodeID())
	log.WithFields(log.Fields{"node": node}).Debug("register node")
	r.nodes.Store(key, node)
}

func (r *simpleResolver) deleteNode(key proto.RawNodeID) {
	r.nodes.Delete(key)
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

var defaultResolver = &simpleResolver{}

type nilPool struct{} // mocks the pool interface with a direct dialer

func (p *nilPool) Get(id proto.NodeID) (Client, error) {
	return p.GetEx(id, false)
}

func (p *nilPool) GetEx(id proto.NodeID, isAnonymous bool) (Client, error) {
	conn, err := Dial(id)
	if err != nil {
		return nil, err
	}
	return NewClient(conn), err
}

func (p *nilPool) Close() error { return nil }

// CountService is a simple count service for testing.
type CountService struct {
	host  proto.NodeID
	Count int32
}

type AddReq struct {
	proto.Envelope
	Delta int32
}

type AddResp struct {
	proto.Envelope
	Count int32
}

func (s *CountService) Add(req *AddReq, resp *AddResp) error {
	resp.SetNodeID(req.NodeID)
	resp.Count = atomic.AddInt32(&s.Count, req.Delta)
	return nil
}

// createLocalNodes uses the cpu miner to mine node IDs for the local public key.
func createLocalNodes(diff int, num int) (nodes []*proto.Node, err error) {
	pub, err := kms.GetLocalPublicKey()
	if err != nil {
		return
	}
	nodes = make([]*proto.Node, num)

	miner := cpuminer.NewCPUMiner(nil)
	nCh := make(chan cpuminer.NonceInfo)
	defer close(nCh)
	block := cpuminer.MiningBlock{
		Data:      pub.Serialize(),
		NonceChan: nCh,
	}
	next := cpuminer.Uint256{}
	wg := &sync.WaitGroup{}

	for i := 0; i < num; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = miner.ComputeBlockNonce(block, next, diff)
		}()

		n := <-nCh
		nodes[i] = &proto.Node{
			ID:        proto.NodeID(n.Hash.String()),
			PublicKey: pub,
			Nonce:     n.Nonce,
		}

		next = n.Nonce
		next.Inc()
		wg.Wait()
	}
	return
}

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

func setupServers(nodes []*proto.Node, f AcceptConn) (stop func(), err error) {
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
			server.WithAcceptConnFunc(f).Serve()
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

func setupEnvironment(n int, f AcceptConn) ([]*proto.Node, func(), error) {
	nodes, err := createLocalNodes(10, n)
	if err != nil {
		return nil, nil, err
	}
	stop, err := setupServers(nodes, f)
	if err != nil {
		return nil, nil, err
	}
	return nodes, stop, nil
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
	naconn.RegisterResolver(defaultResolver)
	if node := thisNode(); node != nil {
		defaultResolver.registerNode(node)
	}

	log.SetLevel(log.DebugLevel)
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
