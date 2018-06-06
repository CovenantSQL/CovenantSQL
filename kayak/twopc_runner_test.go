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
	"github.com/stretchr/testify/mock"
	"github.com/thunderdb/ThunderDB/crypto/signature"
	"github.com/thunderdb/ThunderDB/twopc"
)

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

func TestTwoPCRunner_Init(t *testing.T) {
	// invalid config
	Convey("test invalid config", t, func() {
		runner := NewTwoPCRunner()
		config := &MockConfig{}
		err := runner.Init(config, nil, nil, nil, nil)
		So(err, ShouldNotBeNil)
	})

	Convey("test nil parameters", t, func() {
		runner := NewTwoPCRunner()
		config := &TwoPCConfig{}
		err := runner.Init(config, nil, nil, nil, nil)
		So(err, ShouldNotBeNil)
	})

	Convey("test sign broken peers", t, func() {
		runner := NewTwoPCRunner()
		logger := log.New()
		logger.SetLevel(log.FatalLevel)
		config := &TwoPCConfig{
			RuntimeConfig: RuntimeConfig{
				Logger: logger,
			},
		}
		peers := testPeersFixture(1, []*Server{
			{
				Role:    Leader,
				ID:      "happy",
				Address: "happy_address",
			},
		})
		// change term to invalidate signature
		peers.Term = 2
		mockRouter := &MockTransportRouter{
			transports: make(map[ServerID]*MockTransport),
		}
		mockLogStore := &MockLogStore{}
		mockStableStore := &MockStableStore{}
		mockTransport := mockRouter.getTransport("happy")
		testLog := &Log{
			Term:  1,
			Index: 1,
		}
		testLog.ComputeHash()
		mockStableStore.On("GetUint64", keyCurrentTerm).Return(uint64(1), nil)
		mockStableStore.On("GetUint64", keyCommittedIndex).Return(uint64(1), nil)
		mockStableStore.On("SetUint64", keyCurrentTerm, uint64(2)).Return(nil)
		mockLogStore.On("GetLog", uint64(1), mock.AnythingOfType("*kayak.Log")).
			Return(nil).Run(func(args mock.Arguments) {
			arg := args.Get(1).(*Log)
			*arg = *testLog
		})
		mockLogStore.On("LastIndex").Return(uint64(2), nil)
		mockLogStore.On("DeleteRange",
			mock.AnythingOfType("uint64"), mock.AnythingOfType("uint64")).Return(nil)

		err := runner.Init(config, peers, mockLogStore, mockStableStore, mockTransport)
		So(err, ShouldNotBeNil)
	})

	Convey("test log restore", t, func() {
		runner := NewTwoPCRunner()
		logger := log.New()
		logger.SetLevel(log.FatalLevel)
		config := &TwoPCConfig{
			RuntimeConfig: RuntimeConfig{
				Logger: logger,
			},
		}
		peers := testPeersFixture(1, []*Server{
			{
				Role:    Leader,
				ID:      "happy",
				Address: "happy_address",
			},
		})
		mockRouter := &MockTransportRouter{
			transports: make(map[ServerID]*MockTransport),
		}
		mockLogStore := &MockLogStore{}
		mockStableStore := &MockStableStore{}
		mockTransport := mockRouter.getTransport("happy")
		unknownErr := errors.New("unknown error")

		Convey("failed getting currentTerm from log", func() {
			mockStableStore.On("GetUint64", keyCurrentTerm).Return(uint64(0), unknownErr)

			err := runner.Init(config, peers, mockLogStore, mockStableStore, mockTransport)
			So(err, ShouldNotBeNil)
		})

		Convey("currentTerm in log older than term in peers", func() {
			mockStableStore.On("GetUint64", keyCurrentTerm).Return(uint64(2), nil)

			err := runner.Init(config, peers, mockLogStore, mockStableStore, mockTransport)
			So(err, ShouldNotBeNil)
		})

		Convey("get last committed index failed", func() {
			mockStableStore.On("GetUint64", keyCurrentTerm).Return(uint64(1), nil)
			mockStableStore.On("GetUint64", keyCommittedIndex).Return(uint64(0), unknownErr)

			err := runner.Init(config, peers, mockLogStore, mockStableStore, mockTransport)
			So(err, ShouldNotBeNil)
		})

		Convey("get last committed log data failed", func() {
			mockStableStore.On("GetUint64", keyCurrentTerm).Return(uint64(1), nil)
			mockStableStore.On("GetUint64", keyCommittedIndex).Return(uint64(1), nil)
			mockLogStore.On("GetLog", uint64(1), mock.Anything).Return(unknownErr)

			err := runner.Init(config, peers, mockLogStore, mockStableStore, mockTransport)
			So(err, ShouldNotBeNil)
		})

		Convey("last committed log with higher term than peers", func() {
			mockStableStore.On("GetUint64", keyCurrentTerm).Return(uint64(1), nil)
			mockStableStore.On("GetUint64", keyCommittedIndex).Return(uint64(1), nil)
			mockLogStore.On("GetLog", uint64(1), mock.AnythingOfType("*kayak.Log")).
				Return(nil).Run(func(args mock.Arguments) {
				arg := args.Get(1).(*Log)
				*arg = Log{
					Term:  2,
					Index: 1,
				}
			})

			err := runner.Init(config, peers, mockLogStore, mockStableStore, mockTransport)
			So(err, ShouldNotBeNil)
		})

		Convey("last committed log not equal to index field", func() {
			mockStableStore.On("GetUint64", keyCurrentTerm).Return(uint64(1), nil)
			mockStableStore.On("GetUint64", keyCommittedIndex).Return(uint64(1), nil)
			mockLogStore.On("GetLog", uint64(1), mock.AnythingOfType("*kayak.Log")).
				Return(nil).Run(func(args mock.Arguments) {
				arg := args.Get(1).(*Log)
				*arg = Log{
					Term:  1,
					Index: 2,
				}
			})

			err := runner.Init(config, peers, mockLogStore, mockStableStore, mockTransport)
			So(err, ShouldNotBeNil)
		})

		Convey("get last index failed", func() {
			mockStableStore.On("GetUint64", keyCurrentTerm).Return(uint64(1), nil)
			mockStableStore.On("GetUint64", keyCommittedIndex).Return(uint64(1), nil)
			mockLogStore.On("GetLog", uint64(1), mock.AnythingOfType("*kayak.Log")).
				Return(nil).Run(func(args mock.Arguments) {
				arg := args.Get(1).(*Log)
				*arg = Log{
					Term:  1,
					Index: 1,
				}
			})
			mockLogStore.On("LastIndex").Return(uint64(0), unknownErr)

			err := runner.Init(config, peers, mockLogStore, mockStableStore, mockTransport)
			So(err, ShouldNotBeNil)
		})

		Convey("last index overlaps committed index", func() {
			testLog := &Log{
				Term:  1,
				Index: 1,
			}
			testLog.ComputeHash()
			mockStableStore.On("GetUint64", keyCurrentTerm).Return(uint64(1), nil)
			mockStableStore.On("GetUint64", keyCommittedIndex).Return(uint64(1), nil)
			mockStableStore.On("SetUint64", keyCurrentTerm, uint64(1)).Return(nil)
			mockLogStore.On("GetLog", uint64(1), mock.AnythingOfType("*kayak.Log")).
				Return(nil).Run(func(args mock.Arguments) {
				arg := args.Get(1).(*Log)
				*arg = *testLog
			})
			mockLogStore.On("LastIndex").Return(uint64(2), nil)
			mockLogStore.On("DeleteRange",
				mock.AnythingOfType("uint64"), mock.AnythingOfType("uint64")).Return(nil)

			err := runner.Init(config, peers, mockLogStore, mockStableStore, mockTransport)
			mockLogStore.AssertCalled(t, "DeleteRange", uint64(2), uint64(2))

			So(err, ShouldBeNil)
			So(runner.currentTerm, ShouldEqual, uint64(1))
			So(runner.lastLogTerm, ShouldEqual, uint64(1))
			So(runner.lastLogIndex, ShouldEqual, 1)
			So(runner.lastLogHash, ShouldNotBeNil)
			So(runner.lastLogHash.IsEqual(&testLog.Hash), ShouldBeTrue)
		})
	})
}

