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

package sqlchain

import (
	"testing"

	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
	ct "gitlab.com/thunderdb/ThunderDB/sqlchain/types"
)

var (
	testBlocks []*ct.Block
)

func generateTestBlocks() (err error) {
	testBlocks = make([]*ct.Block, 0, testHeight)

	for index, prev := int32(0), genesisHash; index < testHeight; index++ {
		b, err := createRandomBlock(prev, false)

		if err != nil {
			return err
		}

		prev = b.SignedHeader.BlockHash
		testBlocks = append(testBlocks, b)
	}

	return
}

func init() {
	err := generateTestBlocks()

	if err != nil {
		panic(err)
	}
}

func TestNewBlockNode(t *testing.T) {
	parent := newBlockNode(testBlocks[0], nil)

	if parent == nil {
		t.Fatalf("unexpected result: nil")
	} else if parent.parent != nil {
		t.Fatalf("unexpected parent: %v", parent.parent)
	} else if parent.height != 0 {
		t.Fatalf("unexpected height: %d", parent.height)
	}

	child := newBlockNode(testBlocks[1], parent)

	if child == nil {
		t.Fatalf("unexpected result: nil")
	} else if child.parent != parent {
		t.Fatalf("unexpected parent: %v", child.parent)
	} else if child.height != parent.height+1 {
		t.Fatalf("unexpected height: %d", child.height)
	}
}

func TestInitBlockNode(t *testing.T) {
	parent := &blockNode{
		parent: nil,
		hash:   hash.Hash{},
		height: -1,
	}

	child := &blockNode{
		parent: nil,
		hash:   hash.Hash{},
		height: -1,
	}

	parent.initBlockNode(testBlocks[0], nil)

	if parent == nil {
		t.Fatalf("unexpected result: nil")
	} else if parent.parent != nil {
		t.Fatalf("unexpected parent: %v", parent.parent)
	} else if parent.height != 0 {
		t.Fatalf("unexpected height: %d", parent.height)
	}

	child.initBlockNode(testBlocks[1], parent)

	if child == nil {
		t.Fatalf("unexpected result: nil")
	} else if child.parent != parent {
		t.Fatalf("unexpected parent: %v", child.parent)
	} else if child.height != parent.height+1 {
		t.Fatalf("unexpected height: %d", child.height)
	}
}

func TestAncestor(t *testing.T) {
	cfg := &Config{}
	index := newBlockIndex(cfg)
	parent := (*blockNode)(nil)

	for _, b := range testBlocks {
		bn := newBlockNode(b, parent)
		index.addBlock(bn)
		parent = bn
	}

	for i, b := range testBlocks {
		bn := index.lookupNode(&b.SignedHeader.BlockHash)

		if bn == nil {
			t.Fatalf("unexpected loopup result: %v", bn)
		}

		for j := int32(i - 1); j < int32(i+1); j++ {
			a := bn.ancestor(j)

			if j < 0 || j > bn.height {
				if a != nil {
					t.Fatalf("unexpected ancestor: %v", a)
				}
			} else {
				if a.height != j {
					t.Fatalf("unexpected ancestor height: got %d while expecting %d", a.height, j)
				}
			}
		}
	}
}

func TestIndex(t *testing.T) {
	cfg := &Config{}
	index := newBlockIndex(cfg)
	parent := (*blockNode)(nil)

	for _, b := range testBlocks {
		bn := newBlockNode(b, parent)
		index.addBlock(bn)
		parent = bn
	}

	for _, b := range testBlocks {
		if !index.hasBlock(&b.SignedHeader.BlockHash) {
			t.Fatalf("unexpected loopup result: %v", false)
		}
	}

	for _, b := range testBlocks {
		bn := index.lookupNode(&b.SignedHeader.BlockHash)

		if bn == nil {
			t.Fatalf("unexpected loopup result: %v", bn)
		}
	}
}
