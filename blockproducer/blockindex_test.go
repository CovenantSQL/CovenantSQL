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

package blockproducer

import (
	"encoding/binary"
	"reflect"
	"testing"

	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
)

func TestNewBlockNodeAndIndexKey(t *testing.T) {
	block, err := generateRandomBlock(hash.Hash{}, true)
	if err != nil {
		t.Fatalf("Unexcepted error: %v", err)
	}
	parent := newBlockNode(0, block, nil)
	if parent == nil {
		t.Fatalf("unexpected result: nil")
	} else if parent.parent != nil {
		t.Fatalf("unexpected parent: %v", parent.parent)
	} else if parent.height != 0 {
		t.Fatalf("unexpected height: %d", parent.height)
	}

	block2, err := generateRandomBlock(block.SignedHeader.BlockHash, false)
	if err != nil {
		t.Fatalf("Unexcepted error: %v", err)
	}
	child := newBlockNode(1, block2, parent)
	if child == nil {
		t.Fatalf("unexpected result: nil")
	} else if child.parent != parent {
		t.Fatalf("unexpected parent: %v", parent.parent)
	} else if child.height != parent.height+1 {
		t.Fatalf("unexpected height: %d", parent.height)
	}

	// index key
	key1Raw := parent.indexKey()
	key1 := binary.BigEndian.Uint32(key1Raw[0:4])
	key2Raw := child.indexKey()
	key2 := binary.BigEndian.Uint32(key2Raw[0:4])
	if key2 <= key1 {
		t.Fatalf("key2's first byte should be larger than key1's first byte: \n\tkey1[0]=%d\n\tkey2[0]=%d",
			key1, key2)
	}
}

func TestAncestor(t *testing.T) {
	block, err := generateRandomBlock(hash.Hash{}, true)
	if err != nil {
		t.Fatalf("Unexcepted error: %v", err)
	}
	parent := newBlockNode(0, block, nil)
	if parent == nil {
		t.Fatalf("unexpected result: nil")
	} else if parent.parent != nil {
		t.Fatalf("unexpected parent: %v", parent.parent)
	} else if parent.height != 0 {
		t.Fatalf("unexpected height: %d", parent.height)
	}

	block2, err := generateRandomBlock(block.SignedHeader.BlockHash, false)
	if err != nil {
		t.Fatalf("Unexcepted error: %v", err)
	}
	child := newBlockNode(1, block2, parent)
	if child == nil {
		t.Fatalf("unexpected result: nil")
	} else if child.parent != parent {
		t.Fatalf("unexpected parent: %v", parent.parent)
	} else if child.height != parent.height+1 {
		t.Fatalf("unexpected height: %d", parent.height)
	}

	bn := child.ancestor(2)
	if bn != nil {
		t.Fatalf("should return nil, but get a block node: %v", bn)
	}
	bn = child.ancestor(1)
	if bn == nil || bn.height != 1 {
		t.Fatal("block node should not be nil and its height should be 1")
	}
	bn = child.ancestor(0)
	if bn == nil || bn.height != 0 {
		t.Fatal("block node should not be nil and its height should be 0")
	}
}

func TestIndexBlock(t *testing.T) {
	bi := newBlockIndex()

	if bi == nil {
		t.Fatalf("unexpected result: nil")
	}

	block0, err := generateRandomBlock(hash.Hash{}, true)
	if err != nil {
		t.Fatalf("Unexcepted error: %v", err)
	}
	bn0 := newBlockNode(0, block0, nil)

	block1, err := generateRandomBlock(hash.Hash{}, true)
	if err != nil {
		t.Fatalf("Unexcepted error: %v", err)
	}
	bn1 := newBlockNode(1, block1, bn0)

	block2, err := generateRandomBlock(hash.Hash{}, true)
	if err != nil {
		t.Fatalf("Unexcepted error: %v", err)
	}
	bn2 := newBlockNode(2, block2, bn1)

	bi.addBlock(bn0)
	bi.addBlock(bn1)

	if bi.hasBlock(bn2.hash) {
		t.Fatalf("unexpected block index: %v", bn2)
	}
	if !bi.hasBlock(bn1.hash) {
		t.Fatalf("lack of block index: %v", bn1)
	}

	bn3 := bi.lookupBlock(bn0.hash)
	if !reflect.DeepEqual(bn0, bn3) {
		t.Fatalf("two values should be equal: \n\tv0=%+v\n\tv1=%+v", bn0, bn3)
	}
}