func TestTwoPCRunner_Process(t *testing.T) {
	mockLogCodec := &MockLogCodec{}
	mockRouter := &MockTransportRouter{
		transports: make(map[ServerID]*MockTransport),
	}

	type createMockRes struct {
		runner      *TwoPCRunner
		transport   *MockTransport
		worker      *MockWorker
		config      *TwoPCConfig
		logStore    *MockLogStore
		stableStore *MockStableStore
	}

	createMock := func(serverID ServerID) (res *createMockRes) {
		res = &createMockRes{}
		logger := log.New()
		logger.SetLevel(log.FatalLevel)
		res.runner = NewTwoPCRunner()
		res.transport = mockRouter.getTransport(serverID)
		res.worker = &MockWorker{}
		res.config = &TwoPCConfig{
			RuntimeConfig: RuntimeConfig{
				RootDir:        "test_dir",
				LocalID:        serverID,
				Runner:         res.runner,
				Dialer:         res.transport,
				ProcessTimeout: time.Millisecond * 800,
				Logger:         logger,
			},
			LogCodec:        mockLogCodec,
			Storage:         res.worker,
			PrepareTimeout:  time.Millisecond * 200,
			CommitTimeout:   time.Millisecond * 200,
			RollbackTimeout: time.Millisecond * 200,
		}
		res.logStore = &MockLogStore{}
		res.stableStore = &MockStableStore{}

		// init with no log and no term info
		res.stableStore.On("GetUint64", keyCurrentTerm).Return(uint64(0), nil)
		res.stableStore.On("GetUint64", keyCommittedIndex).Return(uint64(0), nil)
		res.stableStore.On("SetUint64", keyCurrentTerm, uint64(1)).Return(nil)
		res.logStore.On("LastIndex").Return(uint64(0), nil)
		return
	}
	peers := testPeersFixture(1, []*Server{
		{
			Role:    Leader,
			ID:      "leader",
			Address: "leader_address",
		},
		{
			Role:    Follower,
			ID:      "follower",
			Address: "follower_address",
		},
	})

	Convey("call process on no leader", t, func() {
		mockRouter.ResetAll()
		mockRes := createMock("follower")

		err := mockRes.runner.Init(mockRes.config, peers, mockRes.logStore, mockRes.stableStore, mockRes.transport)

		So(err, ShouldBeNil)
		So(mockRes.runner.role, ShouldEqual, Follower)
		So(mockRes.runner.leader.ID, ShouldEqual, ServerID("leader"))

		// try call process
		testData, _ := mockLogCodec.Encode("test data")
		err = mockRes.runner.Process(testData)
		So(err, ShouldNotBeNil)
		So(err, ShouldEqual, ErrNotLeader)
	})

	Convey("call process on leader with single node", t, func() {
		mockRouter.ResetAll()

		// change server id to leader and set peers to single node
		peers := testPeersFixture(1, []*Server{
			{
				Role:    Leader,
				ID:      "leader",
				Address: "leader_address",
			},
		})
		mockRes := createMock("leader")

		err := mockRes.runner.Init(mockRes.config, peers, mockRes.logStore, mockRes.stableStore, mockRes.transport)

		So(err, ShouldBeNil)
		So(mockRes.runner.role, ShouldEqual, Leader)
		So(mockRes.runner.leader.ID, ShouldEqual, ServerID("leader"))

		Convey("commit", func() {
			// mock worker
			var callOrder []string
			mockRes.worker.On("Prepare", mock.Anything, "test data").
				Return(nil).Run(func(args mock.Arguments) {
				callOrder = append(callOrder, "prepare")
			})
			mockRes.logStore.On("StoreLog", mock.AnythingOfType("*kayak.Log")).
				Return(nil).Run(func(args mock.Arguments) {
				callOrder = append(callOrder, "store_log")
			})
			mockRes.worker.On("Commit", mock.Anything, "test data").
				Return(nil).Run(func(args mock.Arguments) {
				callOrder = append(callOrder, "commit")
			})
			mockRes.stableStore.On("SetUint64", keyCommittedIndex, uint64(1)).
				Return(nil).Run(func(args mock.Arguments) {
				callOrder = append(callOrder, "update_committed")
			})

			testData, _ := mockLogCodec.Encode("test data")

			// try call process
			err = mockRes.runner.Process(testData)

			So(err, ShouldBeNil)

			// test call orders
			So(callOrder, ShouldResemble, []string{
				"prepare",
				"store_log",
				"commit",
				"update_committed",
			})
		})

		Convey("rollback", func() {
			// mock worker
			var callOrder []string
			unknownErr := errors.New("unknown error")
			mockRes.worker.On("Prepare", mock.Anything, "test data").
				Return(unknownErr).Run(func(args mock.Arguments) {
				callOrder = append(callOrder, "prepare")
			})
			mockRes.logStore.On("StoreLog", mock.AnythingOfType("*kayak.Log")).
				Return(nil).Run(func(args mock.Arguments) {
				callOrder = append(callOrder, "store_log")
			})
			mockRes.worker.On("Rollback", mock.Anything, "test data").
				Return(nil).Run(func(args mock.Arguments) {
				callOrder = append(callOrder, "rollback")
			})
			mockRes.logStore.On("DeleteRange", uint64(1), uint64(1)).
				Return(nil).Run(func(args mock.Arguments) {
				callOrder = append(callOrder, "truncate_log")
			})

			testData, _ := mockLogCodec.Encode("test data")

			// try call process
			err = mockRes.runner.Process(testData)
			So(err, ShouldNotBeNil)

			// no log should be written to local log store after failed preparing
			mockRes.logStore.AssertNotCalled(t, "StoreLog", mock.AnythingOfType("*kayak.Log"))
			So(callOrder, ShouldResemble, []string{
				"prepare",
				"truncate_log",
				"rollback",
			})
		})

		Convey("prepare timeout", FailureContinues, func(c C) {
			unknownErr := errors.New("unknown error")
			mockRes.worker.On("Prepare", mock.Anything, "test data").
				Return(unknownErr).After(time.Millisecond * 250).Run(func(args mock.Arguments) {
				ctx := args.Get(0).(context.Context)
				c.So(ctx.Err(), ShouldNotBeNil)
			})
			mockRes.worker.On("Rollback", mock.Anything, "test data").Return(nil)
			mockRes.logStore.On("DeleteRange", uint64(1), uint64(1)).Return(nil)

			testData, _ := mockLogCodec.Encode("test data")

			// try call process
			err = mockRes.runner.Process(testData)

			So(err, ShouldNotBeNil)
		})

		Convey("commit timeout", FailureContinues, func(c C) {
			unknownErr := errors.New("unknown error")
			mockRes.worker.On("Prepare", mock.Anything, "test data").
				Return(nil)
			mockRes.logStore.On("StoreLog", mock.AnythingOfType("*kayak.Log")).
				Return(nil)
			mockRes.worker.On("Commit", mock.Anything, "test data").
				Return(unknownErr).After(time.Millisecond * 250).Run(func(args mock.Arguments) {
				ctx := args.Get(0).(context.Context)
				c.So(ctx.Err(), ShouldNotBeNil)
			})
			mockRes.stableStore.On("SetUint64", keyCommittedIndex, uint64(1)).
				Return(nil)

			testData, _ := mockLogCodec.Encode("test data")

			// try call process
			err = mockRes.runner.Process(testData)

			So(err, ShouldNotBeNil)
		})

		Convey("rollback timeout", FailureContinues, func(c C) {
			prepareErr := errors.New("prepare error")
			rollbackErr := errors.New("rollback error")
			mockRes.worker.On("Prepare", mock.Anything, "test data").
				Return(prepareErr)
			mockRes.logStore.On("StoreLog", mock.AnythingOfType("*kayak.Log")).
				Return(nil)
			mockRes.worker.On("Rollback", mock.Anything, "test data").
				Return(rollbackErr).After(time.Millisecond * 250).Run(func(args mock.Arguments) {
				ctx := args.Get(0).(context.Context)
				c.So(ctx.Err(), ShouldNotBeNil)
			})
			mockRes.logStore.On("DeleteRange", uint64(1), uint64(1)).Return(nil)

			testData, _ := mockLogCodec.Encode("test data")

			// try call process
			err = mockRes.runner.Process(testData)

			// rollback error is ignored
			So(err, ShouldNotBeNil)
			So(err, ShouldEqual, prepareErr)
		})
	})

	Convey("call process on leader with multiple nodes", t, func(c C) {
		mockRouter.ResetAll()

		peers := testPeersFixture(1, []*Server{
			{
				Role:    Leader,
				ID:      "leader",
				Address: "leader_address",
			},
			{
				Role:    Follower,
				ID:      "follower1",
				Address: "follower_address",
			},
			{
				Role:    Follower,
				ID:      "follower2",
				Address: "follower2_address",
			},
		})
		initMock := func(r *createMockRes) {
			store := NewMockInmemStore()
			err := r.runner.Init(r.config, peers, store, store, r.transport)
			So(err, ShouldBeNil)
		}

		Convey("commit", func() {
			lMock := createMock("leader")
			f1Mock := createMock("follower1")
			f2Mock := createMock("follower2")

			// init
			initMock(lMock)
			initMock(f1Mock)
			initMock(f2Mock)

			testData, _ := mockLogCodec.Encode("test data")

			var callOrder []string
			f1Mock.worker.On("Prepare", mock.Anything, "test data").
				Return(nil).Run(func(args mock.Arguments) {
				callOrder = append(callOrder, "f_prepare")
			})
			f2Mock.worker.On("Prepare", mock.Anything, "test data").
				Return(nil).Run(func(args mock.Arguments) {
				callOrder = append(callOrder, "f_prepare")
			})
			f1Mock.worker.On("Commit", mock.Anything, "test data").
				Return(nil).Run(func(args mock.Arguments) {
				callOrder = append(callOrder, "f_commit")
			})
			f2Mock.worker.On("Commit", mock.Anything, "test data").
				Return(nil).Run(func(args mock.Arguments) {
				callOrder = append(callOrder, "f_commit")
			})
			lMock.worker.On("Prepare", mock.Anything, "test data").
				Return(nil).Run(func(args mock.Arguments) {
				callOrder = append(callOrder, "l_prepare")
			})
			lMock.worker.On("Commit", mock.Anything, "test data").
				Return(nil).Run(func(args mock.Arguments) {
				callOrder = append(callOrder, "l_commit")
			})

			// try call process
			err := lMock.runner.Process(testData)

			So(err, ShouldBeNil)

			// test call orders
			So(callOrder, ShouldResemble, []string{
				"f_prepare",
				"f_prepare",
				"l_prepare",
				"f_commit",
				"f_commit",
				"l_commit",
			})
		})

		Convey("rollback", func() {
			lMock := createMock("leader")
			f1Mock := createMock("follower1")
			f2Mock := createMock("follower2")

			// init
			initMock(lMock)
			initMock(f1Mock)
			initMock(f2Mock)

			testData, _ := mockLogCodec.Encode("test data")

			var callOrder []string
			unknownErr := errors.New("unknown error")
			// f1 prepare with error
			f1Mock.worker.On("Prepare", mock.Anything, "test data").
				Return(unknownErr).Run(func(args mock.Arguments) {
				callOrder = append(callOrder, "f_prepare")
			})
			f1Mock.worker.On("Rollback", mock.Anything, "test data").
				Return(nil).Run(func(args mock.Arguments) {
				callOrder = append(callOrder, "f_rollback")
			})
			// f2 prepare with no error
			f2Mock.worker.On("Prepare", mock.Anything, "test data").
				Return(nil).Run(func(args mock.Arguments) {
				callOrder = append(callOrder, "f_prepare")
			})
			f2Mock.worker.On("Rollback", mock.Anything, "test data").
				Return(nil).Run(func(args mock.Arguments) {
				callOrder = append(callOrder, "f_rollback")
			})
			lMock.worker.On("Prepare", mock.Anything, "test data").
				Return(nil).Run(func(args mock.Arguments) {
				callOrder = append(callOrder, "l_prepare")
			})
			lMock.worker.On("Rollback", mock.Anything, "test data").
				Return(nil).Run(func(args mock.Arguments) {
				callOrder = append(callOrder, "l_rollback")
			})

			// try call process
			err := lMock.runner.Process(testData)

			So(err, ShouldNotBeNil)
			So(err, ShouldEqual, unknownErr)

			// test call orders
			// prepare failed, so no l_prepare is called
			// since one prepare failed, only one f_rollback with be triggered
			// TODO, mixing coordinator role with worker role in leader node may be a bad idea, need code refactor
			So(callOrder, ShouldResemble, []string{
				"f_prepare",
				"f_prepare",
				//"l_prepare",
				"l_rollback",
				"f_rollback",
				//"f_rollback",
			})
		})
	})

	Convey("sybil test", t, func() {
		mockRouter.ResetAll()

		peers := testPeersFixture(1, []*Server{
			{
				Role:    Leader,
				ID:      "leader",
				Address: "leader_address",
			},
			{
				Role:    Follower,
				ID:      "follower1",
				Address: "follower_address",
			},
			{
				Role:    Follower,
				ID:      "follower2",
				Address: "follower2_address",
			},
		})
		initMock := func(r *createMockRes) {
			err := r.runner.Init(r.config, peers, r.logStore, r.stableStore, r.transport)
			So(err, ShouldBeNil)
		}

		Convey("request from non-leader", func() {
			lMock := createMock("leader")
			f1Mock := createMock("follower1")
			f2Mock := createMock("follower1")

			// init
			initMock(lMock)
			initMock(f1Mock)
			initMock(f2Mock)

			// fake request
			testData, _ := mockLogCodec.Encode("test data")
			fakeLog := &Log{
				Term:  1,
				Index: 1,
				Data:  testData,
			}

			var err, remoteErr error
			err = f1Mock.transport.Request(
				context.Background(),
				f2Mock.config.LocalID,
				"Prepare",
				fakeLog,
				&remoteErr,
			)
			So(err, ShouldBeNil)
			So(remoteErr, ShouldNotBeNil)
			So(remoteErr, ShouldEqual, ErrInvalidRequest)
		})

		Convey("send invalid request", func() {
			lMock := createMock("leader")
			f1Mock := createMock("follower1")

			// init
			initMock(lMock)
			initMock(f1Mock)

			// fake request
			var err, remoteErr error
			err = lMock.transport.Request(
				context.Background(),
				f1Mock.config.LocalID,
				"invalid request",
				nil,
				&remoteErr,
			)
			So(err, ShouldBeNil)
			So(remoteErr, ShouldNotBeNil)
			So(remoteErr, ShouldEqual, ErrInvalidRequest)
		})

		Convey("log could not be decoded", func() {
			lMock := createMock("leader")
			f1Mock := createMock("follower1")

			// init
			initMock(lMock)
			initMock(f1Mock)

			var err, remoteErr error
			err = lMock.transport.Request(
				context.Background(),
				f1Mock.config.LocalID,
				"Prepare",
				nil,
				&remoteErr,
			)
			So(err, ShouldBeNil)
			So(remoteErr, ShouldNotBeNil)
			So(remoteErr, ShouldEqual, ErrInvalidLog)
		})

		Convey("invalid log Hash/LastLogHash", func() {
			// TODO, test log with invalid hash
		})
	})
}

