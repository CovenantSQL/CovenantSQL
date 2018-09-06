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
	bi "github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	"github.com/CovenantSQL/CovenantSQL/proto"
)

type accountTxEntries struct {
	account     proto.AccountAddress
	baseNonce   bi.AccountNonce
	transacions []bi.Transaction
}

func newAccountTxEntries(
	addr proto.AccountAddress, baseNonce bi.AccountNonce) (_ *accountTxEntries,
) {
	return &accountTxEntries{
		account:     addr,
		baseNonce:   baseNonce,
		transacions: nil,
	}
}

func (e *accountTxEntries) nextNonce() bi.AccountNonce {
	return e.baseNonce + bi.AccountNonce(len(e.transacions))
}

func (e *accountTxEntries) addTx(tx bi.Transaction) {
	e.transacions = append(e.transacions, tx)
}

type txPool struct {
	entries map[proto.AccountAddress]*accountTxEntries
}

func (p *txPool) addTx(tx bi.Transaction, baseNonce bi.AccountNonce) {
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
