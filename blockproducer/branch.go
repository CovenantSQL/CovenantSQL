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
 * See the License for the specific language governing permissions and * limitations under the License.
 */

package blockproducer

import (
	"bytes"
	"fmt"
	"sort"
	"time"

	pi "github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	ca "github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/pkg/errors"
)

type branch struct {
	head     *blockNode
	preview  *metaState
	packed   map[hash.Hash]pi.Transaction
	unpacked map[hash.Hash]pi.Transaction
}

func fork(
	from, to *blockNode, initState *metaState, initPool map[hash.Hash]pi.Transaction,
) (
	br *branch, err error,
) {
	var (
		list = to.fetchNodeList(from.count)
		inst = &branch{
			head:     to,
			preview:  initState.makeCopy(),
			packed:   make(map[hash.Hash]pi.Transaction),
			unpacked: make(map[hash.Hash]pi.Transaction),
		}
	)
	// Copy pool
	for k, v := range initPool {
		inst.unpacked[k] = v
	}
	// Apply new blocks to view and pool
	for _, bn := range list {
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
	inst.preview.commit()
	br = inst
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

func (b *branch) applyBlock(n *blockNode) (br *branch, err error) {
	if !b.head.hash.IsEqual(n.block.ParentHash()) {
		err = ErrParentNotMatch
		return
	}
	var cpy = b.makeCopy()
	for _, v := range n.block.Transactions {
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
	cpy.preview.commit()
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
		cpy  = b.makeCopy()
		txs  = cpy.sortUnpackedTxs()
		out  = make([]pi.Transaction, 0, len(txs))
		ierr error
	)
	for _, v := range txs {
		var k = v.Hash()
		if ierr = cpy.preview.apply(v); ierr != nil {
			continue
		}
		delete(cpy.unpacked, k)
		cpy.packed[k] = v
		out = append(out, v)
	}
	cpy.preview.commit()
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

func (b *branch) sprint(from uint32) (buff string) {
	var nodes = b.head.fetchNodeList(from)
	for i, v := range nodes {
		if i == 0 {
			var p = v.parent
			buff += fmt.Sprintf("* #%d:%d %s {%d}",
				p.height, p.count, p.hash.Short(4), len(p.block.Transactions))
		}
		if d := v.height - v.parent.height; d > 1 {
			buff += fmt.Sprintf(" <-- (skip %d blocks)", d-1)
		}
		buff += fmt.Sprintf(" <-- #%d:%d %s {%d}",
			v.height, v.count, v.hash.Short(4), len(v.block.Transactions))
	}
	return
}
