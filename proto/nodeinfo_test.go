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

package proto

import (
	"testing"

	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/thunderdb/ThunderDB/crypto/hash"
)

func TestNode_InitNodeCryptoInfo(t *testing.T) {
	NewNodeIDDifficultyTimeout = 1000 * time.Millisecond
	Convey("InitNodeCryptoInfo", t, func() {
		node := NewNode()
		node.InitNodeCryptoInfo()
		hashTmp := hash.THashH(append(node.PublicKey.SerializeCompressed(),
			node.Nonce.Bytes()...))
		t.Logf("ComputeBlockNonce, Difficulty: %d, nonce %v, hash %s",
			node.ID.Difficulty(), node.Nonce, hashTmp.String())

		So(node.ID.Difficulty(), ShouldBeGreaterThan, 1)
		So(hashTmp.String(), ShouldEqual, node.ID)
	})
	Convey("NodeID Difficulty", t, func() {
		var node NodeID
		So(node.Difficulty(), ShouldEqual, -1)
		node = NodeID("")
		So(node.Difficulty(), ShouldEqual, -1)
		node = NodeID("1")
		So(node.Difficulty(), ShouldEqual, -1)
		node = NodeID("00000000011a34cb8142780f692a4097d883aa2ac8a534a070a134f11bcca573")
		So(node.Difficulty(), ShouldEqual, 39)
		node = NodeID("#0000000011a34cb8142780f692a4097d883aa2ac8a534a070a134f11bcca573")
		So(node.Difficulty(), ShouldEqual, -1)
		So((*NodeID)(nil).Difficulty(), ShouldEqual, -1)
	})
}
