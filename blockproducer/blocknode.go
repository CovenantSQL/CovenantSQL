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
	"time"

	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

type blockNode struct {
	parent *blockNode
	// Extra fields
	count  uint32
	height uint32
	// Cached fields for quick reference
	hash  hash.Hash
	block *types.BPBlock
}

func newBlockNodeEx(h uint32, b *types.BPBlock, p *blockNode) *blockNode {
	return &blockNode{
		parent: p,

		count: func() uint32 {
			if p != nil {
				return p.count + 1
			}
			return 0
		}(),
		height: h,

		hash:  b.SignedHeader.BlockHash,
		block: b,
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
	n := &blockNode{
		hash:   *block.BlockHash(),
		parent: parent,
		height: h,
		count:  count,
		block:  block,
	}

	return n
}

func (n *blockNode) indexKey() (key []byte) {
	key = make([]byte, hash.HashSize+4)
	binary.BigEndian.PutUint32(key[0:4], n.height)
	copy(key[4:hash.HashSize], n.hash[:])
	return
}

// fetchNodeList returns the block node list within range (from, n.count] from node head n.
func (n *blockNode) fetchNodeList(from uint32) (bl []*blockNode) {
	if n.count <= from {
		return
	}
	bl = make([]*blockNode, n.count-from)
	var iter = n
	for i := len(bl) - 1; i >= 0; i-- {
		bl[i] = iter
		iter = iter.parent
	}
	return
}

func (n *blockNode) ancestor(h uint32) *blockNode {
	if h > n.height {
		return nil
	}

	ancestor := n
	for ancestor != nil && ancestor.height != h {
		ancestor = ancestor.parent
	}
	return ancestor
}

func (n *blockNode) ancestorByCount(c uint32) *blockNode {
	if c > n.count {
		return nil
	}

	ancestor := n
	for ancestor != nil && ancestor.count != c {
		ancestor = ancestor.parent
	}
	return ancestor
}

// lastIrreversible returns the last irreversible block node with the given confirmations
// from head n. Especially, the block at count 0, also known as the genesis block,
// is irreversible.
func (n *blockNode) lastIrreversible(confirm uint32) (irr *blockNode) {
	var count uint32
	if n.count > confirm {
		count = n.count - confirm
	}
	for irr = n; irr.count > count; irr = irr.parent {
	}
	return
}

func (n *blockNode) hasAncestor(anc *blockNode) bool {
	var match = n.ancestorByCount(anc.count)
	return match != nil && match.hash == anc.hash
}

func (n *blockNode) canForkFrom(
	parent hash.Hash, lastIrreCount uint32) (match *blockNode, ok bool,
) {
	for match = n; match.count >= lastIrreCount; match = match.parent {
		if match.hash.IsEqual(&parent) {
			ok = true
			return
		}
	}
	match = nil
	return
}
