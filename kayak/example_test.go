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
	"io/ioutil"
	"os"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/mock"
	"gitlab.com/thunderdb/ThunderDB/proto"
)

func TestExampleTwoPCCommit(t *testing.T) {
	type createMockRes struct {
		runner    *TwoPCRunner
		transport *MockTransport
		worker    *MockWorker
		config    *TwoPCConfig
		runtime   *Runtime
	}

	// codec to encode/decode data in kayak.Log structure
	mockLogCodec := &MockLogCodec{}
	// router is a dummy channel based local rpc transport router
	mockRouter := &MockTransportRouter{
		transports: make(map[proto.NodeID]*MockTransport),
	}
	// peers is a simple 3-node peer configuration
	peers := testPeersFixture(1, []*Server{
		{
			Role: Leader,
			ID:   "leader",
		},
		{
			Role: Follower,
			ID:   "follower1",
		},
		{
			Role: Follower,
			ID:   "follower2",
		},
	})
	// create mock returns basic arguments to prepare for a server
	createMock := func(nodeID proto.NodeID) (res *createMockRes) {
		res = &createMockRes{}
		logger := log.New()
		logger.SetLevel(log.FatalLevel)
		d, _ := ioutil.TempDir("", "kayak_test")

		// runner instance
		res.runner = NewTwoPCRunner()
		// transport for this instance
		res.transport = mockRouter.getTransport(nodeID)
		// underlying worker
		res.worker = &MockWorker{}
		// runner config including timeout settings, commit log storage, local server id
		res.config = &TwoPCConfig{
			RuntimeConfig: RuntimeConfig{
				RootDir:        d,
				LocalID:        nodeID,
				Runner:         res.runner,
				Transport:      res.transport,
				ProcessTimeout: time.Millisecond * 800,
				Logger:         logger,
			},
			LogCodec:        mockLogCodec,
			Storage:         res.worker,
			PrepareTimeout:  time.Millisecond * 200,
			CommitTimeout:   time.Millisecond * 200,
			RollbackTimeout: time.Millisecond * 200,
		}
		res.runtime, _ = NewRuntime(res.config, peers)
		return
	}
	// cleanup log storage after execution
	cleanupDir := func(c *createMockRes) {
		os.RemoveAll(c.config.RuntimeConfig.RootDir)
	}

	// only commit logic
	Convey("commit", t, func() {
		var err error

		lMock := createMock("leader")
		f1Mock := createMock("follower1")
		f2Mock := createMock("follower2")
		defer cleanupDir(lMock)
		defer cleanupDir(f1Mock)
		defer cleanupDir(f2Mock)

		// init
		err = lMock.runtime.Init()
		So(err, ShouldBeNil)
		err = f1Mock.runtime.Init()
		So(err, ShouldBeNil)
		err = f2Mock.runtime.Init()
		So(err, ShouldBeNil)

		// payload to send
		testPayload := "test data"
		testData, _ := mockLogCodec.Encode(testPayload)

		// underlying worker mock, prepare/commit/rollback with be received the decoded data
		callOrder := &CallCollector{}
		f1Mock.worker.On("Prepare", mock.Anything, testPayload).
			Return(nil).Run(func(args mock.Arguments) {
			callOrder.Append("f_prepare")
		})
		f2Mock.worker.On("Prepare", mock.Anything, testPayload).
			Return(nil).Run(func(args mock.Arguments) {
			callOrder.Append("f_prepare")
		})
		f1Mock.worker.On("Commit", mock.Anything, testPayload).
			Return(nil).Run(func(args mock.Arguments) {
			callOrder.Append("f_commit")
		})
		f2Mock.worker.On("Commit", mock.Anything, testPayload).
			Return(nil).Run(func(args mock.Arguments) {
			callOrder.Append("f_commit")
		})
		lMock.worker.On("Prepare", mock.Anything, testPayload).
			Return(nil).Run(func(args mock.Arguments) {
			callOrder.Append("l_prepare")
		})
		lMock.worker.On("Commit", mock.Anything, testPayload).
			Return(nil).Run(func(args mock.Arguments) {
			callOrder.Append("l_commit")
		})

		// process the encoded data
		err = lMock.runtime.Apply(testData)
		So(err, ShouldBeNil)
		So(callOrder.Get(), ShouldResemble, []string{
			"f_prepare",
			"f_prepare",
			"l_prepare",
			"f_commit",
			"f_commit",
			"l_commit",
		})

		// process the encoded data again
		callOrder.Reset()
		err = lMock.runner.Apply(testData)
		So(err, ShouldBeNil)
		So(callOrder.Get(), ShouldResemble, []string{
			"f_prepare",
			"f_prepare",
			"l_prepare",
			"f_commit",
			"f_commit",
			"l_commit",
		})

		// shutdown
		lMock.runtime.Shutdown()
		f1Mock.runtime.Shutdown()
		f2Mock.runtime.Shutdown()
	})
}
