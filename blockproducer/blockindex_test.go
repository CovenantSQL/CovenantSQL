/*
 * Copyright 2018 The ThunderDB Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the “License”);
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an “AS IS” BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package blockproducer

import (
	"reflect"
	"testing"

	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
)

func TestNewBlockNode(t *testing.T) {
	block, err := generateRandomBlock(hash.Hash{}, true)
	if err != nil {
		t.Fatalf("Unexcepted error: %v", err)
	}
	parent := newBlockNode(block, nil)
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
	child := newBlockNode(block2, parent)
	if child == nil {
		t.Fatalf("unexpected result: nil")
	} else if child.parent != parent {
		t.Fatalf("unexpected parent: %v", parent.parent)
	} else if child.height != parent.height+1 {
		t.Fatalf("unexpected height: %d", parent.height)
	}
}

func TestIndexBlock(t *testing.T) {
	cfg := NewConfig()
	bi := newBlockIndex(cfg)

	if bi == nil {
		t.Fatalf("unexpected result: nil")
	} else if bi.cfg == nil {
		t.Fatalf("unexpected config")
	}

	block0, err := generateRandomBlock(hash.Hash{}, true)
	if err != nil {
		t.Fatalf("Unexcepted error: %v", err)
	}
	bn0 := newBlockNode(block0, nil)

	block1, err := generateRandomBlock(hash.Hash{}, true)
	if err != nil {
		t.Fatalf("Unexcepted error: %v", err)
	}
	bn1 := newBlockNode(block1, bn0)

	block2, err := generateRandomBlock(hash.Hash{}, true)
	if err != nil {
		t.Fatalf("Unexcepted error: %v", err)
	}
	bn2 := newBlockNode(block2, bn1)

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
