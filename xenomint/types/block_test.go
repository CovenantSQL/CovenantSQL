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

package types

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/crypto/verifier"
	"github.com/CovenantSQL/CovenantSQL/types"
)

func TestBlock(t *testing.T) {
	Convey("Given a block and a pair of keys", t, func() {
		var (
			block = &Block{
				SignedBlockHeader: SignedBlockHeader{
					BlockHeader: BlockHeader{},
				},
				ReadQueries: []*types.Ack{
					{
						Header: types.SignedAckHeader{
							DefaultHashSignVerifierImpl: verifier.DefaultHashSignVerifierImpl{
								DataHash: hash.Hash{0x0, 0x0, 0x0, 0x1},
							},
						},
					},
				},
				WriteQueries: []*types.Ack{
					{
						Header: types.SignedAckHeader{
							DefaultHashSignVerifierImpl: verifier.DefaultHashSignVerifierImpl{
								DataHash: hash.Hash{0x0, 0x0, 0x0, 0x2},
							},
						},
					},
				},
			}
			priv, _, err = asymmetric.GenSecp256k1KeyPair()
		)
		So(err, ShouldBeNil)
		So(priv, ShouldNotBeNil)
		Convey("When the block is signed by the key pair", func() {
			err = block.Sign(priv)
			So(err, ShouldBeNil)
			Convey("The block should be verifiable", func() {
				err = block.Verify()
				So(err, ShouldBeNil)
			})
			Convey("The object should have data hash", func() {
				var enc, err = block.BlockHeader.MarshalHash()
				So(err, ShouldBeNil)
				So(enc, ShouldNotBeNil)
				So(block.SignedBlockHeader.Hash(), ShouldEqual, hash.THashH(enc))
			})
			Convey("When the queries is modified", func() {
				block.ReadQueries = append(block.ReadQueries, &types.Ack{
					Header: types.SignedAckHeader{
						DefaultHashSignVerifierImpl: verifier.DefaultHashSignVerifierImpl{
							DataHash: hash.Hash{0x0, 0x0, 0x0, 0x3},
						},
					},
				})
				Convey("The verifier should return merkle root not match error", func() {
					err = block.Verify()
					So(err, ShouldEqual, ErrMerkleRootNotMatch)
				})
			})
		})
	})
}
