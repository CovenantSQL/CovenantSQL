/*
 * Copyright 2018 The ThunderDB Authors.
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
	"testing"

	"net"

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
		So(err, ShouldEqual, NotEnoughPorts)
	})

	Convey("port range error", t, func() {
		ports, err := GetRandomPorts("127.0.0.1", 3000, 2010, 1)
		So(ports, ShouldBeEmpty)
		So(err, ShouldEqual, NotEnoughPorts)
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
}
