/*
 * Copyright 2018 The ThunderDB Authors.
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

package chain

import (
	"sync"

	ci "gitlab.com/thunderdb/ThunderDB/chain/interfaces"
	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
)

type txCache struct {
	bh *hash.Hash
	tx ci.Transaction
}

type TxIndex struct {
	sync.Map
}

func NewTxIndex() *TxIndex {
	return &TxIndex{}
}

func (i *TxIndex) StoreTx(tx ci.Transaction) {
	i.Store(tx.GetHash(), &txCache{tx: tx})
}

func (i *TxIndex) HasTx(th hash.Hash) (ok bool) {
	_, ok = i.Load(th)
	return
}

func (i *TxIndex) LoadTx(th hash.Hash) (tx ci.Transaction, ok bool) {
	var (
		val interface{}
		tc  *txCache
	)
	if val, ok = i.Load(th); ok {
		if tc = val.(*txCache); tc != nil {
			tx = tc.tx
		}
	}
	return
}

func (i *TxIndex) SetBlock(th hash.Hash, bh hash.Hash) (ok bool) {
	var (
		val interface{}
		tc  *txCache
	)
	if val, ok = i.Load(th); ok {
		if tc = val.(*txCache); tc != nil {
			tc.bh = &bh
		}
	}
	return
}

func (i *TxIndex) ResetBlock(th hash.Hash) (ok bool) {
	var (
		val interface{}
		tc  *txCache
	)
	if val, ok = i.Load(th); ok {
		if tc = val.(*txCache); tc != nil {
			tc.bh = nil
		}
	}
	return
}

func (i *TxIndex) FetchUnpackedTxes() (txes []ci.Transaction) {
	i.Range(func(key interface{}, val interface{}) bool {
		if tc := val.(*txCache); tc != nil && tc.bh == nil {
			txes = append(txes, tc.tx)
		}
		return true
	})
	return
}
