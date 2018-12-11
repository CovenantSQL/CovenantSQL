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

package sqlchain

import (
	"testing"

	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/types"
)

var (
	testBlocks      []*types.Block
	testBlockNumber = 50
)

func generateTestBlocks() (err error) {
	testBlocks = make([]*types.Block, 0, testBlockNumber)

	for i, prev := 0, genesisHash; i < testBlockNumber; i++ {
		b, err := createRandomBlock(prev, false)

		if err != nil {
			return err
		}

		prev = *b.BlockHash()
		testBlocks = append(testBlocks, b)
	}

	return
}

func init() {
	if err := generateTestBlocks(); err != nil {
		panic(err)
	}
}

func TestNewBlockNode(t *testing.T) {
	parent := newBlockNode(0, testBlocks[0], nil)

	if parent == nil {
		t.Fatal("unexpected result: nil")
	} else if parent.parent != nil {
		t.Fatalf("unexpected parent: %v", parent.parent)
	} else if parent.count != 0 {
		t.Fatalf("unexpected height: %d", parent.count)
	}

	child := newBlockNode(1, testBlocks[1], parent)

	if child == nil {
		t.Fatal("unexpected result: nil")
	} else if child.parent != parent {
		t.Fatalf("unexpected parent: %v", child.parent)
	} else if child.count != parent.count+1 {
		t.Fatalf("unexpected height: %d", child.count)
	}
}

func TestInitBlockNode(t *testing.T) {
	parent := &blockNode{
		parent: nil,
		hash:   hash.Hash{},
		count:  -1,
	}

	child := &blockNode{
		parent: nil,
		hash:   hash.Hash{},
		count:  -1,
	}

	parent.initBlockNode(0, testBlocks[0], nil)

	if parent == nil {
		t.Fatal("unexpected result: nil")
	} else if parent.parent != nil {
		t.Fatalf("unexpected parent: %v", parent.parent)
	} else if parent.count != 0 {
		t.Fatalf("unexpected height: %d", parent.count)
	}

	child.initBlockNode(1, testBlocks[1], parent)

	if child == nil {
		t.Fatal("unexpected result: nil")
	} else if child.parent != parent {
		t.Fatalf("unexpected parent: %v", child.parent)
	} else if child.count != parent.count+1 {
		t.Fatalf("unexpected height: %d", child.count)
	}
}

func TestAncestor(t *testing.T) {
	index := newBlockIndex()
	parent := (*blockNode)(nil)

	for h, b := range testBlocks {
		bn := newBlockNode(int32(h), b, parent)
		index.addBlock(bn)
		parent = bn
	}

	for i, b := range testBlocks {
		bn := index.lookupNode(b.BlockHash())

		if bn == nil {
			t.Fatalf("unexpected loopup result: %v", bn)
		}

		for j := int32(i - 1); j < int32(i+1); j++ {
			a := bn.ancestor(j)

			if j < 0 || j > bn.count {
				if a != nil {
					t.Fatalf("unexpected ancestor: %v", a)
				}
			} else {
				if a.count != j {
					t.Fatalf("unexpected ancestor height: got %d while expecting %d", a.count, j)
				}
			}
		}
	}
}

func TestIndex(t *testing.T) {
	index := newBlockIndex()
	parent := (*blockNode)(nil)

	for h, b := range testBlocks {
		bn := newBlockNode(int32(h), b, parent)
		index.addBlock(bn)
		parent = bn
	}

	for _, b := range testBlocks {
		if !index.hasBlock(b.BlockHash()) {
			t.Fatalf("unexpected loopup result: %v", false)
		}
	}

	for _, b := range testBlocks {
		bn := index.lookupNode(b.BlockHash())

		if bn == nil {
			t.Fatalf("unexpected loopup result: %v", bn)
		}
	}
}
