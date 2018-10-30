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

package kayak

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/twopc"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	. "github.com/smartystreets/goconvey/convey"
)

// common mocks
type MockTransportRouter struct {
	reqSeq        uint64
	transports    map[proto.NodeID]*MockTransport
	transportLock sync.Mutex
}

type MockTransport struct {
	nodeID    proto.NodeID
	router    *MockTransportRouter
	queue     chan Request
	waitQueue chan *MockResponse
	giveUp    map[uint64]bool
}

type MockRequest struct {
	transport *MockTransport
	ctx       context.Context
	RequestID uint64
	NodeID    proto.NodeID
	Method    string
	Log       *Log
}

type MockResponse struct {
	ResponseID uint64
	Data       []byte
	Error      error
}

type MockTwoPCWorker struct {
	nodeID proto.NodeID
	state  string
	data   int64
	total  int64
}

var (
	_ twopc.Worker = &MockTwoPCWorker{}
)

func (m *MockTransportRouter) getTransport(nodeID proto.NodeID) *MockTransport {
	m.transportLock.Lock()
	defer m.transportLock.Unlock()

	if _, ok := m.transports[nodeID]; !ok {
		m.transports[nodeID] = &MockTransport{
			nodeID:    nodeID,
			router:    m,
			queue:     make(chan Request, 1000),
			waitQueue: make(chan *MockResponse, 1000),
			giveUp:    make(map[uint64]bool),
		}
	}

	return m.transports[nodeID]
}

func (m *MockTransportRouter) ResetTransport(nodeID proto.NodeID) {
	m.transportLock.Lock()
	defer m.transportLock.Unlock()

	if _, ok := m.transports[nodeID]; ok {
		// reset
		delete(m.transports, nodeID)
	}
}

func (m *MockTransportRouter) ResetAll() {
	m.transportLock.Lock()
	defer m.transportLock.Unlock()

	m.transports = make(map[proto.NodeID]*MockTransport)
}

func (m *MockTransportRouter) getReqID() uint64 {
	return atomic.AddUint64(&m.reqSeq, 1)
}

func (m *MockTransport) Init() error {
	return nil
}

func (m *MockTransport) Request(ctx context.Context, nodeID proto.NodeID, method string, log *Log) ([]byte, error) {
	return m.router.getTransport(nodeID).sendRequest(&MockRequest{
		RequestID: m.router.getReqID(),
		NodeID:    m.nodeID,
		Method:    method,
		Log:       log,
		ctx:       ctx,
	})
}

func (m *MockTransport) Process() <-chan Request {
	return m.queue
}

func (m *MockTransport) Shutdown() error {
	return nil
}

func (m *MockTransport) sendRequest(req Request) ([]byte, error) {
	r := req.(*MockRequest)
	r.transport = m

	if log.GetLevel() >= log.DebugLevel {
		fmt.Println()
	}
	log.Debugf("[%v] [%v] -> [%v] request %v", r.RequestID, r.NodeID, req.GetPeerNodeID(), r.GetLog())
	m.queue <- r

	for {
		select {
		case <-r.ctx.Done():
			// deadline reached
			log.Debugf("[%v] [%v] -> [%v] request timeout",
				r.RequestID, r.NodeID, req.GetPeerNodeID())
			m.giveUp[r.RequestID] = true
			return nil, r.ctx.Err()
		case res := <-m.waitQueue:
			if res.ResponseID != r.RequestID {
				// put back to queue
				if !m.giveUp[res.ResponseID] {
					m.waitQueue <- res
				} else {
					delete(m.giveUp, res.ResponseID)
				}
			} else {
				log.Debugf("[%v] [%v] -> [%v] response %v: %v",
					r.RequestID, req.GetPeerNodeID(), r.NodeID, res.Data, res.Error)
				return res.Data, res.Error
			}
		}
	}
}

func (m *MockRequest) GetPeerNodeID() proto.NodeID {
	return m.NodeID
}

func (m *MockRequest) GetMethod() string {
	return m.Method
}

