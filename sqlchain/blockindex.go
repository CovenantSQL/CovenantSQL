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
	"encoding/binary"
	"sync"
	"sync/atomic"

	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/types"
)

type blockNode struct {
	parent *blockNode
	block  atomic.Value // store *types.Block
	hash   hash.Hash
	height int32 // height is the chain height of the head
	count  int32 // count counts the blocks (except genesis) at this head
}

func newBlockNodeEx(height int32, hash *hash.Hash, block *types.Block, parent *blockNode) *blockNode {
	var count int32
	if parent != nil {
		count = parent.count + 1
	}
	node := &blockNode{
		hash:   *hash,
		parent: parent,
		height: height,
		count:  count,
	}
	node.block.Store(block)
	return node
}

func newBlockNode(height int32, block *types.Block, parent *blockNode) *blockNode {
	return newBlockNodeEx(height, block.BlockHash(), block, parent)
}

func (n *blockNode) load() *types.Block {
	return n.block.Load().(*types.Block)
}

func (n *blockNode) clear() (cleared bool) {
	if n.load() != nil {
		cleared = true
		n.block.Store((*types.Block)(nil))
	}
	return
}

func (n *blockNode) ancestor(height int32) (ancestor *blockNode) {
	if height < 0 || height > n.height {
		return nil
	}

	for ancestor = n; ancestor != nil && ancestor.height > height; ancestor = ancestor.parent {
	}

	// The block at this height may not exist
	if ancestor != nil && ancestor.height < height {
		ancestor = nil
	}

	return
}

func (n *blockNode) ancestorByCount(count int32) (ancestor *blockNode) {
	if count < 0 || count > n.count {
		return nil
	}

	for ancestor = n; ancestor != nil && ancestor.count > count; ancestor = ancestor.parent {
	}

	if ancestor != nil && ancestor.count < count {
		ancestor = nil
	}

	return
}

func (n *blockNode) indexKey() (key []byte) {
	key = make([]byte, hash.HashSize+4)
	binary.BigEndian.PutUint32(key[0:4], uint32(n.height))
	copy(key[4:hash.HashSize+4], n.hash[:])
	return
}

type blockIndex struct {
	mu    sync.RWMutex
	index map[hash.Hash]*blockNode
}

func newBlockIndex() (index *blockIndex) {
	return &blockIndex{
		index: make(map[hash.Hash]*blockNode),
	}
}

func (i *blockIndex) addBlock(newBlock *blockNode) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.index[newBlock.hash] = newBlock
}

func (i *blockIndex) hasBlock(hash *hash.Hash) (hasBlock bool) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	_, hasBlock = i.index[*hash]
	return
}

func (i *blockIndex) lookupNode(hash *hash.Hash) (b *blockNode) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	b = i.index[*hash]
	return
}
