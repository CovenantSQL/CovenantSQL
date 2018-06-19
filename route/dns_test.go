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
	"github.com/thunderdb/ThunderDB/crypto/hash"
	"github.com/thunderdb/ThunderDB/proto"
)

const addr = "127.0.0.1:22"

func TestResolver(t *testing.T) {
	Convey("resolver init", t, func() {
		InitResolver()
		InitResolveCache(make(ResolveCache))
		addr, err := GetNodeAddr(&proto.RawNodeID{
			Hash: hash.Hash([32]byte{0xde, 0xad}),
		})
		So(err, ShouldEqual, ErrUnknownNodeID)
		So(addr, ShouldBeBlank)

		addr, err = GetNodeAddr(nil)
		So(err, ShouldEqual, ErrNilNodeID)
		So(addr, ShouldBeBlank)

		err = SetNodeAddr(nil, addr)
		So(err, ShouldEqual, ErrNilNodeID)

		nodeA := &proto.RawNodeID{
			Hash: hash.Hash([32]byte{0xaa, 0xaa}),
		}
		err = SetNodeAddr(nodeA, addr)
		So(err, ShouldBeNil)

		addr, err = GetNodeAddr(nodeA)
		So(err, ShouldBeNil)
		So(addr, ShouldEqual, addr)

		So(IsBPNodeID(nil), ShouldBeFalse)

		So(IsBPNodeID(nodeA), ShouldBeFalse)

		So(GetBPAddr(), ShouldNotBeNil)
	})
}
