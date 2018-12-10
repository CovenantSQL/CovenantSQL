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
	"time"

	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

type blockNode struct {
	hash   hash.Hash
	parent *blockNode
	height uint32
	count  uint32
	block  *types.BPBlock
}

func newBlockNodeEx(b *types.BPBlock, p *blockNode) *blockNode {
	return &blockNode{
		hash:   b.SignedHeader.BlockHash,
		parent: p,
		block:  b,
	}
}

func newBlockNode(chainInitTime time.Time, period time.Duration, block *types.BPBlock, parent *blockNode) *blockNode {
	var count uint32

	h := uint32(block.Timestamp().Sub(chainInitTime) / period)

	log.Debugf("chain init time %s, block generation time %s, block height %d", chainInitTime.String(), block.Timestamp().String(), h)

	if parent != nil {
		count = parent.count + 1
	} else {
		count = 0
	}
	bn := &blockNode{
		hash:   *block.BlockHash(),
		parent: parent,
		height: h,
		count:  count,
		block:  block,
	}

	return bn
}

func (bn *blockNode) indexKey() (key []byte) {
	key = make([]byte, hash.HashSize+4)
	binary.BigEndian.PutUint32(key[0:4], bn.height)
	copy(key[4:hash.HashSize], bn.hash[:])
	return
}

func (bn *blockNode) initBlockNode(block *types.BPBlock, parent *blockNode) {
	bn.hash = block.SignedHeader.BlockHash
	bn.parent = nil
	bn.height = 0

	if parent != nil {
		bn.parent = parent
		bn.height = parent.height + 1
	}
}

// blockNodeListFrom return the block node list (forkPoint, bn].
func (bn *blockNode) blockNodeListFrom(forkPoint uint32) (bl []*blockNode) {
	if bn.count <= forkPoint {
		return
	}
	bl = make([]*blockNode, bn.count-forkPoint)
	var iter = bn
	for i := len(bl) - 1; i >= 0; i-- {
		bl[i] = iter
		iter = iter.parent
	}
	return
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

func (bn *blockNode) ancestorByCount(c uint32) *blockNode {
	if c > bn.count {
		return nil
	}

	ancestor := bn
	for ancestor != nil && ancestor.count != c {
		ancestor = ancestor.parent
	}
	return ancestor
}

func (bn *blockNode) lastIrreversible(comfirm uint32) (irr *blockNode) {
	var count uint32
	if bn.count > comfirm {
		count = bn.count - comfirm
	}
	for irr = bn; irr.count > count; irr = irr.parent {
	}
	return
}

func (bn *blockNode) hasAncestor(anc *blockNode) bool {
	return bn.ancestorByCount(anc.count).hash == anc.hash
}

func (bn *blockNode) findNodeAfterCount(hash hash.Hash, min uint32) (match *blockNode, ok bool) {
	for match = bn; match.count >= min; match = match.parent {
		if match.hash.IsEqual(&hash) {
			ok = true
			return
		}
	}
	match = nil
	return
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

func (bi *blockIndex) lookupNode(h *hash.Hash) *blockNode {
	bi.mu.RLock()
	defer bi.mu.RUnlock()

	return bi.index[*h]
}
