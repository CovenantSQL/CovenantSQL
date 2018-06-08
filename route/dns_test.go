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

package route

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/thunderdb/ThunderDB/proto"
)

const addr = "127.0.0.1:22"

func TestResolver(t *testing.T) {
	Convey("resolver init", t, func() {
		InitResolver()
		InitResolveCache(make(map[proto.NodeID]string))
		addr, err := GetNodeAddr(proto.NodeID("no exist"))
		So(err, ShouldEqual, ErrUnknownNodeID)
		So(addr, ShouldBeBlank)
		err = SetNodeAddr(proto.NodeID("aaa"), addr)
		So(err, ShouldBeNil)

		addr, err = GetNodeAddr(proto.NodeID("aaa"))
		So(err, ShouldBeNil)
		So(addr, ShouldEqual, addr)

		So(IsBPNodeID(proto.NodeID("aa")), ShouldBeFalse)

		So(GetBPAddr(), ShouldNotBeNil)
	})
}
