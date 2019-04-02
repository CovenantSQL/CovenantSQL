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
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

type CountService struct {
	Counrt int32
}

type AddReq struct {
	Delta int32
}

type AddResp struct {
	Count int32
}

func (s *CountService) Add(req *AddReq, resp *AddResp) error {
	resp.Count = atomic.AddInt32(&s.Counrt, req.Delta)
	log.WithFields(log.Fields{"result": resp.Count}).Debug("call Add")
	return nil
}

func setupServer(node *proto.Node) (server *Server, err error) {
	if server, err = NewServerWithService(ServiceMap{"Count": &CountService{}}); err != nil {
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
		for _, v := range servers {
			v.Stop()
		}
		wg.Wait()
	}, nil
}

func callWithCaller(c C, pool NOConnPool, remote proto.NodeID, reqs int) {
	caller := NewCallerWithPool(pool)
	for i := 0; i < reqs; i++ {
		req := &AddReq{Delta: 1}
		resp := &AddResp{}
		err := caller.CallNode(remote, "Count.Add", req, resp)
		c.So(err, ShouldBeNil)
	}
}

func TestRPCComponents(t *testing.T) {
	Convey("Setup servers", t, func(c C) {
		svCount := 10
		nodes, err := createLocalNodes(10, svCount)
		So(err, ShouldBeNil)
		So(len(nodes), ShouldEqual, svCount)

		stop, err := setupServers(nodes)
		So(err, ShouldBeNil)
		defer stop()

		cliCount := 100
		reqCount := 1000
		pool := &ConnPool{}
		for _, v := range nodes {
			for i := 0; i < cliCount; i++ {
				callWithCaller(c, pool, v.ID, reqCount)
			}
		}
	})
}
