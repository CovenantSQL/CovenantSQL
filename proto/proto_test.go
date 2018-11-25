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
	"time"

	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	. "github.com/smartystreets/goconvey/convey"
)

func TestEnvelope_GetSet(t *testing.T) {
	Convey("set get", t, func() {
		env := Envelope{}
		env.SetExpire(time.Second)
		So(env.GetExpire(), ShouldEqual, time.Second)

		nodeID := &RawNodeID{
			Hash: hash.Hash{0xa, 0xa},
		}
		env.SetNodeID(nodeID)
		So(env.GetNodeID(), ShouldEqual, nodeID)

		env.SetTTL(time.Second)
		So(env.GetTTL(), ShouldEqual, time.Second)

		env.SetVersion("0.0.1")
		So(env.GetVersion(), ShouldEqual, "0.0.1")
	})
}

func TestDatabaseID_AccountAddress(t *testing.T) {
	target := []string{
		"1224a1e9f72eb00d08afa4030dc642edefb6e3249aafe20cf1a5f9d46d0c0bbe",
		"5b0b8fd3b0700bd0858f3d61ff0a1b621dbbeb2013a3aab5df2885dc10ccf6ce",
		"b90f502d8aa95573cdc3c50ea1552aa1c163b567980e2555fe84cfd1d5e78765",
	}

	Convey("AccountAddress convert", t, func() {
		for i := range target {
			dbid := DatabaseID(target[i])
			h, err := hash.NewHashFromStr(target[i])
			So(err, ShouldBeNil)
			a, err := dbid.AccountAddress()
			So(err, ShouldBeNil)
			So(h[:], ShouldResemble, a[:])
		}

	})
}
