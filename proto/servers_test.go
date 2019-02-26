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

package proto

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/utils"
)

func TestPeers(t *testing.T) {
	Convey("test peers", t, func() {
		privKey, _, err := asymmetric.GenSecp256k1KeyPair()
		So(err, ShouldBeNil)
		p := &Peers{
			PeersHeader: PeersHeader{
				Term:   1,
				Leader: NodeID("00000bef611d346c0cbe1beaa76e7f0ed705a194fdf9ac3a248ec70e9c198bf9"),
				Servers: []NodeID{
					NodeID("00000bef611d346c0cbe1beaa76e7f0ed705a194fdf9ac3a248ec70e9c198bf9"),
					NodeID("00000381d46fd6cf7742d7fb94e2422033af989c0e348b5781b3219599a3af35"),
				},
			},
		}
		err = p.Sign(privKey)
		So(err, ShouldBeNil)
		err = p.Verify()
		So(err, ShouldBeNil)

		// after encode/decode
		buf, err := utils.EncodeMsgPack(p)
		var peers *Peers
		err = utils.DecodeMsgPack(buf.Bytes(), &peers)
		So(err, ShouldBeNil)
		err = peers.Verify()
		So(err, ShouldBeNil)

		peers2 := peers.Clone()
		err = peers2.Verify()
		So(err, ShouldBeNil)

		i, found := peers.Find(NodeID("00000381d46fd6cf7742d7fb94e2422033af989c0e348b5781b3219599a3af35"))
		So(i, ShouldEqual, 1)
		So(found, ShouldBeTrue)

		i, found = peers.Find(NodeID("0000000000000000000000000000000000000000000000000000000000000001"))
		So(found, ShouldBeFalse)

		// verify hash failed
		peers.Term = 2
		err = peers.Verify()
		So(err, ShouldNotBeNil)
		err = peers.Sign(privKey)
		So(err, ShouldBeNil)

		// verify failed
		p.Signature = peers.Signature
		err = p.Verify()
		So(err, ShouldNotBeNil)
	})
}