func (m *MockRequest) GetLog() *Log {
	return m.Log
}

func (m *MockRequest) SendResponse(v []byte, err error) error {
	m.transport.waitQueue <- &MockResponse{
		ResponseID: m.RequestID,
		Data:       v,
		Error:      err,
	}

	return nil
}

func (w *MockTwoPCWorker) Prepare(ctx context.Context, wb twopc.WriteBatch) error {
	// test prepare
	if w.state != "" {
		return errors.New("invalid state")
	}

	value, ok := wb.(int64)
	if !ok {
		return errors.New("invalid data")
	}

	w.state = "prepared"
	w.data = value

	return nil
}

func (w *MockTwoPCWorker) Commit(ctx context.Context, wb twopc.WriteBatch) (interface{}, error) {
	// test commit
	if w.state != "prepared" {
		return nil, errors.New("invalid state")
	}

	if !reflect.DeepEqual(wb, w.data) {
		return nil, errors.New("commit data not same as last")
	}

	w.total += w.data
	w.state = ""

	return nil, nil
}

func (w *MockTwoPCWorker) Rollback(ctx context.Context, wb twopc.WriteBatch) error {
	// test rollback
	if w.state != "prepared" {
		return errors.New("invalid state")
	}

	if !reflect.DeepEqual(wb, w.data) {
		return errors.New("commit data not same as last")
	}

	w.data = 0
	w.state = ""

	return nil
}

func (w *MockTwoPCWorker) GetTotal() int64 {
	return w.total
}

func (w *MockTwoPCWorker) GetState() string {
	return w.state
}

type CallCollector struct {
	l         sync.Mutex
	callOrder []string
}

func (c *CallCollector) Append(call string) {
	c.l.Lock()
	defer c.l.Unlock()
	c.callOrder = append(c.callOrder, call)
}

func (c *CallCollector) Get() []string {
	c.l.Lock()
	defer c.l.Unlock()
	return c.callOrder[:]
}

func (c *CallCollector) Reset() {
	c.l.Lock()
	defer c.l.Unlock()
	c.callOrder = c.callOrder[:0]
}

func testPeersFixture(term uint64, servers []*Server) *Peers {
	testPriv := []byte{
		0xea, 0xf0, 0x2c, 0xa3, 0x48, 0xc5, 0x24, 0xe6,
		0x39, 0x26, 0x55, 0xba, 0x4d, 0x29, 0x60, 0x3c,
		0xd1, 0xa7, 0x34, 0x7d, 0x9d, 0x65, 0xcf, 0xe9,
		0x3c, 0xe1, 0xeb, 0xff, 0xdc, 0xa2, 0x26, 0x94,
	}
	privKey, pubKey := asymmetric.PrivKeyFromBytes(testPriv)

	newServers := make([]*Server, 0, len(servers))
	var leaderNode *Server

	for _, s := range servers {
		newS := &Server{
			Role:   s.Role,
			ID:     s.ID,
			PubKey: pubKey,
		}
		newServers = append(newServers, newS)
		if newS.Role == proto.Leader {
			leaderNode = newS
		}
	}

	peers := &Peers{
		Term:    term,
		Leader:  leaderNode,
		Servers: servers,
		PubKey:  pubKey,
	}

	peers.Sign(privKey)

	return peers
}

func testLogFixture(data []byte) (log *Log) {
	log = &Log{
		Index: uint64(1),
		Term:  uint64(1),
		Data:  data,
	}

	log.ComputeHash()

	return
}

