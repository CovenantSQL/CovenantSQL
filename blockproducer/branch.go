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
	"bytes"
	"sort"

	pi "github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/types"
)

type view struct {
}

func (v *view) apply(tx pi.Transaction) (err error) {
	return
}

func (v *view) makeCopy() *view {
	return &view{}
}

type branch struct {
	head     *blockNode
	preview  *view
	packed   map[hash.Hash]pi.Transaction
	unpacked map[hash.Hash]pi.Transaction
}

func fork(
	from, to *blockNode, v *view, txPool map[hash.Hash]pi.Transaction) (br *branch, err error,
) {
	var (
		diff = to.blockNodeListFrom(from.count)
		inst = &branch{
			head:     to,
			preview:  v.makeCopy(),
			packed:   make(map[hash.Hash]pi.Transaction),
			unpacked: make(map[hash.Hash]pi.Transaction),
		}
	)
	// Copy pool
	for k, v := range txPool {
		inst.unpacked[k] = v
	}
	// Apply new blocks to view and pool
	for _, bn := range diff {
		for _, v := range bn.block.Transactions {
			var k = v.Hash()
			// Check in tx pool
			if _, ok := inst.unpacked[k]; ok {
				delete(inst.unpacked, k)
			} else if err = v.Verify(); err != nil {
				return
			}
			if _, ok := inst.packed[k]; ok {
				err = ErrExistedTx
				return
			}
			inst.packed[k] = v
			// Apply to preview
			if err = inst.preview.apply(v); err != nil {
				return
			}
		}
	}
	return
}

func (b *branch) makeCopy() *branch {
	var (
		p = make(map[hash.Hash]pi.Transaction)
		u = make(map[hash.Hash]pi.Transaction)
	)
	for k, v := range b.packed {
		p[k] = v
	}
	for k, v := range b.unpacked {
		u[k] = v
	}
	return &branch{
		head:     b.head,
		preview:  b.preview.makeCopy(),
		packed:   p,
		unpacked: u,
	}
}

func (b *branch) addTx(tx pi.Transaction) {
	var k = tx.Hash()
	if _, ok := b.packed[k]; !ok {
		if _, ok := b.unpacked[k]; !ok {
			b.unpacked[k] = tx
		}
	}
}

func (b *branch) applyBlock(bl *types.BPBlock) (br *branch, err error) {
	if !b.head.hash.IsEqual(bl.ParentHash()) {
		err = ErrParentNotMatch
		return
	}
	var cpy = b.makeCopy()
	for _, v := range bl.Transactions {
		var k = v.Hash()
		// Check in tx pool
		if _, ok := cpy.unpacked[k]; ok {
			delete(cpy.unpacked, k)
		} else if err = v.Verify(); err != nil {
			return
		}
		if _, ok := cpy.packed[k]; ok {
			err = ErrExistedTx
			return
		}
		cpy.packed[k] = v
		// Apply to preview
		if err = cpy.preview.apply(v); err != nil {
			return
		}
	}
	cpy.head = newBlockNodeEx(bl, cpy.head)
	br = cpy
	return
}

func (b *branch) sortUnpackedTxs() (txs []pi.Transaction) {
	txs = make([]pi.Transaction, 0, len(b.unpacked))
	for _, v := range b.unpacked {
		txs = append(txs, v)
	}
	sort.Slice(txs, func(i, j int) bool {
		if cmp := bytes.Compare(txs[i].Hash().AsBytes(), txs[j].Hash().AsBytes()); cmp != 0 {
			return cmp < 0
		}
		return txs[i].GetAccountNonce() < txs[j].GetAccountNonce()
	})
	return
}

func newBlock(out []pi.Transaction) (bl *types.BPBlock) {
	return
}

func (b *branch) produceBlock() (br *branch, bl *types.BPBlock, err error) {
	var (
		cpy = b.makeCopy()
		txs = cpy.sortUnpackedTxs()
		out = make([]pi.Transaction, 0, len(txs))
	)
	for _, v := range txs {
		var k = v.Hash()
		if err = cpy.preview.apply(v); err != nil {
			continue
		}
		delete(cpy.unpacked, k)
		cpy.packed[k] = v
		out = append(out, v)
	}
	// Create new block and update head
	bl = newBlock(out)
	cpy.head = newBlockNodeEx(bl, cpy.head)
	br = cpy
	return
}

func (b *branch) clearPackedTxs(txs []pi.Transaction) {
	for _, v := range txs {
		delete(b.packed, v.Hash())
	}
}
