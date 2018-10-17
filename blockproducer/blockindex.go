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
	"sync"

	"github.com/CovenantSQL/CovenantSQL/blockproducer/types"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
)

type blockNode struct {
	hash   hash.Hash
	parent *blockNode
	height uint32
	count  uint32
}

func newBlockNode(h uint32, block *types.Block, parent *blockNode) *blockNode {
	var count uint32

	if parent != nil {
		count = parent.height + 1
	} else {
		count = 0
	}
	bn := &blockNode{
		hash:   block.SignedHeader.BlockHash,
		parent: parent,
		height: h,
		count:  count,
	}

	return bn
}

func (bn *blockNode) indexKey() (key []byte) {
	key = make([]byte, hash.HashSize+4)
	binary.BigEndian.PutUint32(key[0:4], bn.height)
	copy(key[4:hash.HashSize], bn.hash[:])
	return
}

func (bn *blockNode) initBlockNode(block *types.Block, parent *blockNode) {
	bn.hash = block.SignedHeader.BlockHash
	bn.parent = nil
	bn.height = 0

	if parent != nil {
		bn.parent = parent
		bn.height = parent.height + 1
	}
}

func (bn *blockNode) ancestor(h uint32) *blockNode {
	if h > bn.height {
		return nil
	}

	ancestor := bn
	for ancestor != nil && ancestor.height != h {
		ancestor = ancestor.parent
	}
	return ancestor
}

type blockIndex struct {
	mu    sync.RWMutex
	index map[hash.Hash]*blockNode
}

func newBlockIndex() *blockIndex {
	bi := &blockIndex{
		index: make(map[hash.Hash]*blockNode),
	}

	return bi
}

func (bi *blockIndex) addBlock(b *blockNode) {
	bi.mu.Lock()
	defer bi.mu.Unlock()

	bi.index[b.hash] = b
}

func (bi *blockIndex) hasBlock(h hash.Hash) bool {
	bi.mu.RLock()
	defer bi.mu.RUnlock()

	_, has := bi.index[h]
	return has
}

func (bi *blockIndex) lookupBlock(h hash.Hash) *blockNode {
	bi.mu.RLock()
	defer bi.mu.RUnlock()

	return bi.index[h]
}