// test mock library itself
func TestMockTransport(t *testing.T) {
	Convey("test transport with request timeout", t, func() {
		mockRouter := &MockTransportRouter{
			transports: make(map[proto.NodeID]*MockTransport),
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*50)
		defer cancel()

		var err error
		var response []byte
		response, err = mockRouter.getTransport("a").Request(
			ctx, "b", "Test", testLogFixture([]byte("happy")))

		So(response, ShouldBeNil)
		So(err, ShouldNotBeNil)
	})

	Convey("test transport with successful request", t, func(c C) {
		mockRouter := &MockTransportRouter{
			transports: make(map[proto.NodeID]*MockTransport),
		}
		testLog := testLogFixture([]byte("happy"))
		var wg sync.WaitGroup

		wg.Add(1)

		go func() {
			defer wg.Done()
			select {
			case req := <-mockRouter.getTransport("d").Process():
				c.So(req.GetPeerNodeID(), ShouldEqual, proto.NodeID("c"))
				c.So(req.GetMethod(), ShouldEqual, "Test")
				c.So(req.GetLog(), ShouldResemble, testLog)
				req.SendResponse([]byte("happy too"), nil)
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			var err error
			var response []byte
			response, err = mockRouter.getTransport("c").Request(
				context.Background(), "d", "Test", testLog)

			c.So(err, ShouldBeNil)
			c.So(response, ShouldResemble, []byte("happy too"))
		}()

		wg.Wait()
	})

	Convey("test transport with concurrent request", t, FailureContinues, func(c C) {
		mockRouter := &MockTransportRouter{
			transports: make(map[proto.NodeID]*MockTransport),
		}
		testLog := testLogFixture([]byte("happy"))
		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			defer wg.Done()
			var err error
			var response []byte
			response, err = mockRouter.getTransport("e").Request(
				context.Background(), "g", "test1", testLog)

			c.So(err, ShouldBeNil)
			c.So(response, ShouldResemble, []byte("happy e test1"))
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			var err error
			var response []byte
			response, err = mockRouter.getTransport("f").Request(
				context.Background(), "g", "test2", testLog)

			c.So(err, ShouldBeNil)
			c.So(response, ShouldResemble, []byte("happy f test2"))
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()

			for i := 0; i < 2; i++ {
				select {
				case req := <-mockRouter.getTransport("g").Process():
					c.So(req.GetPeerNodeID(), ShouldBeIn, []proto.NodeID{"e", "f"})
					c.So(req.GetMethod(), ShouldBeIn, []string{"test1", "test2"})
					c.So(req.GetLog(), ShouldResemble, testLog)
					req.SendResponse([]byte(fmt.Sprintf("happy %s %s", req.GetPeerNodeID(), req.GetMethod())), nil)
				}
			}
		}()

		wg.Wait()
	})

	Convey("test transport with piped request", t, FailureContinues, func(c C) {
		mockRouter := &MockTransportRouter{
			transports: make(map[proto.NodeID]*MockTransport),
		}
		var wg sync.WaitGroup

		randReq := testLogFixture([]byte("happy"))
		randResp := make([]byte, 4)
		rand.Read(randResp)

		t.Logf("test with request %d, response %d", randReq, randResp)

		wg.Add(1)
		go func() {
			defer wg.Done()
			var err error
			var response []byte
			var req Request

			select {
			case req = <-mockRouter.getTransport("j").Process():
				c.So(req.GetPeerNodeID(), ShouldEqual, proto.NodeID("i"))
				c.So(req.GetMethod(), ShouldEqual, "pass1")
			}

			response, err = mockRouter.getTransport("j").Request(
				context.Background(), "k", "pass2", req.GetLog())

			c.So(err, ShouldBeNil)
			req.SendResponse(response, nil)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case req := <-mockRouter.getTransport("k").Process():
				c.So(req.GetPeerNodeID(), ShouldEqual, proto.NodeID("j"))
				c.So(req.GetMethod(), ShouldEqual, "pass2")
				c.So(req.GetLog(), ShouldResemble, randReq)
				req.SendResponse(randResp, nil)
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			var err error
			var response []byte

			response, err = mockRouter.getTransport("i").Request(
				context.Background(), "j", "pass1", randReq)

			c.So(err, ShouldBeNil)
			c.So(response, ShouldResemble, randResp)
		}()

		wg.Wait()
	})
}

func init() {
	// set logger level by env
	if os.Getenv("DEBUG") != "" {
		log.SetLevel(log.DebugLevel)
	}
}
