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
	"encoding/binary"
	"reflect"
	"testing"
	"time"

	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
)

func TestNewBlockNodeAndIndexKey(t *testing.T) {
	chainInitTime := time.Now().UTC()
	period := time.Second
	block, err := generateRandomBlock(hash.Hash{}, true)
	if err != nil {
		t.Fatalf("Unexcepted error: %v", err)
	}
	parent := newBlockNode(chainInitTime, period, block, nil)
	if parent == nil {
		t.Fatal("unexpected result: nil")
	} else if parent.parent != nil {
		t.Fatalf("unexpected parent: %v", parent.parent)
	} else if parent.height != 0 {
		t.Fatalf("unexpected height: %d", parent.height)
	}

	time.Sleep(time.Second)

	block2, err := generateRandomBlock(block.SignedHeader.BlockHash, false)
	if err != nil {
		t.Fatalf("Unexcepted error: %v", err)
	}
	child := newBlockNode(chainInitTime, period, block2, parent)
	if child == nil {
		t.Fatal("unexpected result: nil")
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
	chainInitTime := time.Now()
	period := time.Second
	block, err := generateRandomBlock(hash.Hash{}, true)
	if err != nil {
		t.Fatalf("Unexcepted error: %v", err)
	}
	parent := newBlockNode(chainInitTime, period, block, nil)
	if parent == nil {
		t.Fatal("unexpected result: nil")
	} else if parent.parent != nil {
		t.Fatalf("unexpected parent: %v", parent.parent)
	} else if parent.height != 0 {
		t.Fatalf("unexpected height: %d", parent.height)
	}

	time.Sleep(time.Second)

	block2, err := generateRandomBlock(block.SignedHeader.BlockHash, false)
	if err != nil {
		t.Fatalf("Unexcepted error: %v", err)
	}

	child := newBlockNode(chainInitTime, period, block2, parent)
	if child == nil {
		t.Fatal("unexpected result: nil")
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
	chainInitTime := time.Now()
	period := time.Second

	bi := newBlockIndex()

	if bi == nil {
		t.Fatal("unexpected result: nil")
	}

	block0, err := generateRandomBlock(hash.Hash{}, true)
	if err != nil {
		t.Fatalf("Unexcepted error: %v", err)
	}
	bn0 := newBlockNode(chainInitTime, period, block0, nil)

	time.Sleep(time.Second)

	block1, err := generateRandomBlock(hash.Hash{}, true)
	if err != nil {
		t.Fatalf("Unexcepted error: %v", err)
	}
	bn1 := newBlockNode(chainInitTime, period, block1, bn0)

	time.Sleep(time.Second)

	block2, err := generateRandomBlock(hash.Hash{}, true)
	if err != nil {
		t.Fatalf("Unexcepted error: %v", err)
	}
	bn2 := newBlockNode(chainInitTime, period, block2, bn1)

	bi.addBlock(bn0)
	bi.addBlock(bn1)

	if bi.hasBlock(bn2.hash) {
		t.Fatalf("unexpected block index: %v", bn2)
	}
	if !bi.hasBlock(bn1.hash) {
		t.Fatalf("lack of block index: %v", bn1)
	}

	bn3 := bi.lookupNode(&bn0.hash)
	if !reflect.DeepEqual(bn0, bn3) {
		t.Fatalf("two values should be equal: \n\tv0=%+v\n\tv1=%+v", bn0, bn3)
	}
}
