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

package kms

import (
	"bytes"
	"testing"

	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	mine "github.com/CovenantSQL/CovenantSQL/pow/cpuminer"
	"github.com/CovenantSQL/CovenantSQL/proto"
	. "github.com/smartystreets/goconvey/convey"
)

func TestLocalKeyStore(t *testing.T) {
	Convey("set and get key", t, func() {
		initLocalKeyStore()
		So(localKey, ShouldNotBeNil)
		gotPrivate, err := GetLocalPrivateKey()
		So(gotPrivate, ShouldBeNil)
		So(err, ShouldEqual, ErrNilField)
		gotPublic, err := GetLocalPublicKey()
		So(gotPublic, ShouldBeNil)
		So(err, ShouldEqual, ErrNilField)

		privKey1, pubKey1, _ := asymmetric.GenSecp256k1KeyPair()
		privKey2, pubKey2, _ := asymmetric.GenSecp256k1KeyPair()
		SetLocalKeyPair(privKey1, pubKey1)
		SetLocalKeyPair(privKey2, pubKey2) // no effect
		gotPrivate, err = GetLocalPrivateKey()
		So(err, ShouldBeNil)
		gotPublic, err = GetLocalPublicKey()
		So(err, ShouldBeNil)
		So(bytes.Compare(gotPrivate.Serialize(), privKey1.Serialize()), ShouldBeZeroValue)
		So(gotPublic.IsEqual(pubKey1), ShouldBeTrue)
		So(gotPrivate.PubKey().IsEqual(pubKey1), ShouldBeTrue)
	})
	Convey("set and get key", t, func() {
		initLocalKeyStore()
		So(localKey, ShouldNotBeNil)
		gotID, err := GetLocalNodeIDBytes()
		So(gotID, ShouldBeNil)
		So(err, ShouldEqual, ErrNilField)
		gotNonce, err := GetLocalNonce()
		So(gotNonce, ShouldBeNil)
		So(err, ShouldEqual, ErrNilField)
		SetLocalNodeIDNonce([]byte("aaa"), nil)
		SetLocalNodeIDNonce([]byte("aaa"), &mine.Uint256{A: 1, B: 1, C: 1, D: 1})
		gotID, err = GetLocalNodeIDBytes()
		So(bytes.Compare(gotID, []byte("aaa")), ShouldBeZeroValue)
		So(err, ShouldBeNil)
		gotNonce, err = GetLocalNonce()
		So(*gotNonce == mine.Uint256{A: 1, B: 1, C: 1, D: 1}, ShouldBeTrue)
		So(err, ShouldBeNil)
		rawNodeID := &proto.RawNodeID{}
		nodeIDBefore := rawNodeID.ToNodeID()
		SetLocalNodeIDNonce(nodeIDBefore.ToRawNodeID().CloneBytes(), nil)
		nodeID, err := GetLocalNodeID()
		So(nodeID, ShouldResemble, nodeIDBefore)
	})
}
