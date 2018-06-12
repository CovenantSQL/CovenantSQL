/*
 * Copyright 2018 The ThunderDB Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the “License”);
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an “AS IS” BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package transport

import (
	"context"
	"sync"
	"testing"

	"io/ioutil"

	"github.com/jordwest/mock-conn"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/thunderdb/ThunderDB/proto"
)

type TestConn struct {
	*mock_conn.End
	peerNodeID proto.NodeID
}

type TestStreamRouter struct {
	sync.Mutex
	streamMap map[proto.NodeID]*TestStream
}

type TestStream struct {
	nodeID proto.NodeID
	router *TestStreamRouter
	queue  chan *TestConn
}

func NewTestStreamRouter() *TestStreamRouter {
	return &TestStreamRouter{
		streamMap: make(map[proto.NodeID]*TestStream),
	}
}

func NewTestStream(nodeID proto.NodeID, router *TestStreamRouter) *TestStream {
	return &TestStream{
		nodeID: nodeID,
		router: router,
		queue:  make(chan *TestConn),
	}
}

func NewSocketPair(fromNode proto.NodeID, toNode proto.NodeID) (clientConn *TestConn, serverConn *TestConn) {
	conn := mock_conn.NewConn()
	clientConn = NewTestConn(conn.Server, toNode)
	serverConn = NewTestConn(conn.Client, fromNode)
	return
}

func NewTestConn(endpoint *mock_conn.End, peerNodeID proto.NodeID) *TestConn {
	return &TestConn{
		End:        endpoint,
		peerNodeID: peerNodeID,
	}
}

func (r *TestStreamRouter) Get(id proto.NodeID) *TestStream {
	r.Lock()
	defer r.Unlock()

	if _, ok := r.streamMap[id]; !ok {
		r.streamMap[id] = NewTestStream(id, r)
	}

	return r.streamMap[id]
}

func (c *TestConn) GetPeerNodeID() proto.NodeID {
	return c.peerNodeID
}

func (s *TestStream) Accept() (conn ConnWithPeerNodeID, err error) {
	select {
	case conn := <-s.queue:
		return conn, nil
	}
}

func (s *TestStream) Dial(ctx context.Context, nodeID proto.NodeID) (conn ConnWithPeerNodeID, err error) {
	clientConn, serverConn := NewSocketPair(s.nodeID, nodeID)

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case s.router.Get(nodeID).queue <- serverConn:
	}

	return clientConn, nil
}

func TestConnPair(t *testing.T) {
	Convey("test transport", t, FailureContinues, func(c C) {
		router := NewTestStreamRouter()
		stream1 := router.Get("id1")
		stream2 := router.Get("id2")

		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			defer wg.Done()
			var err error
			clientConn, err := stream1.Dial(context.Background(), "id2")
			c.So(err, ShouldBeNil)
			_, err = clientConn.Write([]byte("test"))
			c.So(err, ShouldBeNil)
			clientConn.Close()
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			var err error
			serverConn, err := stream2.Accept()
			c.So(err, ShouldBeNil)
			buffer, err := ioutil.ReadAll(serverConn)
			c.So(err, ShouldBeNil)
			c.So(buffer, ShouldResemble, []byte("test"))
		}()

		wg.Wait()
	})
}

func TestTransport(t *testing.T) {
	Convey("test transport", t, FailureContinues, func(c C) {
		router := NewTestStreamRouter()
		stream1 := router.Get("id1")
		stream2 := router.Get("id2")
		config1 := NewConfig("id1", stream1)
		config2 := NewConfig("id2", stream2)
		t1 := NewTransport(config1)
		t2 := NewTransport(config2)

		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := t1.Request(context.Background(), "id2", "test method", "test request")
			c.So(err, ShouldBeNil)
			c.So(res, ShouldResemble, "test response")
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case req := <-t2.Process():
				c.So(req.GetRequest(), ShouldResemble, "test request")
				c.So(req.GetMethod(), ShouldEqual, "test method")
				c.So(req.GetNodeID(), ShouldEqual, proto.NodeID("id1"))
				req.SendResponse("test response")
			}
		}()

		t1.Close()
		t2.Close()

		wg.Wait()
	})
}
