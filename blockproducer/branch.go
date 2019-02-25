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
	"fmt"
	"sort"
	"time"

	"github.com/pkg/errors"

	pi "github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	"github.com/CovenantSQL/CovenantSQL/conf"
	ca "github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/types"
)

type branch struct {
	head     *blockNode
	preview  *metaState
	packed   map[hash.Hash]pi.Transaction
	unpacked map[hash.Hash]pi.Transaction
}

func newBranch(
	baseNode, headNode *blockNode, baseState *metaState, basePool map[hash.Hash]pi.Transaction,
) (
	br *branch, err error,
) {
	var (
		list = headNode.fetchNodeList(baseNode.count + 1)
		inst = &branch{
			head:     headNode,
			preview:  baseState.makeCopy(),
			packed:   make(map[hash.Hash]pi.Transaction),
			unpacked: make(map[hash.Hash]pi.Transaction),
		}
	)
	// Copy pool
	for k, v := range basePool {
		inst.unpacked[k] = v
	}
	// Apply new blocks to view and pool
	for _, bn := range list {
		if bn.txCount > conf.MaxTransactionsPerBlock {
			return nil, ErrTooManyTransactionsInBlock
		}

		var block = bn.load()
		for _, v := range block.Transactions {
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
	inst.preview.commit()
	br = inst
	return
}

// makeArena creates an arena branch from the receiver (origin) branch for tx/block applying test.
// It copies head node and transaction pools, but references the read-only preview index
// of the origin branch.
//
// This branch is typically used to attempt a block producing or applying base on the origin
// branch.
// Since it's sharing read-only index with the origin branch, result changes of the transactions
// in the new block should be written to its dirty index first, and committed to the read-only
// index later when the origin branch is being replaced by this new branch.
func (b *branch) makeArena() *branch {
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
		head: b.head,
		preview: &metaState{
			dirty:    newMetaIndex(),
			readonly: b.preview.readonly,
		},
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

func (b *branch) applyBlock(n *blockNode) (br *branch, err error) {
	var block = n.load()
	if !b.head.hash.IsEqual(block.ParentHash()) {
		err = ErrParentNotMatch
		return
	}
	var cpy = b.makeArena()

	if n.txCount > conf.MaxTransactionsPerBlock {
		return nil, ErrTooManyTransactionsInBlock
	}

	for _, v := range block.Transactions {
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
	cpy.head = n
	br = cpy
	return
}

func (b *branch) sortUnpackedTxs() (txs []pi.Transaction) {
	txs = make([]pi.Transaction, 0, len(b.unpacked))
	for _, v := range b.unpacked {
		txs = append(txs, v)
	}
	sort.Slice(txs, func(i, j int) bool {
		if cmp := bytes.Compare(
			hash.Hash(txs[i].GetAccountAddress()).AsBytes(),
			hash.Hash(txs[j].GetAccountAddress()).AsBytes(),
		); cmp != 0 {
			return cmp < 0
		}
		return txs[i].GetAccountNonce() < txs[j].GetAccountNonce()
	})
	return
}

func (b *branch) produceBlock(
	h uint32, ts time.Time, addr proto.AccountAddress, signer *ca.PrivateKey,
) (
	br *branch, bl *types.BPBlock, err error,
) {
	var (
		cpy       = b.makeArena()
		txs       = cpy.sortUnpackedTxs()
		ierr      error
		packCount = conf.MaxTransactionsPerBlock
	)

	if len(txs) < packCount {
		packCount = len(txs)
	}

	out := make([]pi.Transaction, 0, packCount)
	for _, v := range txs {
		var k = v.Hash()
		if ierr = cpy.preview.apply(v); ierr != nil {
			continue
		}
		delete(cpy.unpacked, k)
		cpy.packed[k] = v
		out = append(out, v)
		if len(out) == packCount {
			break
		}
	}

	// Create new block and update head
	var block = &types.BPBlock{
		SignedHeader: types.BPSignedHeader{
			BPHeader: types.BPHeader{
				Version:    0x01000000,
				Producer:   addr,
				ParentHash: cpy.head.hash,
				Timestamp:  ts,
			},
		},
		Transactions: out,
	}
	if ierr = block.PackAndSignBlock(signer); ierr != nil {
		err = errors.Wrap(ierr, "failed to sign block")
		return
	}
	cpy.head = newBlockNode(h, block, cpy.head)
	br = cpy
	bl = block
	return
}

func (b *branch) clearPackedTxs(txs []pi.Transaction) {
	for _, v := range txs {
		delete(b.packed, v.Hash())
	}
}

func (b *branch) clearUnpackedTxs(txs []pi.Transaction) {
	for _, v := range txs {
		delete(b.unpacked, v.Hash())
	}
}

func (b *branch) queryTxState(hash hash.Hash) (state pi.TransactionState, ok bool) {
	if _, ok = b.unpacked[hash]; ok {
		state = pi.TransactionStatePending
		return
	}
	if _, ok = b.packed[hash]; ok {
		state = pi.TransactionStatePacked
		return
	}
	return
}

func (b *branch) sprint(from uint32) (buff string) {
	var nodes = b.head.fetchNodeList(from)
	for i, v := range nodes {
		if i == 0 {
			var p = v.parent
			buff += fmt.Sprintf("* #%d:%d %s {%d}",
				p.height, p.count, p.hash.Short(4), p.txCount)
		}
		if d := v.height - v.parent.height; d > 1 {
			buff += fmt.Sprintf(" <-- (skip %d blocks)", d-1)
		}
		buff += fmt.Sprintf(" <-- #%d:%d %s {%d}",
			v.height, v.count, v.hash.Short(4), v.txCount)
	}
	return
}
