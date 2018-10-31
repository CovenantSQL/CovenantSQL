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

package utils

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGetRandomPorts(t *testing.T) {
	Convey("get 1 port", t, func() {
		ports, err := GetRandomPorts("127.0.0.1", 1, 10000, 1)
		So(ports, ShouldHaveLength, 1)
		So(err, ShouldBeNil)
	})

	// make one unusable port here
	net.Listen("tcp", "127.0.0.1:2001")
	Convey("get too many ports", t, func() {
		ports, err := GetRandomPorts("127.0.0.1", 2000, 2010, 100)
		So(len(ports), ShouldBeBetween, 0, 12)
		So(err, ShouldEqual, ErrNotEnoughPorts)
	})

	Convey("port range error", t, func() {
		ports, err := GetRandomPorts("127.0.0.1", 3000, 2010, 1)
		So(ports, ShouldBeEmpty)
		So(err, ShouldEqual, ErrNotEnoughPorts)
	})

	Convey("port range start 0", t, func() {
		ports, err := GetRandomPorts("127.0.0.1", 0, 65535, 1)
		So(ports, ShouldHaveLength, 1)
		So(err, ShouldBeNil)
	})

	Convey("port range error", t, func() {
		ports, _ := GetRandomPorts("127.0.0.1", 0, 65535, 0)
		So(ports, ShouldBeEmpty)
	})

	Convey("previously allocated should be unavailable", t, func() {
		ports, err := GetRandomPorts("127.0.0.1", 1, 10000, 1)
		So(ports, ShouldHaveLength, 1)
		So(err, ShouldBeNil)
		lastAllocated := ports[0]
		ports, err = GetRandomPorts("127.0.0.1", lastAllocated, lastAllocated, 1)
		So(err, ShouldNotBeNil)
		// any address should be banned to allocate too
		ports, err = GetRandomPorts("0.0.0.0", lastAllocated, lastAllocated, 1)
		So(err, ShouldNotBeNil)
		// but another bind address should be available, should be use 127.0.0.2, but mac does not support 127.0.0.2
		ports, err = GetRandomPorts("", lastAllocated, lastAllocated, 1)
		So(err, ShouldBeNil)
		So(ports, ShouldHaveLength, 1)
		So(ports[0], ShouldEqual, lastAllocated)
	})
}

func TestWaitForPorts(t *testing.T) {
	Convey("test wait for ports", t, func() {
		ports, err := GetRandomPorts("127.0.0.1", 1, 10000, 1)
		So(ports, ShouldHaveLength, 1)
		So(err, ShouldBeNil)

		err = WaitForPorts(context.Background(), "127.0.0.1", ports, time.Millisecond*100)
		So(err, ShouldBeNil)

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*300)
		defer cancel()
		err = WaitToConnect(ctx, "127.0.0.1", ports, time.Millisecond*100)
		So(err, ShouldNotBeNil)

		// listen
		ln, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", fmt.Sprint(ports[0])))
		So(ln, ShouldNotBeNil)
		So(err, ShouldBeNil)
		defer ln.Close()

		err = WaitToConnect(context.Background(), "127.0.0.1", ports, time.Millisecond*100)
		So(err, ShouldBeNil)

		ctx, cancel = context.WithTimeout(context.Background(), time.Millisecond*300)
		defer cancel()
		err = WaitForPorts(ctx, "127.0.0.1", ports, time.Millisecond*100)
		So(err, ShouldNotBeNil)

		go func() {
			time.Sleep(time.Millisecond * 100)
			ln.Close()
		}()

		ctx, cancel = context.WithTimeout(context.Background(), time.Millisecond*500)
		defer cancel()
		err = WaitForPorts(ctx, "127.0.0.1", ports, time.Millisecond*100)
		So(err, ShouldBeNil)
	})
}
