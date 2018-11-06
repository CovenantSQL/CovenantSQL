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
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/twopc"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/mock"
)

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
		log.SetLevel(log.FatalLevel)
		config := &TwoPCConfig{
			RuntimeConfig: RuntimeConfig{},
		}
		peers := testPeersFixture(1, []*Server{
			{
				Role: proto.Leader,
				ID:   "happy",
			},
		})
		// change term to invalidate signature
		peers.Term = 2
		mockRouter := &MockTransportRouter{
			transports: make(map[proto.NodeID]*MockTransport),
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
		log.SetLevel(log.FatalLevel)
		config := &TwoPCConfig{
			RuntimeConfig: RuntimeConfig{},
		}
		peers := testPeersFixture(1, []*Server{
			{
				Role: proto.Leader,
				ID:   "happy",
			},
		})
		mockRouter := &MockTransportRouter{
			transports: make(map[proto.NodeID]*MockTransport),
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

func TestTwoPCRunner_Apply(t *testing.T) {
	mockRouter := &MockTransportRouter{
		transports: make(map[proto.NodeID]*MockTransport),
	}

	type createMockRes struct {
		runner      *TwoPCRunner
		transport   *MockTransport
		worker      *MockWorker
		config      *TwoPCConfig
		logStore    *MockLogStore
		stableStore *MockStableStore
	}

	createMock := func(nodeID proto.NodeID) (res *createMockRes) {
		res = &createMockRes{}
		log.SetLevel(log.FatalLevel)
		res.runner = NewTwoPCRunner()
		res.transport = mockRouter.getTransport(nodeID)
		res.worker = &MockWorker{}
		res.config = &TwoPCConfig{
			RuntimeConfig: RuntimeConfig{
				RootDir:        "test_dir",
				LocalID:        nodeID,
				Runner:         res.runner,
				Transport:      res.transport,
				ProcessTimeout: time.Millisecond * 300,
			},
			Storage: res.worker,
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
			Role: proto.Leader,
			ID:   "leader",
		},
		{
			Role: proto.Follower,
			ID:   "follower",
		},
	})

	Convey("call process on no leader", t, func() {
		mockRouter.ResetAll()
		mockRes := createMock("follower")

		err := mockRes.runner.Init(mockRes.config, peers, mockRes.logStore, mockRes.stableStore, mockRes.transport)

		So(err, ShouldBeNil)
		So(mockRes.runner.role, ShouldEqual, proto.Follower)
		So(mockRes.runner.leader.ID, ShouldEqual, proto.NodeID("leader"))

		// try call process
		testPayload := []byte("test data")
		_, _, err = mockRes.runner.Apply(testPayload)
		So(err, ShouldNotBeNil)
		So(err, ShouldEqual, ErrNotLeader)
	})

	Convey("call process on leader with single node", t, func() {
		mockRouter.ResetAll()

		// change server id to leader and set peers to single node
		peers := testPeersFixture(1, []*Server{
			{
				Role: proto.Leader,
				ID:   "leader",
			},
		})
		mockRes := createMock("leader")

		err := mockRes.runner.Init(mockRes.config, peers, mockRes.logStore, mockRes.stableStore, mockRes.transport)

		So(err, ShouldBeNil)
		So(mockRes.runner.role, ShouldEqual, proto.Leader)
		So(mockRes.runner.leader.ID, ShouldEqual, proto.NodeID("leader"))

		Convey("commit", func() {
			testPayload := []byte("test data")

			// mock worker
			callOrder := &CallCollector{}
			mockRes.worker.On("Prepare", mock.Anything, testPayload).
				Return(nil).Run(func(args mock.Arguments) {
				callOrder.Append("prepare")
			})
			mockRes.logStore.On("StoreLog", mock.AnythingOfType("*kayak.Log")).
				Return(nil).Run(func(args mock.Arguments) {
				callOrder.Append("store_log")
			})
			mockRes.worker.On("Commit", mock.Anything, testPayload).
				Return(nil, nil).Run(func(args mock.Arguments) {
				callOrder.Append("commit")
			})
			mockRes.stableStore.On("SetUint64", keyCommittedIndex, uint64(1)).
				Return(nil).Run(func(args mock.Arguments) {
				callOrder.Append("update_committed")
			})

			// try call process
			var offset uint64
			_, offset, err = mockRes.runner.Apply(testPayload)
			So(err, ShouldBeNil)
			So(offset, ShouldEqual, uint64(1))

			// test call orders
			So(callOrder.Get(), ShouldResemble, []string{
				"prepare",
				"store_log",
				"commit",
				"update_committed",
			})
		})

		Convey("rollback", func() {
			testPayload := []byte("test data")

			// mock worker
			callOrder := &CallCollector{}
			unknownErr := errors.New("unknown error")
			mockRes.worker.On("Prepare", mock.Anything, testPayload).
				Return(unknownErr).Run(func(args mock.Arguments) {
				callOrder.Append("prepare")
			})
			mockRes.logStore.On("StoreLog", mock.AnythingOfType("*kayak.Log")).
				Return(nil).Run(func(args mock.Arguments) {
				callOrder.Append("store_log")
			})
			mockRes.worker.On("Rollback", mock.Anything, testPayload).
				Return(nil).Run(func(args mock.Arguments) {
				callOrder.Append("rollback")
			})
			mockRes.logStore.On("DeleteRange", uint64(1), uint64(1)).
				Return(nil).Run(func(args mock.Arguments) {
				callOrder.Append("truncate_log")
			})

			// try call process
			_, _, err = mockRes.runner.Apply(testPayload)
			So(err, ShouldNotBeNil)

			// no log should be written to local log store after failed preparing
			mockRes.logStore.AssertNotCalled(t, "StoreLog", mock.AnythingOfType("*kayak.Log"))
			So(callOrder.Get(), ShouldResemble, []string{
				"prepare",
				"truncate_log",
				"rollback",
			})
		})

		Convey("prepare timeout", FailureContinues, func(c C) {
			testPayload := []byte("test data")
			unknownErr := errors.New("unknown error")
			mockRes.worker.On("Prepare", mock.Anything, testPayload).
				Return(func(ctx context.Context, _ twopc.WriteBatch) error {
					c.So(ctx.Err(), ShouldNotBeNil)
					return unknownErr
				}).After(time.Millisecond * 400)
			mockRes.worker.On("Rollback", mock.Anything, testPayload).Return(nil)
			mockRes.logStore.On("DeleteRange", uint64(1), uint64(1)).Return(nil)

			// try call process
			_, _, err = mockRes.runner.Apply(testPayload)

			So(err, ShouldNotBeNil)
		})

		Convey("commit timeout", FailureContinues, func(c C) {
			testPayload := []byte("test data")
			unknownErr := errors.New("unknown error")
			mockRes.worker.On("Prepare", mock.Anything, testPayload).
				Return(nil)
			mockRes.logStore.On("StoreLog", mock.AnythingOfType("*kayak.Log")).
				Return(nil)
			mockRes.worker.On("Commit", mock.Anything, testPayload).
				Return(nil, func(ctx context.Context, _ twopc.WriteBatch) error {
					c.So(ctx.Err(), ShouldNotBeNil)
					return unknownErr
				}).After(time.Millisecond * 400)
			mockRes.stableStore.On("SetUint64", keyCommittedIndex, uint64(1)).
				Return(nil)

			// try call process
			_, _, err = mockRes.runner.Apply(testPayload)

			So(err, ShouldNotBeNil)
		})

		Convey("rollback timeout", FailureContinues, func(c C) {
			testPayload := []byte("test data")
			prepareErr := errors.New("prepare error")
			rollbackErr := errors.New("rollback error")
			mockRes.worker.On("Prepare", mock.Anything, testPayload).
				Return(prepareErr)
			mockRes.logStore.On("StoreLog", mock.AnythingOfType("*kayak.Log")).
				Return(nil)
			mockRes.worker.On("Rollback", mock.Anything, testPayload).
				Return(func(ctx context.Context, _ twopc.WriteBatch) error {
					c.So(ctx.Err(), ShouldNotBeNil)
					return rollbackErr
				}).After(time.Millisecond * 400)
			mockRes.logStore.On("DeleteRange", uint64(1), uint64(1)).Return(nil)

			// try call process
			_, _, err = mockRes.runner.Apply(testPayload)

			// rollback error is ignored
			So(err, ShouldNotBeNil)
			So(err, ShouldEqual, prepareErr)
		})
	})

	Convey("call process on leader with multiple nodes", t, func(c C) {
		mockRouter.ResetAll()

		peers := testPeersFixture(1, []*Server{
			{
				Role: proto.Leader,
				ID:   "leader",
			},
			{
				Role: proto.Follower,
				ID:   "follower1",
			},
			{
				Role: proto.Follower,
				ID:   "follower2",
			},
		})
		initMock := func(mocks ...*createMockRes) {
			for _, r := range mocks {
				store := NewMockInmemStore()
				err := r.runner.Init(r.config, peers, store, store, r.transport)
				So(err, ShouldBeNil)
			}
		}

		Convey("commit", func() {
			lMock := createMock("leader")
			f1Mock := createMock("follower1")
			f2Mock := createMock("follower2")

			// init
			initMock(lMock, f1Mock, f2Mock)

			testPayload := []byte("test data")

			callOrder := &CallCollector{}
			f1Mock.worker.On("Prepare", mock.Anything, testPayload).
				Return(nil).Run(func(args mock.Arguments) {
				callOrder.Append("prepare")
			})
			f2Mock.worker.On("Prepare", mock.Anything, testPayload).
				Return(nil).Run(func(args mock.Arguments) {
				callOrder.Append("prepare")
			})
			f1Mock.worker.On("Commit", mock.Anything, testPayload).
				Return(nil, nil).Run(func(args mock.Arguments) {
				callOrder.Append("commit")
			})
			f2Mock.worker.On("Commit", mock.Anything, testPayload).
				Return(nil, nil).Run(func(args mock.Arguments) {
				callOrder.Append("commit")
			})
			lMock.worker.On("Prepare", mock.Anything, testPayload).
				Return(nil).Run(func(args mock.Arguments) {
				callOrder.Append("prepare")
			})
			lMock.worker.On("Commit", mock.Anything, testPayload).
				Return(nil, nil).Run(func(args mock.Arguments) {
				callOrder.Append("commit")
			})

			// try call process
			_, _, err := lMock.runner.Apply(testPayload)

			So(err, ShouldBeNil)

			// test call orders
			So(callOrder.Get(), ShouldResemble, []string{
				"prepare",
				"prepare",
				"prepare",
				"commit",
				"commit",
				"commit",
			})

			lastLogHash := lMock.runner.lastLogHash
			lastLogIndex := lMock.runner.lastLogIndex
			lastLogTerm := lMock.runner.lastLogTerm

			So(lastLogHash, ShouldNotBeNil)
			So(lastLogIndex, ShouldEqual, uint64(1))
			So(lastLogTerm, ShouldEqual, uint64(1))

			// check with log
			var firstLog Log
			err = lMock.runner.logStore.GetLog(1, &firstLog)
			So(err, ShouldBeNil)

			So(firstLog.LastHash, ShouldBeNil)
			So(lastLogHash.IsEqual(&firstLog.Hash), ShouldBeTrue)
			So(lastLogIndex, ShouldResemble, firstLog.Index)
			So(lastLogTerm, ShouldResemble, firstLog.Term)

			// commit second log
			callOrder.Reset()

			_, _, err = lMock.runner.Apply(testPayload)

			So(err, ShouldBeNil)

			// test call orders
			So(callOrder.Get(), ShouldResemble, []string{
				"prepare",
				"prepare",
				"prepare",
				"commit",
				"commit",
				"commit",
			})

			// check with log
			var secondLog Log
			err = lMock.runner.logStore.GetLog(2, &secondLog)
			So(err, ShouldBeNil)

			So(secondLog.LastHash.IsEqual(lastLogHash), ShouldBeTrue)
		})

		Convey("rollback", func() {
			lMock := createMock("leader")
			f1Mock := createMock("follower1")
			f2Mock := createMock("follower2")

			// init
			initMock(lMock, f1Mock, f2Mock)

			testPayload := []byte("test data")

			callOrder := &CallCollector{}
			unknownErr := errors.New("unknown error")
			// f1 prepare with error
			f1Mock.worker.On("Prepare", mock.Anything, testPayload).
				Return(unknownErr).Run(func(args mock.Arguments) {
				callOrder.Append("prepare")
			})
			f1Mock.worker.On("Rollback", mock.Anything, testPayload).
				Return(nil).Run(func(args mock.Arguments) {
				callOrder.Append("rollback")
			})
			// f2 prepare with no error
			f2Mock.worker.On("Prepare", mock.Anything, testPayload).
				Return(nil).Run(func(args mock.Arguments) {
				callOrder.Append("prepare")
			})
			f2Mock.worker.On("Rollback", mock.Anything, testPayload).
				Return(nil).Run(func(args mock.Arguments) {
				callOrder.Append("rollback")
			})
			lMock.worker.On("Prepare", mock.Anything, testPayload).
				Return(nil).Run(func(args mock.Arguments) {
				callOrder.Append("prepare")
			})
			lMock.worker.On("Rollback", mock.Anything, testPayload).
				Return(nil).Run(func(args mock.Arguments) {
				callOrder.Append("rollback")
			})

			// try call process
			origLevel := log.GetLevel()
			log.SetLevel(log.DebugLevel)
			_, _, err := lMock.runner.Apply(testPayload)
			log.SetLevel(origLevel)

			So(err, ShouldNotBeNil)
			So(err, ShouldEqual, unknownErr)

			// test call orders
			// FIXME, one prepare error worker will not trigger rollback function on storage
			So(callOrder.Get(), ShouldResemble, []string{
				"prepare",
				"prepare",
				"prepare",
				"rollback",
				"rollback",
				//"rollback",
			})
		})
	})

	Convey("sybil test", t, func() {
		mockRouter.ResetAll()

		peers := testPeersFixture(1, []*Server{
			{
				Role: proto.Leader,
				ID:   "leader",
			},
			{
				Role: proto.Follower,
				ID:   "follower1",
			},
			{
				Role: proto.Follower,
				ID:   "follower2",
			},
		})
		initMock := func(mocks ...*createMockRes) {
			for _, r := range mocks {
				err := r.runner.Init(r.config, peers, r.logStore, r.stableStore, r.transport)
				So(err, ShouldBeNil)
			}
		}

		Convey("request from non-leader", func() {
			lMock := createMock("leader")
			f1Mock := createMock("follower1")
			f2Mock := createMock("follower1")

			// init
			initMock(lMock, f1Mock, f2Mock)

			// fake request
			testPayload := []byte("test data")
			fakeLog := &Log{
				Term:  1,
				Index: 1,
				Data:  testPayload,
			}

			var err error
			var rv []byte
			rv, err = f1Mock.transport.Request(
				context.Background(),
				f2Mock.config.LocalID,
				"Prepare",
				fakeLog,
			)

			So(rv, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldEqual, ErrInvalidRequest.Error())
		})

		Convey("send invalid request", func() {
			lMock := createMock("leader")
			f1Mock := createMock("follower1")

			// init
			initMock(lMock, f1Mock)

			// fake request
			var err error
			var rv []byte
			rv, err = lMock.transport.Request(
				context.Background(),
				f1Mock.config.LocalID,
				"invalid request",
				nil,
			)

			So(rv, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldEqual, ErrInvalidRequest.Error())
		})

		Convey("log could not be decoded", func() {
			lMock := createMock("leader")
			f1Mock := createMock("follower1")

			// init
			initMock(lMock, f1Mock)

			var err error
			var rv []byte
			rv, err = lMock.transport.Request(
				context.Background(),
				f1Mock.config.LocalID,
				"Prepare",
				nil,
			)

			So(rv, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err, ShouldEqual, ErrInvalidLog)
		})
	})
}

func TestTwoPCRunner_UpdatePeers(t *testing.T) {
	mockRouter := &MockTransportRouter{
		transports: make(map[proto.NodeID]*MockTransport),
	}

	type createMockRes struct {
		runner      *TwoPCRunner
		transport   *MockTransport
		worker      *MockWorker
		config      *TwoPCConfig
		logStore    *MockLogStore
		stableStore *MockStableStore
	}

	createMock := func(nodeID proto.NodeID) (res *createMockRes) {
		res = &createMockRes{}
		log.SetLevel(log.FatalLevel)
		res.runner = NewTwoPCRunner()
		res.transport = mockRouter.getTransport(nodeID)
		res.worker = &MockWorker{}
		res.config = &TwoPCConfig{
			RuntimeConfig: RuntimeConfig{
				RootDir:        "test_dir",
				LocalID:        nodeID,
				Runner:         res.runner,
				Transport:      res.transport,
				ProcessTimeout: time.Millisecond * 800,
			},
			Storage: res.worker,
		}
		res.logStore = &MockLogStore{}
		res.stableStore = &MockStableStore{}

		// init with no log and no term info
		res.stableStore.On("GetUint64", keyCurrentTerm).Return(uint64(0), nil)
		res.stableStore.On("GetUint64", keyCommittedIndex).Return(uint64(0), nil)
		res.stableStore.On("SetUint64", keyCurrentTerm, uint64(2)).Return(nil)
		res.logStore.On("LastIndex").Return(uint64(0), nil)
		return
	}
	peers := testPeersFixture(2, []*Server{
		{
			Role: proto.Leader,
			ID:   "leader",
		},
		{
			Role: proto.Follower,
			ID:   "follower1",
		},
		{
			Role: proto.Follower,
			ID:   "follower2",
		},
	})
	initMock := func(mocks ...*createMockRes) {
		for _, r := range mocks {
			err := r.runner.Init(r.config, peers, r.logStore, r.stableStore, r.transport)
			So(err, ShouldBeNil)
		}
	}
	testMock := func(peers *Peers, testFunc func(*createMockRes, error), mocks ...*createMockRes) {
		wg := new(sync.WaitGroup)

		for _, r := range mocks {
			wg.Add(1)
			go func(m *createMockRes) {
				defer wg.Done()
				err := m.runner.UpdatePeers(peers)
				if testFunc != nil {
					testFunc(m, err)
				}
			}(r)
		}

		wg.Wait()
	}

	Convey("update peers with invalid configuration", t, func() {
		mockRouter.ResetAll()

		lMock := createMock("leader")
		f1Mock := createMock("follower1")
		f2Mock := createMock("follower2")

		// init
		initMock(lMock, f1Mock, f2Mock)

		Convey("same peers term", FailureContinues, func(c C) {
			newPeers := testPeersFixture(2, []*Server{
				{
					Role: proto.Leader,
					ID:   "leader",
				},
				{
					Role: proto.Follower,
					ID:   "follower1",
				},
				{
					Role: proto.Follower,
					ID:   "follower2",
				},
			})

			testFunc := func(_ *createMockRes, err error) {
				c.So(err, ShouldBeNil)
			}
			testMock(newPeers, testFunc, lMock, f1Mock, f2Mock)
		})

		Convey("invalid peers term", FailureContinues, func(c C) {
			newPeers := testPeersFixture(1, []*Server{
				{
					Role: proto.Leader,
					ID:   "leader",
				},
				{
					Role: proto.Follower,
					ID:   "follower1",
				},
				{
					Role: proto.Follower,
					ID:   "follower2",
				},
			})

			testFunc := func(_ *createMockRes, err error) {
				c.So(err, ShouldNotBeNil)
				c.So(err, ShouldEqual, ErrInvalidConfig)
			}
			testMock(newPeers, testFunc, lMock, f1Mock, f2Mock)
		})

		Convey("invalid peers signature", FailureContinues, func(c C) {
			newPeers := testPeersFixture(4, []*Server{
				{
					Role: proto.Leader,
					ID:   "leader",
				},
				{
					Role: proto.Follower,
					ID:   "follower1",
				},
				{
					Role: proto.Follower,
					ID:   "follower2",
				},
			})

			newPeers.Term = 3

			testFunc := func(_ *createMockRes, err error) {
				c.So(err, ShouldNotBeNil)
				c.So(err, ShouldEqual, ErrInvalidConfig)
			}
			testMock(newPeers, testFunc, lMock, f1Mock, f2Mock)
		})

		Convey("peers update success", FailureContinues, func(c C) {
			updateMock := func(mocks ...*createMockRes) {
				for _, r := range mocks {
					r.stableStore.On("SetUint64", keyCurrentTerm, uint64(3)).Return(nil)
				}
			}

			updateMock(lMock, f1Mock, f2Mock)

			newPeers := testPeersFixture(3, []*Server{
				{
					Role: proto.Leader,
					ID:   "leader",
				},
				{
					Role: proto.Follower,
					ID:   "follower1",
				},
				{
					Role: proto.Follower,
					ID:   "follower2",
				},
			})

			testFunc := func(r *createMockRes, err error) {
				c.So(err, ShouldBeNil)
				c.So(r.runner.currentTerm, ShouldEqual, uint64(3))
				c.So(r.runner.peers, ShouldResemble, newPeers)
				r.stableStore.AssertCalled(t, "SetUint64", keyCurrentTerm, uint64(3))
			}

			testMock(newPeers, testFunc, lMock, f1Mock, f2Mock)
		})

		Convey("peers update include leader change", FailureContinues, func(c C) {
			updateMock := func(mocks ...*createMockRes) {
				for _, r := range mocks {
					r.stableStore.On("SetUint64", keyCurrentTerm, uint64(3)).Return(nil)
				}
			}

			updateMock(lMock, f1Mock, f2Mock)

			newPeers := testPeersFixture(3, []*Server{
				{
					Role: proto.Follower,
					ID:   "leader",
				},
				{
					Role: proto.Leader,
					ID:   "follower1",
				},
				{
					Role: proto.Follower,
					ID:   "follower2",
				},
			})

			testFunc := func(r *createMockRes, err error) {
				c.So(err, ShouldBeNil)
				c.So(r.runner.currentTerm, ShouldEqual, uint64(3))
				c.So(r.runner.peers, ShouldResemble, newPeers)

				switch r.config.LocalID {
				case "leader":
					c.So(r.runner.role, ShouldEqual, proto.Follower)
				case "follower1":
					c.So(r.runner.role, ShouldEqual, proto.Leader)
				case "follower2":
					c.So(r.runner.role, ShouldEqual, proto.Follower)
				}

				r.stableStore.AssertCalled(t, "SetUint64", keyCurrentTerm, uint64(3))
			}

			testMock(newPeers, testFunc, lMock, f1Mock, f2Mock)

			// test call process
			testPayload := []byte("test data")
			_, _, err := lMock.runner.Apply(testPayload)

			// no longer leader
			So(err, ShouldNotBeNil)
		})

		Convey("peers update with shutdown", FailureContinues, func(c C) {
			updateMock := func(mocks ...*createMockRes) {
				for _, r := range mocks {
					r.stableStore.On("SetUint64", keyCurrentTerm, uint64(3)).Return(nil)
				}
			}

			updateMock(lMock, f1Mock, f2Mock)

			newPeers := testPeersFixture(3, []*Server{
				{
					Role: proto.Leader,
					ID:   "leader",
				},
				{
					Role: proto.Follower,
					ID:   "follower1",
				},
			})

			testFunc := func(r *createMockRes, err error) {
				c.So(err, ShouldBeNil)
				c.So(r.runner.currentTerm, ShouldEqual, uint64(3))
				c.So(r.runner.peers, ShouldResemble, newPeers)
				r.stableStore.AssertCalled(t, "SetUint64", keyCurrentTerm, uint64(3))
			}

			testMock(newPeers, testFunc, lMock, f1Mock, f2Mock)

			So(f2Mock.runner.currentState, ShouldEqual, Shutdown)
		})
	})
}
