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
	"testing"
	. "github.com/smartystreets/goconvey/convey"
	"time"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"github.com/stretchr/testify/mock"
	"errors"
)

func testConfig(rootDir string, serverID ServerID) Config {
	config := &MockConfig{}
	logger := log.New()
	logger.SetLevel(log.FatalLevel)

	runtimeConfig := &RuntimeConfig{
		RootDir:        rootDir,
		LocalID:        serverID,
		Runner:         &MockRunner{},
		Transport:      &MockTransport{},
		ProcessTimeout: time.Microsecond * 800,
		AutoBanCount:   100,
		Logger:         logger,
	}

	config.On("GetRuntimeConfig").Return(runtimeConfig)

	return config
}

func TestNewRuntime(t *testing.T) {
	Convey("new runtime", t, func() {
		config := testConfig(".", "leader")
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
			runner.On("Process", mock.Anything).Return(nil)

			err = r.Process([]byte("test"))
			So(err, ShouldBeNil)

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
		runner.On("Process", mock.Anything).Return(nil)

		err = r.Init()
		So(err, ShouldBeNil)
		defer r.Shutdown()

		err = r.Process([]byte("test"))
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

			newPeers.Term = 5

			// not valid
			err := r.UpdatePeers(newPeers)

			So(err, ShouldNotBeNil)
			So(err, ShouldEqual, ErrInvalidConfig)
		})

		Convey("change leader", func() {
			newPeers := testPeersFixture(3, []*Server{
				{
					Role:    Follower,
					ID:      "leader",
					Address: "leader_address",
				},
				{
					Role:    Leader,
					ID:      "follower1",
					Address: "follower_address",
				},
				{
					Role:    Follower,
					ID:      "follower2",
					Address: "follower2_address",
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
					Role:    Leader,
					ID:      "follower1",
					Address: "follower_address",
				},
				{
					Role:    Follower,
					ID:      "follower2",
					Address: "follower2_address",
				},
			})

			// valid
			err := r.UpdatePeers(newPeers)

			So(err, ShouldBeNil)
			runner.AssertCalled(t, "Shutdown", true)
		})
	})
}
