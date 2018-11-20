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
	pi "github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

type accountTxEntries struct {
	account      proto.AccountAddress
	baseNonce    pi.AccountNonce
	transactions []pi.Transaction
}

func newAccountTxEntries(
	addr proto.AccountAddress, baseNonce pi.AccountNonce) (_ *accountTxEntries,
) {
	return &accountTxEntries{
		account:      addr,
		baseNonce:    baseNonce,
		transactions: nil,
	}
}

func (e *accountTxEntries) nextNonce() pi.AccountNonce {
	return e.baseNonce + pi.AccountNonce(len(e.transactions))
}

func (e *accountTxEntries) addTx(tx pi.Transaction) {
	e.transactions = append(e.transactions, tx)
}

func (e *accountTxEntries) halfDeepCopy() (cpy *accountTxEntries) {
	return &accountTxEntries{
		account:      e.account,
		baseNonce:    e.baseNonce,
		transactions: e.transactions[:],
	}
}

type txPool struct {
	entries map[proto.AccountAddress]*accountTxEntries
}

func newTxPool() *txPool {
	return &txPool{
		entries: make(map[proto.AccountAddress]*accountTxEntries),
	}
}

func (p *txPool) addTx(tx pi.Transaction, baseNonce pi.AccountNonce) {
	addr := tx.GetAccountAddress()
	e, ok := p.entries[addr]
	if !ok {
		e = newAccountTxEntries(addr, baseNonce)
		p.entries[addr] = e
	}
	e.addTx(tx)
}

func (p *txPool) getTxEntries(addr proto.AccountAddress) (e *accountTxEntries, ok bool) {
	e, ok = p.entries[addr]
	return
}

func (p *txPool) hasTx(tx pi.Transaction) (ok bool) {
	var te *accountTxEntries
	if te, ok = p.entries[tx.GetAccountAddress()]; !ok {
		return
	}
	// Out of range
	var (
		nonce = tx.GetAccountNonce()
		index = int(nonce - te.baseNonce)
	)
	if ok = (nonce >= te.baseNonce && index < len(te.transactions)); !ok {
		log.Debug("transaction nonce or index already exists")
		return
	}
	// Check transaction hash
	if ok = (tx.Hash() == te.transactions[index].Hash()); !ok {
		log.Debug("transaction hash already exists")
		return
	}

	return
}

func (p *txPool) cmpAndMoveNextTx(tx pi.Transaction) (ok bool) {
	var te *accountTxEntries
	if te, ok = p.entries[tx.GetAccountAddress()]; !ok {
		return
	}
	// Out of range
	if ok = (tx.GetAccountNonce() == te.baseNonce && len(te.transactions) > 0); !ok {
		return
	}
	// Check transaction hash
	if ok = (tx.Hash() == te.transactions[0].Hash()); !ok {
		return
	}
	// Move forward
	te.transactions = te.transactions[1:]
	te.baseNonce++
	return
}

func (p *txPool) halfDeepCopy() (cpy *txPool) {
	cpy = newTxPool()
	for k, v := range p.entries {
		cpy.entries[k] = v.halfDeepCopy()
	}
	return
}
