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

package chain

import (
	"sync"

	ci "github.com/CovenantSQL/CovenantSQL/chain/interfaces"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
)

type txCache struct {
	bh *hash.Hash
	tx ci.Transaction
}

// TxIndex defines transaction index.
type TxIndex struct {
	index sync.Map
}

// NewTxIndex returns a new TxIndex instance.
func NewTxIndex() *TxIndex {
	return &TxIndex{}
}

// StoreTx stores tx in the transaction index.
func (i *TxIndex) StoreTx(tx ci.Transaction) {
	i.index.Store(tx.GetIndexKey(), &txCache{tx: tx})
}

// HasTx returns a boolean value indicating wether the transaction index has key or not.
func (i *TxIndex) HasTx(key interface{}) (ok bool) {
	_, ok = i.index.Load(key)
	return
}

// LoadTx loads a transaction with key.
func (i *TxIndex) LoadTx(key interface{}) (tx ci.Transaction, ok bool) {
	var (
		val interface{}
		tc  *txCache
	)
	if val, ok = i.index.Load(key); ok {
		if tc = val.(*txCache); tc != nil {
			tx = tc.tx
		}
	}
	return
}

// SetBlock sets the block hash filed of txCache with key in the transaction index.
func (i *TxIndex) SetBlock(key interface{}, bh hash.Hash) (ok bool) {
	var (
		val interface{}
		tc  *txCache
	)
	if val, ok = i.index.Load(key); ok {
		if tc = val.(*txCache); tc != nil {
			tc.bh = &bh
		}
	}
	return
}

// DelTx deletes transaction with key in the transaction index.
func (i *TxIndex) DelTx(key interface{}) {
	i.index.Delete(key)
}

// ResetBlock resets the block hash field of txCache with key in the transaction index.
func (i *TxIndex) ResetBlock(key interface{}) (ok bool) {
	var (
		val interface{}
		tc  *txCache
	)
	if val, ok = i.index.Load(key); ok {
		if tc = val.(*txCache); tc != nil {
			tc.bh = nil
		}
	}
	return
}

// CheckTxState checks the transaction state for block packing with key in the transaction index.
func (i *TxIndex) CheckTxState(key interface{}) error {
	var (
		ok  bool
		val interface{}
	)
	if val, ok = i.index.Load(key); !ok {
		return ErrUnknownTx
	}
	if tc := val.(*txCache); tc == nil {
		return ErrCorruptedIndex
	} else if tc.bh != nil {
		return ErrDuplicateTx
	}
	return nil
}

// FetchUnpackedTxes fetches all unpacked tranactions and returns them as a slice.
func (i *TxIndex) FetchUnpackedTxes() (txes []ci.Transaction) {
	i.index.Range(func(key interface{}, val interface{}) bool {
		if tc := val.(*txCache); tc != nil && tc.bh == nil {
			txes = append(txes, tc.tx)
		}
		return true
	})
	return
}
