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

package kayak

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/btcsuite/btcd/btcec"
	log "github.com/sirupsen/logrus"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/thunderdb/ThunderDB/crypto/signature"
	"github.com/thunderdb/ThunderDB/twopc"
)

// common mocks
type MockLogCodec struct{}

type MockTransportRouter struct {
	reqSeq        uint64
	transports    map[ServerID]*MockTransport
	transportLock sync.Mutex
}

type MockTransport struct {
	serverID  ServerID
	router    *MockTransportRouter
	queue     chan Request
	waitQueue chan *MockResponse
	giveUp    map[uint64]bool
}

type MockRequest struct {
	transport *MockTransport
	ctx       context.Context
	RequestID uint64
	ServerID  ServerID
	Method    string
	Payload   interface{}
}

type MockResponse struct {
	ResponseID uint64
	Payload    interface{}
}

type MockTwoPCWorker struct {
	serverID ServerID
	state    string
	data     int64
	total    int64
}

var (
	_ twopc.Worker = &MockTwoPCWorker{}
)

func (m *MockLogCodec) Encode(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (m *MockLogCodec) Decode(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func (m *MockTransportRouter) getTransport(serverID ServerID) *MockTransport {
	m.transportLock.Lock()
	defer m.transportLock.Unlock()

	if _, ok := m.transports[serverID]; !ok {
		m.transports[serverID] = &MockTransport{
			serverID:  serverID,
			router:    m,
			queue:     make(chan Request, 1000),
			waitQueue: make(chan *MockResponse, 1000),
			giveUp:    make(map[uint64]bool),
		}
	}

	return m.transports[serverID]
}

func (m *MockTransportRouter) ResetTransport(serverID ServerID) {
	m.transportLock.Lock()
	defer m.transportLock.Unlock()

	if _, ok := m.transports[serverID]; ok {
		// reset
		delete(m.transports, serverID)
	}
}

func (m *MockTransportRouter) ResetAll() {
	m.transportLock.Lock()
	defer m.transportLock.Unlock()

	m.transports = make(map[ServerID]*MockTransport)
}

func (m *MockTransportRouter) getReqID() uint64 {
	return atomic.AddUint64(&m.reqSeq, 1)
}

func (m *MockTransport) Request(ctx context.Context, serverID ServerID,
	method string, args interface{}, response interface{}) error {
	resp, err := m.router.getTransport(serverID).sendRequest(&MockRequest{
		RequestID: m.router.getReqID(),
		ServerID:  m.serverID,
		Method:    method,
		Payload:   args,
		ctx:       ctx,
	})

	if err == nil && resp != nil {
		// set response
		rv := reflect.Indirect(reflect.ValueOf(response))
		if rv.CanSet() && resp != nil {
			rv.Set(reflect.ValueOf(resp))
		}
	}

	return err
}

func (m *MockTransport) Process() <-chan Request {
	return m.queue
}

func (m *MockTransport) sendRequest(req Request) (interface{}, error) {
	r := req.(*MockRequest)
	r.transport = m

	if log.GetLevel() >= log.DebugLevel {
		fmt.Println()
	}
	log.Debugf("[%v] [%v] -> [%v] request %v", r.RequestID, r.ServerID, req.GetServerID(), r.GetRequest())
	m.queue <- r

	for {
		select {
		case <-r.ctx.Done():
			// deadline reached
			log.Debugf("[%v] [%v] -> [%v] request timeout",
				r.RequestID, r.ServerID, req.GetServerID())
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
				log.Debugf("[%v] [%v] -> [%v] response %v",
					r.RequestID, req.GetServerID(), r.ServerID, res.Payload)
				return res.Payload, nil
			}
		}
	}
}

func (m *MockRequest) GetServerID() ServerID {
	return m.ServerID
}

func (m *MockRequest) GetMethod() string {
	return m.Method
}

func (m *MockRequest) GetRequest() interface{} {
	return m.Payload
}

func (m *MockRequest) SendResponse(v interface{}) error {
	m.transport.waitQueue <- &MockResponse{
		ResponseID: m.RequestID,
		Payload:    v,
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

func (w *MockTwoPCWorker) Commit(ctx context.Context, wb twopc.WriteBatch) error {
	// test commit
	if w.state != "prepared" {
		return errors.New("invalid state")
	}

	if !reflect.DeepEqual(wb, w.data) {
		return errors.New("commit data not same as last")
	}

	w.total += w.data
	w.state = ""

	return nil
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
	privKey, pubKey := signature.PrivKeyFromBytes(btcec.S256(), testPriv)

	newServers := make([]*Server, 0, len(servers))
	var leaderNode *Server

	for _, s := range servers {
		newS := &Server{
			Role:    s.Role,
			ID:      s.ID,
			Address: s.Address,
			PubKey:  pubKey,
		}
		newServers = append(newServers, newS)
		if newS.Role == Leader {
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

// test mock library itself
func TestMockTransport(t *testing.T) {
	Convey("test transport with request timeout", t, func() {
		mockRouter := &MockTransportRouter{
			transports: make(map[ServerID]*MockTransport),
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*50)
		defer cancel()

		var err, remoteErr error
		err = mockRouter.getTransport("a").Request(ctx, "b", "Test", "happy", &remoteErr)

		So(err, ShouldNotBeNil)
		So(remoteErr, ShouldBeNil)
	})

	Convey("test transport with successful request", t, func(c C) {
		mockRouter := &MockTransportRouter{
			transports: make(map[ServerID]*MockTransport),
		}
		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case req := <-mockRouter.getTransport("d").Process():
				c.So(req.GetServerID(), ShouldEqual, ServerID("c"))
				c.So(req.GetMethod(), ShouldEqual, "Test")
				c.So(req.GetRequest(), ShouldResemble, "happy")
				req.SendResponse("happy too")
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			var err error
			var response string
			err = mockRouter.getTransport("c").Request(
				context.Background(), "d", "Test", "happy", &response)

			c.So(err, ShouldBeNil)
			c.So(response, ShouldEqual, "happy too")
		}()

		wg.Wait()
	})

	Convey("test transport with concurrent request", t, FailureContinues, func(c C) {
		mockRouter := &MockTransportRouter{
			transports: make(map[ServerID]*MockTransport),
		}
		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			defer wg.Done()
			var err error
			var response string
			err = mockRouter.getTransport("e").Request(
				context.Background(), "g", "test1", "happy", &response)

			c.So(err, ShouldBeNil)
			c.So(response, ShouldEqual, "happy e test1")
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			var err error
			var response string
			err = mockRouter.getTransport("f").Request(
				context.Background(), "g", "test2", "happy", &response)

			c.So(err, ShouldBeNil)
			c.So(response, ShouldEqual, "happy f test2")
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()

			for i := 0; i < 2; i++ {
				select {
				case req := <-mockRouter.getTransport("g").Process():
					c.So(req.GetServerID(), ShouldBeIn, []ServerID{"e", "f"})
					c.So(req.GetMethod(), ShouldBeIn, []string{"test1", "test2"})
					c.So(req.GetRequest(), ShouldResemble, "happy")
					req.SendResponse(fmt.Sprintf("happy %s %s", req.GetServerID(), req.GetMethod()))
				}
			}
		}()

		wg.Wait()
	})

	Convey("test transport with piped request", t, FailureContinues, func(c C) {
		mockRouter := &MockTransportRouter{
			transports: make(map[ServerID]*MockTransport),
		}
		var wg sync.WaitGroup

		randReq := rand.Int63()
		randResp := rand.Int63()

		t.Logf("test with request %d, response %d", randReq, randResp)

		wg.Add(1)
		go func() {
			defer wg.Done()
			var err error
			var response interface{}
			var req Request

			select {
			case req = <-mockRouter.getTransport("j").Process():
				c.So(req.GetServerID(), ShouldEqual, ServerID("i"))
				c.So(req.GetMethod(), ShouldEqual, "pass1")
			}

			err = mockRouter.getTransport("j").Request(
				context.Background(), "k", "pass2", req.GetRequest(), &response)

			c.So(err, ShouldBeNil)
			req.SendResponse(response)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case req := <-mockRouter.getTransport("k").Process():
				c.So(req.GetServerID(), ShouldEqual, ServerID("j"))
				c.So(req.GetMethod(), ShouldEqual, "pass2")
				c.So(req.GetRequest(), ShouldResemble, randReq)
				req.SendResponse(randResp)
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			var err error
			var response interface{}

			err = mockRouter.getTransport("i").Request(
				context.Background(), "j", "pass1", randReq, &response)

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