func TestTwoPCRunner_UpdatePeers(t *testing.T) {
	mockLogCodec := &MockLogCodec{}
	mockRouter := &MockTransportRouter{
		transports: make(map[ServerID]*MockTransport),
	}

	type createMockRes struct {
		runner      *TwoPCRunner
		transport   *MockTransport
		worker      *MockWorker
		config      *TwoPCConfig
		logStore    *MockLogStore
		stableStore *MockStableStore
	}

	createMock := func(serverID ServerID) (res *createMockRes) {
		res = &createMockRes{}
		logger := log.New()
		logger.SetLevel(log.FatalLevel)
		res.runner = NewTwoPCRunner()
		res.transport = mockRouter.getTransport(serverID)
		res.worker = &MockWorker{}
		res.config = &TwoPCConfig{
			RuntimeConfig: RuntimeConfig{
				RootDir:        "test_dir",
				LocalID:        serverID,
				Runner:         res.runner,
				Dialer:         res.transport,
				ProcessTimeout: time.Millisecond * 800,
				Logger:         logger,
			},
			LogCodec:        mockLogCodec,
			Storage:         res.worker,
			PrepareTimeout:  time.Millisecond * 200,
			CommitTimeout:   time.Millisecond * 200,
			RollbackTimeout: time.Millisecond * 200,
		}
		res.logStore = &MockLogStore{}
		res.stableStore = &MockStableStore{}

		// init with no log and no term info
		res.stableStore.On("GetUint64", keyCurrentTerm).Return(uint64(0), nil)
		res.stableStore.On("GetUint64", keyCommittedIndex).Return(uint64(0), nil)
		res.stableStore.On("SetUint64", keyCurrentTerm, uint64(1)).Return(nil)
		res.logStore.On("LastIndex").Return(uint64(0), nil)
		return
	}
	peers := testPeersFixture(1, []*Server{
		{
			Role:    Leader,
			ID:      "leader",
			Address: "leader_address",
		},
		{
			Role:    Follower,
			ID:      "follower1",
			Address: "follower_address",
		},
		{
			Role:    Follower,
			ID:      "follower2",
			Address: "follower2_address",
		},
	})
	initMock := func(r *createMockRes) {
		err := r.runner.Init(r.config, peers, r.logStore, r.stableStore, r.transport)
		So(err, ShouldBeNil)
	}

	Convey("update peers with incorrect leader", t, func() {
		mockRouter.ResetAll()

		lMock := createMock("leader")
		f1Mock := createMock("follower1")
		f2Mock := createMock("follower2")
	})
}

func init() {
	// set logger level by env
	if os.Getenv("DEBUG") != "" {
		log.SetLevel(log.DebugLevel)
	}
}
