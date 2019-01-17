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

package blockproducer

import (
	"testing"

	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/crypto/verifier"
	"github.com/CovenantSQL/CovenantSQL/types"
	. "github.com/smartystreets/goconvey/convey"
)

func TestBlockNode(t *testing.T) {
	Convey("Given a set of block nodes", t, func() {
		var (
			b0 = &types.BPBlock{
				SignedHeader: types.BPSignedHeader{
					DefaultHashSignVerifierImpl: verifier.DefaultHashSignVerifierImpl{
						DataHash: hash.Hash{0x1},
					},
				},
			}
			b1 = &types.BPBlock{
				SignedHeader: types.BPSignedHeader{
					BPHeader: types.BPHeader{
						ParentHash: b0.SignedHeader.DataHash,
					},
					DefaultHashSignVerifierImpl: verifier.DefaultHashSignVerifierImpl{
						DataHash: hash.Hash{0x2},
					},
				},
			}
			b2 = &types.BPBlock{
				SignedHeader: types.BPSignedHeader{
					BPHeader: types.BPHeader{
						ParentHash: b1.SignedHeader.DataHash,
					},
					DefaultHashSignVerifierImpl: verifier.DefaultHashSignVerifierImpl{
						DataHash: hash.Hash{0x3},
					},
				},
			}
			b3 = &types.BPBlock{
				SignedHeader: types.BPSignedHeader{
					BPHeader: types.BPHeader{
						ParentHash: b2.SignedHeader.DataHash,
					},
					DefaultHashSignVerifierImpl: verifier.DefaultHashSignVerifierImpl{
						DataHash: hash.Hash{0x4},
					},
				},
			}
			b4 = &types.BPBlock{
				SignedHeader: types.BPSignedHeader{
					BPHeader: types.BPHeader{
						ParentHash: b3.SignedHeader.DataHash,
					},
					DefaultHashSignVerifierImpl: verifier.DefaultHashSignVerifierImpl{
						DataHash: hash.Hash{0x5},
					},
				},
			}
			n0 = newBlockNode(0, b0, nil)
			n1 = newBlockNode(1, b1, n0)
			n2 = newBlockNode(2, b2, n1)
			n3 = newBlockNode(3, b3, n2)
			n4 = newBlockNode(5, b4, n3)

			b3p = &types.BPBlock{
				SignedHeader: types.BPSignedHeader{
					BPHeader: types.BPHeader{
						ParentHash: b2.SignedHeader.DataHash,
					},
					DefaultHashSignVerifierImpl: verifier.DefaultHashSignVerifierImpl{
						DataHash: hash.Hash{0x6},
					},
				},
			}
			b4p = &types.BPBlock{
				SignedHeader: types.BPSignedHeader{
					BPHeader: types.BPHeader{
						ParentHash: b3p.SignedHeader.DataHash,
					},
					DefaultHashSignVerifierImpl: verifier.DefaultHashSignVerifierImpl{
						DataHash: hash.Hash{0x7},
					},
				},
			}
			n3p = newBlockNode(3, b3p, n2)
			n4p = newBlockNode(4, b4p, n3p)
		)

		/*
			n0 --- n1 --- n2 --- n3 --- x ---- n4
			               \          (skip)
			                \
			                 +---n3p -- n4p
		*/

		So(n0.count, ShouldEqual, 0)
		So(n1.count, ShouldEqual, n0.count+1)

		So(n0.fetchNodeList(0), ShouldBeEmpty)
		So(n0.fetchNodeList(1), ShouldBeEmpty)
		So(n0.fetchNodeList(2), ShouldBeEmpty)
		So(n3.fetchNodeList(0), ShouldResemble, []*blockNode{n1, n2, n3})
		So(n4p.fetchNodeList(2), ShouldResemble, []*blockNode{n3p, n4p})

		So(n0.ancestor(1), ShouldBeNil)
		So(n3.ancestor(3), ShouldEqual, n3)
		So(n3.ancestor(0), ShouldEqual, n0)
		So(n4.ancestor(4), ShouldBeNil)

		So(n0.ancestorByCount(1), ShouldBeNil)
		So(n3.ancestorByCount(3), ShouldEqual, n3)
		So(n3.ancestorByCount(0), ShouldEqual, n0)

		So(n3.lastIrreversible(0), ShouldEqual, n3)
		So(n3.lastIrreversible(1), ShouldEqual, n2)
		So(n3.lastIrreversible(3), ShouldEqual, n0)
		So(n3.lastIrreversible(9), ShouldEqual, n0)

		So(n0.hasAncestor(n4), ShouldBeFalse)
		So(n3.hasAncestor(n0), ShouldBeTrue)
		So(n3p.hasAncestor(n0), ShouldBeTrue)
		So(n4.hasAncestor(n3p), ShouldBeFalse)
		So(n4p.hasAncestor(n3), ShouldBeFalse)

		var (
			f  *blockNode
			ok bool
		)

		f, ok = n4.hasAncestorWithMinCount(n2.hash, n0.count)
		So(ok, ShouldBeTrue)
		So(f, ShouldEqual, n2)
		f, ok = n4.hasAncestorWithMinCount(n4.hash, n0.count)
		So(ok, ShouldBeTrue)
		So(f, ShouldEqual, n4)
		f, ok = n4.hasAncestorWithMinCount(n0.hash, n2.count)
		So(ok, ShouldBeFalse)
		f, ok = n4.hasAncestorWithMinCount(n3p.hash, n2.count)
		So(ok, ShouldBeFalse)
		f, ok = n4p.hasAncestorWithMinCount(n3.hash, n2.count)
		So(ok, ShouldBeFalse)
	})
}
