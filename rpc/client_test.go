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

package rpc

import (
	"encoding/hex"
	"testing"

	"net"

	"os"

	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/thunderdb/ThunderDB/crypto/asymmetric"
	"gitlab.com/thunderdb/ThunderDB/crypto/kms"
	mine "gitlab.com/thunderdb/ThunderDB/pow/cpuminer"
	"gitlab.com/thunderdb/ThunderDB/proto"
	"gitlab.com/thunderdb/ThunderDB/route"
)

const nodeID = "0000"
const privateKey = "test.private"
const publicKeyStore = "./test.keystore"
const pass = "abc"

func TestDial(t *testing.T) {
	Convey("dial error case", t, func() {
		c, err := dial("tcp", "wrongaddr", nil, nil)
		So(c, ShouldBeNil)
		So(err, ShouldNotBeNil)

		var l net.Listener
		l, _ = net.Listen("tcp", "127.0.0.1:0")
		c, err = dial("tcp", l.Addr().String(), nil, nil)
		So(c, ShouldBeNil)
		So(err, ShouldNotBeNil)

		kms.SetLocalNodeIDNonce([]byte(nodeID), nil)
		c, err = dial("tcp", l.Addr().String(), nil, nil)
		So(c, ShouldBeNil)
		So(err, ShouldNotBeNil)

		kms.SetLocalNodeIDNonce([]byte(nodeID), &mine.Uint256{1, 1, 1, 1})
		c, err = dial("tcp", l.Addr().String(), nil, nil)
		So(c, ShouldNotBeNil)
		So(err, ShouldBeNil)

		go func() {
			l.Accept()
		}()
		c, err = dial("tcp", l.Addr().String(), nil, nil)
		So(c, ShouldNotBeNil)
		So(err, ShouldBeNil)
	})
}

func TestDialToNode(t *testing.T) {
	Convey("DialToNode error case", t, func() {
		defer os.Remove(publicKeyStore)
		defer os.Remove(privateKey)
		c, err := DialToNode(kms.BP.NodeID, nil)
		So(c, ShouldBeNil)
		So(err, ShouldNotBeNil)

		publicKeyBytes, _ := hex.DecodeString(kms.BP.PublicKeyStr)
		kms.BP.PublicKey, _ = asymmetric.ParsePubKey(publicKeyBytes)
		BPNode := &proto.Node{
			ID:        kms.BP.NodeID,
			Addr:      "",
			PublicKey: kms.BP.PublicKey,
			Nonce:     kms.BP.Nonce,
		}

		kms.InitPublicKeyStore(publicKeyStore, BPNode)
		c, err = DialToNode(proto.NodeID(nodeID), nil)
		So(c, ShouldBeNil)
		So(err, ShouldNotBeNil)

		kms.InitLocalKeyPair(privateKey, []byte(pass))
		//route.initResolver()
		c, err = DialToNode(kms.BP.NodeID, nil)
		So(c, ShouldBeNil)
		So(err, ShouldNotBeNil)

		l, _ := net.Listen("tcp", "127.0.0.1:0")

		route.SetNodeAddrCache(&kms.BP.RawNodeID, l.Addr().String())
		c, err = DialToNode(kms.BP.NodeID, nil)
		So(err, ShouldBeNil)
		So(c, ShouldNotBeNil)
	})
}
