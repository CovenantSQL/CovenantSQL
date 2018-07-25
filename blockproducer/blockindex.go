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
	"encoding/binary"
	"sync"

	"gitlab.com/thunderdb/ThunderDB/blockproducer/types"
	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
)

type blockNode struct {
	hash   hash.Hash
	parent *blockNode
	height uint64
}

func newBlockNode(block *types.Block, parent *blockNode) *blockNode {
	var height uint64

	if parent != nil {
		height = parent.height + 1
	} else {
		height = 0
	}
	bn := &blockNode{
		hash:   block.SignedHeader.BlockHash,
		parent: parent,
		height: height,
	}

	return bn
}

func (bn *blockNode) indexKey() (key []byte) {
	key = make([]byte, hash.HashSize+4)
	binary.BigEndian.PutUint32(key[0:4], uint32(bn.height))
	copy(key[4:hash.HashSize], bn.hash[:])
	return
}

func (bn *blockNode) initBlockNode(b *types.Block, p *blockNode) {
	bn.hash = b.SignedHeader.BlockHash
	bn.parent = nil
	bn.height = 0
	if p != nil {
		bn.parent = p
		bn.height = p.height + 1
	}
}

type blockIndex struct {
	cfg *config

	mu    sync.RWMutex
	index map[hash.Hash]*blockNode
}

func newBlockIndex(config *config) *blockIndex {
	bi := &blockIndex{
		cfg:   config,
		index: make(map[hash.Hash]*blockNode),
	}

	return bi
}

func (bi *blockIndex) addBlock(b *blockNode) {
	bi.mu.RLock()
	defer bi.mu.RUnlock()

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
