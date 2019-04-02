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
	"testing"
	"time"

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/noconn"
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
	noconn.RegisterResolver(defaultResolver)
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
