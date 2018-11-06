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
	"errors"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/mock"
)

func testConfig(rootDir string, nodeID proto.NodeID) Config {
	config := &MockConfig{}
	log.SetLevel(log.FatalLevel)

	runtimeConfig := &RuntimeConfig{
		RootDir:        rootDir,
		LocalID:        nodeID,
		Runner:         &MockRunner{},
		Transport:      &MockTransport{},
		ProcessTimeout: time.Microsecond * 800,
		AutoBanCount:   100,
	}

	config.On("GetRuntimeConfig").Return(runtimeConfig)

	return config
}

func TestNewRuntime(t *testing.T) {
	Convey("new runtime", t, func() {
		config := testConfig(".", "leader")
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

		Convey("missing arguments", func() {
			var r *Runtime
			var err error

			r, err = NewRuntime(nil, nil)
			So(r, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err, ShouldEqual, ErrInvalidConfig)

			r, err = NewRuntime(config, nil)
			So(r, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err, ShouldEqual, ErrInvalidConfig)

			r, err = NewRuntime(nil, peers)
			So(r, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err, ShouldEqual, ErrInvalidConfig)
		})

		Convey("invalid peer", func() {
			newPeers := peers.Clone()
			// change peer signature
			newPeers.Term = 3

			r, err := NewRuntime(config, &newPeers)
			So(r, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err, ShouldEqual, ErrInvalidConfig)
		})

		Convey("server not in peers", func() {
			newConfig := testConfig(".", "test2")

			r, err := NewRuntime(newConfig, peers)

			So(r, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err, ShouldEqual, ErrInvalidConfig)
		})

		Convey("success", func() {
			r, err := NewRuntime(config, peers)

			So(r, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(r.isLeader, ShouldBeTrue)
		})

		Convey("success with follower", func() {
			newConfig := testConfig(".", "follower1")
			r, err := NewRuntime(newConfig, peers)

			So(r, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(r.isLeader, ShouldBeFalse)
		})
	})
}

func TestRuntimeAll(t *testing.T) {
	Convey("init", t, func() {
		d, err := ioutil.TempDir("", "kayak_test")
		So(err, ShouldBeNil)
		if err == nil {
			defer os.RemoveAll(d)
		}

		config := testConfig(d, "leader")
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

		r, err := NewRuntime(config, peers)
		So(err, ShouldBeNil)

		runner := config.GetRuntimeConfig().Runner.(*MockRunner)

		Convey("runner init failed", func() {
			unknownErr := errors.New("unknown error")
			runner.On("Init",
				mock.Anything, // config
				mock.Anything, // peers
				mock.Anything, // logStore
				mock.Anything, // stableStore
				mock.Anything, // transport
			).Return(unknownErr)

			err := r.Init()

			So(err, ShouldNotBeNil)
			So(r.logStore, ShouldBeNil)
		})

		Convey("runner init success", func() {
			runner.On("Init",
				mock.Anything, // config
				mock.Anything, // peers
				mock.Anything, // logStore
				mock.Anything, // stableStore
				mock.Anything, // transport
			).Return(nil)
			runner.On("Shutdown", mock.Anything).
				Return(nil)

			var err error
			err = r.Init()
			So(err, ShouldBeNil)
			So(r.logStore, ShouldNotBeNil)

			// run process
			runner.On("Apply", mock.Anything).Return(nil, uint64(1), nil)

			_, _, err = r.Apply([]byte("test"))
			So(err, ShouldBeNil)

			// test get log
			var l Log
			l.Data = []byte("test")
			l.Index = uint64(1)
			err = r.logStore.StoreLog(&l)
			So(err, ShouldBeNil)

			data, err := r.GetLog(1)
			So(err, ShouldBeNil)
			So(data, ShouldResemble, []byte("test"))

			// call shutdowns
			err = r.Shutdown()
			So(err, ShouldBeNil)
		})
	})

	Convey("init success with follower", t, func() {
		d, err := ioutil.TempDir("", "kayak_test")
		So(err, ShouldBeNil)
		if err == nil {
			defer os.RemoveAll(d)
		}

		config := testConfig(d, "follower1")
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

		r, err := NewRuntime(config, peers)
		So(err, ShouldBeNil)

		runner := config.GetRuntimeConfig().Runner.(*MockRunner)
		runner.On("Init",
			mock.Anything, // config
			mock.Anything, // peers
			mock.Anything, // logStore
			mock.Anything, // stableStore
			mock.Anything, // transport
		).Return(nil)
		runner.On("Shutdown", mock.Anything).
			Return(nil)
		runner.On("Apply", mock.Anything).Return(nil, uint64(1), nil)

		err = r.Init()
		So(err, ShouldBeNil)
		defer r.Shutdown()

		_, _, err = r.Apply([]byte("test"))
		So(err, ShouldNotBeNil)
		So(err, ShouldEqual, ErrNotLeader)
	})

	Convey("init success with peers update", t, func() {
		d, err := ioutil.TempDir("", "kayak_test")
		So(err, ShouldBeNil)
		if err == nil {
			defer os.RemoveAll(d)
		}

		config := testConfig(d, "leader")
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

		r, err := NewRuntime(config, peers)
		So(err, ShouldBeNil)

		runner := config.GetRuntimeConfig().Runner.(*MockRunner)
		runner.On("Init",
			mock.Anything, // config
			mock.Anything, // peers
			mock.Anything, // logStore
			mock.Anything, // stableStore
			mock.Anything, // transport
		).Return(nil)
		runner.On("Shutdown", mock.Anything).Return(nil)
		runner.On("UpdatePeers", mock.Anything).Return(nil)

		err = r.Init()
		So(err, ShouldBeNil)
		defer r.Shutdown()

		Convey("invalid peers", func() {
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

			newPeers.Term = 5

			// not valid
			err := r.UpdatePeers(newPeers)

			So(err, ShouldNotBeNil)
			So(err, ShouldEqual, ErrInvalidConfig)
		})

		Convey("change leader", func() {
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

			// valid
			err := r.UpdatePeers(newPeers)

			So(err, ShouldBeNil)
			So(r.isLeader, ShouldBeFalse)
		})

		Convey("dropped peer", func() {
			newPeers := testPeersFixture(3, []*Server{
				{
					Role: proto.Leader,
					ID:   "follower1",
				},
				{
					Role: proto.Follower,
					ID:   "follower2",
				},
			})

			// valid
			err := r.UpdatePeers(newPeers)

			So(err, ShouldBeNil)
			runner.AssertCalled(t, "Shutdown", true)
		})
	})
}
