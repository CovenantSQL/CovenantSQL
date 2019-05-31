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
	"context"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
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

		ctx := env.GetContext()
		So(ctx, ShouldEqual, context.Background())
		cldCtx, cancel := context.WithCancel(ctx)
		defer cancel()
		env.SetContext(cldCtx)
		So(env.GetContext(), ShouldEqual, cldCtx)
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
			dbID := DatabaseID(target[i])
			h, err := hash.NewHashFromStr(target[i])
			So(err, ShouldBeNil)
			a, err := dbID.AccountAddress()
			So(err, ShouldBeNil)
			So(h[:], ShouldResemble, a[:])
		}
	})

	Convey("AccountAddress invalid convert", t, func() {
		invalid := "invalid"
		dbID := DatabaseID(invalid)
		_, err := dbID.AccountAddress()
		So(err, ShouldNotBeNil)
	})
}

func TestFromAccountAndNonce(t *testing.T) {
	target := []struct {
		account string
		nonce   uint32
		result  string
	}{
		{
			"aecd0b238518f023a81db3bdce5b2f1211b4e383dac96efc5d7205c761e15519",
			1,
			"a2710c0f9550c2713e9ff25543d9b5ef96bdc4927ea8d56262bb938718b8b717",
		},
		{
			"8374e39c82c847681341245d9964f31a15cf2844994ec64a9b99c38dd3b7f54b",
			2,
			"da05187b8da4ec8dc3320c7d103590573d4f40beed414ad2afed7c3bdbaca628",
		},
		{
			"72cb315f225c8ce03cd4b8d8ef802e871f9aa0924ddd9a358b647f4187730b12",
			0,
			"293e94eecddd941eab73d47e0cb967102b271bd4fc3a1398c1c1496c3dd06f0f",
		},
	}
	Convey("Generate DatabaseID", t, func() {
		for i := range target {
			h, err := hash.NewHashFromStr(target[i].account)
			So(err, ShouldBeNil)
			a := AccountAddress(*h)
			dbID := FromAccountAndNonce(a, target[i].nonce)
			So(string(dbID), ShouldResemble, target[i].result)
		}
	})
}
