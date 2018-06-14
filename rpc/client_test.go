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

	ec "github.com/btcsuite/btcd/btcec"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/thunderdb/ThunderDB/crypto/kms"
	mine "github.com/thunderdb/ThunderDB/pow/cpuminer"
	"github.com/thunderdb/ThunderDB/proto"
	"github.com/thunderdb/ThunderDB/route"
)

const nodeID = "0000"
const privateKey = "test.private"
const publicKeyStore = "./test.keystore"
const pass = "abc"

func TestDail(t *testing.T) {
	Convey("Dial error case", t, func() {
		c, err := Dial("tcp", "wrongaddr", nil)
		So(c, ShouldBeNil)
		So(err, ShouldNotBeNil)

		kms.InitLocalKeyStore()
		var l net.Listener
		l, _ = net.Listen("tcp", "127.0.0.1:0")
		c, err = Dial("tcp", l.Addr().String(), nil)
		So(c, ShouldBeNil)
		So(err, ShouldNotBeNil)

		kms.SetLocalNodeIDNonce([]byte(nodeID), nil)
		c, err = Dial("tcp", l.Addr().String(), nil)
		So(c, ShouldBeNil)
		So(err, ShouldNotBeNil)

		kms.SetLocalNodeIDNonce([]byte(nodeID), &mine.Uint256{1, 1, 1, 1})
		c, err = Dial("tcp", l.Addr().String(), nil)
		So(c, ShouldNotBeNil)
		So(err, ShouldBeNil)

		go func() {
			l.Accept()
		}()
		c, err = Dial("tcp", l.Addr().String(), nil)
		So(c, ShouldNotBeNil)
		So(err, ShouldBeNil)
	})
}

func TestDailToNode(t *testing.T) {
	Convey("DialToNode error case", t, func() {
		defer os.Remove(publicKeyStore)
		defer os.Remove(privateKey)
		kms.InitLocalKeyStore()
		c, err := DailToNode(proto.NodeID(kms.BPNodeID))
		So(c, ShouldBeNil)
		So(err, ShouldNotBeNil)

		publicKeyBytes, _ := hex.DecodeString(kms.BPPublicKeyStr)
		kms.BPPublicKey, _ = ec.ParsePubKey(publicKeyBytes, ec.S256())
		BPNode := &proto.Node{
			ID:        proto.NodeID(kms.BPNodeID),
			Addr:      "",
			PublicKey: kms.BPPublicKey,
			Nonce:     kms.BPNonce,
		}

		kms.InitPublicKeyStore(publicKeyStore, BPNode)
		c, err = DailToNode(proto.NodeID(nodeID))
		So(c, ShouldBeNil)
		So(err, ShouldNotBeNil)

		kms.InitLocalKeyPair(privateKey, []byte(pass))
		route.InitResolver()
		c, err = DailToNode(proto.NodeID(kms.BPNodeID))
		So(c, ShouldBeNil)
		So(err, ShouldNotBeNil)

		l, _ := net.Listen("tcp", "127.0.0.1:0")

		route.SetNodeAddr(proto.NodeID(kms.BPNodeID), l.Addr().String())
		c, err = DailToNode(proto.NodeID(kms.BPNodeID))
		So(c, ShouldNotBeNil)
		So(err, ShouldBeNil)
	})
}
