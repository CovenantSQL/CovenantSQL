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
	"sync"

	"github.com/CovenantSQL/CovenantSQL/blockproducer/types"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/proto"
)

// txIndex indexes the tx blockchain receive.
type txIndex struct {
	mu sync.Mutex

	billingHashIndex map[hash.Hash]*types.TxBilling
	// lastBillingIndex indexes last appearing txBilling of DatabaseID
	// to ensure the nonce of txbilling monotone increasing
	lastBillingIndex map[*proto.DatabaseID]uint32
}

// newTxIndex creates a new TxIndex.
func newTxIndex() *txIndex {
	ti := txIndex{
		billingHashIndex: make(map[hash.Hash]*types.TxBilling),
		lastBillingIndex: make(map[*proto.DatabaseID]uint32),
	}
	return &ti
}

// addTxBilling adds a checked TxBilling in the TxIndex.
func (ti *txIndex) addTxBilling(tb *types.TxBilling) error {
	ti.mu.Lock()
	defer ti.mu.Unlock()

	if v, ok := ti.billingHashIndex[*tb.TxHash]; ok {
		// TODO(lambda): ensure whether the situation will happen
		if v == nil {
			return ErrCorruptedIndex
		}
	}

	ti.billingHashIndex[*tb.TxHash] = tb

	return nil
}

// updateLatTxBilling updates the last billing index of specific databaseID.
func (ti *txIndex) updateLastTxBilling(databaseID *proto.DatabaseID, sequenceID uint32) (err error) {
	ti.mu.Lock()
	defer ti.mu.Unlock()

	if v, ok := ti.lastBillingIndex[databaseID]; ok {
		if v >= sequenceID {
			return ErrSmallerSequenceID
		}
		ti.lastBillingIndex[databaseID] = sequenceID
	}
	ti.lastBillingIndex[databaseID] = sequenceID
	return
}

// fetchUnpackedTxBillings fetch all txbillings in index.
func (ti *txIndex) fetchUnpackedTxBillings() []*types.TxBilling {
	ti.mu.Lock()
	defer ti.mu.Unlock()

	txes := make([]*types.TxBilling, 0, 1024)

	for _, t := range ti.billingHashIndex {
		if t != nil && t.SignedBlock == nil {
			txes = append(txes, t)
		}
	}
	return txes
}

// hasTxBilling look up the specific txbilling in index.
func (ti *txIndex) hasTxBilling(h *hash.Hash) bool {
	_, ok := ti.billingHashIndex[*h]
	return ok
}

// getTxBilling look up the specific txbilling in index.
func (ti *txIndex) getTxBilling(h *hash.Hash) *types.TxBilling {
	val := ti.billingHashIndex[*h]
	return val
}

// lastSequenceID look up the last sequenceID of specific databaseID.
func (ti *txIndex) lastSequenceID(databaseID *proto.DatabaseID) (uint32, error) {
	if seqID, ok := ti.lastBillingIndex[databaseID]; ok {
		return seqID, nil
	}
	return 0, ErrNoSuchTxBilling
}
