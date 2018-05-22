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

package kms

import (
	"testing"

	"os"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/thunderdb/ThunderDB/crypto/asymmetric"
	"github.com/thunderdb/ThunderDB/proto"
)

const dbFile = ".test.db"

func TestDB(t *testing.T) {
	privKey1, pubKey1, _ := asymmetric.GenSecp256k1Keypair()
	privKey2, pubKey2, _ := asymmetric.GenSecp256k1Keypair()
	Convey("Init db", t, func() {
		defer os.Remove(dbFile)
		pks, err := NewPublicKeyStore(dbFile)
		So(pks.bucket, ShouldNotBeNil)
		So(err, ShouldBeNil)

		pubk, err := pks.GetPublicKey(proto.NodeID("not exist"))
		So(pubk, ShouldBeNil)
		So(err, ShouldEqual, ErrKeyNotFound)

		err = pks.SetPublicKey(proto.NodeID("node1"), pubKey1)
		So(err, ShouldBeNil)

		err = pks.SetPublicKey(proto.NodeID("node2"), pubKey2)
		So(err, ShouldBeNil)

		pubk, err = pks.GetPublicKey(proto.NodeID("node1"))
		So(pubk, ShouldNotBeNil)
		So(err, ShouldBeNil)
		So(privKey1.PubKey().IsEqual(pubKey1), ShouldBeTrue)

		pubk, err = pks.GetPublicKey(proto.NodeID("node2"))
		So(pubk, ShouldNotBeNil)
		So(err, ShouldBeNil)
		So(privKey2.PubKey().IsEqual(pubKey2), ShouldBeTrue)

		err = pks.DelPublicKey(proto.NodeID("node2"))
		So(err, ShouldBeNil)

		err = pks.DelPublicKey(proto.NodeID("node2"))
		So(err, ShouldBeNil)

		pubk, err = pks.GetPublicKey(proto.NodeID("node2"))
		So(pubk, ShouldBeNil)
		So(err, ShouldEqual, ErrKeyNotFound)

		err = pks.RemoveBucket()
		So(err, ShouldBeNil)

		pubk, err = pks.GetPublicKey(proto.NodeID("not exist"))
		So(pubk, ShouldBeNil)
		So(err, ShouldEqual, ErrBucketNotInitialized)

		err = pks.SetPublicKey(proto.NodeID("node1"), pubKey1)
		So(err, ShouldEqual, ErrBucketNotInitialized)

		err = pks.DelPublicKey(proto.NodeID("node2"))
		So(err, ShouldEqual, ErrBucketNotInitialized)
	})
}

func TestErrorPath(t *testing.T) {
	Convey("can not init db", t, func() {
		pks, err := NewPublicKeyStore("/path/not/exist")
		So(pks, ShouldBeNil)
		So(err, ShouldNotBeNil)
	})
}
