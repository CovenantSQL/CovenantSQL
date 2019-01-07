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
	"os"
	"reflect"
	"testing"

	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/pow/cpuminer"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/pkg/errors"
	. "github.com/smartystreets/goconvey/convey"
	yaml "gopkg.in/yaml.v2"
)

const dbFile = ".test.db"

func TestDB(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	privKey1, pubKey1, _ := asymmetric.GenSecp256k1KeyPair()
	privKey2, pubKey2, _ := asymmetric.GenSecp256k1KeyPair()
	node1 := &proto.Node{
		ID:        proto.NodeID("1111"),
		Addr:      "",
		PublicKey: pubKey1,
		Nonce:     cpuminer.Uint256{},
	}
	node2 := &proto.Node{
		ID:        proto.NodeID("2222"),
		Addr:      "",
		PublicKey: pubKey2,
		Nonce:     cpuminer.Uint256{},
	}
	BPNode := &proto.Node{
		ID:        BP.NodeID,
		Addr:      "",
		PublicKey: BP.PublicKey,
		Nonce:     BP.Nonce,
	}

	Convey("Init db", t, func() {
		pks = nil
		os.Remove(dbFile)
		defer os.Remove(dbFile)
		InitPublicKeyStore(dbFile, []proto.Node{*BPNode})
		So(pks.bucket, ShouldNotBeNil)

		nodeInfo, err := GetNodeInfo(BP.NodeID)
		log.Debugf("nodeInfo %v", nodeInfo)
		pubk, err := GetPublicKey(BP.NodeID)
		So(pubk, ShouldNotBeNil)
		So(err, ShouldBeNil)
		So(pubk.IsEqual(BP.PublicKey), ShouldBeTrue)

		pubk, err = GetPublicKey(proto.NodeID("99999999"))
		So(pubk, ShouldBeNil)
		So(errors.Cause(err), ShouldEqual, ErrKeyNotFound)

		err = SetNode(nil)
		So(err, ShouldEqual, ErrNilNode)

		err = setNode(node1)
		So(err, ShouldBeNil)

		err = setNode(node2)
		So(err, ShouldBeNil)

		err = SetPublicKey(BP.NodeID, BP.Nonce, BP.PublicKey)
		So(err, ShouldBeNil)

		err = SetPublicKey(BP.NodeID, cpuminer.Uint256{}, BP.PublicKey)
		So(err, ShouldEqual, ErrNodeIDKeyNonceNotMatch)

		err = SetPublicKey(proto.NodeID("00"+BP.NodeID), BP.Nonce, BP.PublicKey)
		So(err, ShouldEqual, ErrNodeIDKeyNonceNotMatch)

		pubk, err = GetPublicKey(proto.NodeID("1111"))
		So(pubk, ShouldNotBeNil)
		So(err, ShouldBeNil)
		So(privKey1.PubKey().IsEqual(pubKey1), ShouldBeTrue)

		pubk, err = GetPublicKey(proto.NodeID("2222"))
		So(pubk, ShouldNotBeNil)
		So(err, ShouldBeNil)
		So(privKey2.PubKey().IsEqual(pubKey2), ShouldBeTrue)

		IDs, err := GetAllNodeID()
		So(err, ShouldBeNil)
		So(IDs, ShouldHaveLength, 3)
		So(IDs, ShouldContain, proto.NodeID("1111"))
		So(IDs, ShouldContain, proto.NodeID("2222"))
		So(IDs, ShouldContain, BP.NodeID)

		err = DelNode(proto.NodeID("2222"))
		So(err, ShouldBeNil)

		err = DelNode(proto.NodeID("2222"))
		So(err, ShouldBeNil)

		pubk, err = GetPublicKey(proto.NodeID("2222"))
		So(pubk, ShouldBeNil)
		So(errors.Cause(err), ShouldEqual, ErrKeyNotFound)

		err = removeBucket()
		So(err, ShouldBeNil)

		pubk, err = GetPublicKey(proto.NodeID("not exist"))
		So(pubk, ShouldBeNil)
		So(errors.Cause(err), ShouldEqual, ErrBucketNotInitialized)

		err = setNode(node1)
		So(errors.Cause(err), ShouldEqual, ErrBucketNotInitialized)

		err = DelNode(proto.NodeID("2222"))
		So(errors.Cause(err), ShouldEqual, ErrBucketNotInitialized)

		IDs, err = GetAllNodeID()
		So(IDs, ShouldBeNil)
		So(errors.Cause(err), ShouldEqual, ErrBucketNotInitialized)

		err = ResetBucket()
		So(err, ShouldBeNil)

		pubk, err = GetPublicKey(proto.NodeID("2222"))
		So(pubk, ShouldBeNil)
		So(errors.Cause(err), ShouldEqual, ErrKeyNotFound)

		IDs, err = GetAllNodeID()
		So(IDs, ShouldBeNil)
		So(err, ShouldBeNil)
	})
}

func TestErrorPath(t *testing.T) {
	Convey("can not init db", t, func() {
		pks = nil
		err := InitPublicKeyStore("/path/not/exist", nil)
		So(pks, ShouldBeNil)
		So(err, ShouldNotBeNil)
	})
}

func TestMarshalNode(t *testing.T) {
	Convey("marshal unmarshal node", t, func() {
		nodeInfo := &proto.Node{
			ID:        "0000000000000000000000000000000000000000000000000000000000001111",
			Addr:      "addr",
			PublicKey: nil,
			Nonce: cpuminer.Uint256{
				A: 1,
				B: 2,
				C: 3,
				D: 4,
			},
		}
		nodeBuf, err := utils.EncodeMsgPack(nodeInfo)
		if err != nil {
			log.Errorf("encode error: %s", err)
		}

		nodeDec := proto.NewNode()
		err = utils.DecodeMsgPack(nodeBuf.Bytes(), nodeDec)

		So(reflect.DeepEqual(nodeDec, nodeInfo), ShouldBeTrue)
	})
}

func TestMarshalBPInfo(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	Convey("marshal unmarshal BPInfo", t, func() {
		sBP, err := yaml.Marshal(BP)
		So(err, ShouldBeNil)
		log.Debugf("BP:\n%s", sBP)
	})
}
