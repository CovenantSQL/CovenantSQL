package blockproducer

import (
	"sync"

	"gitlab.com/thunderdb/ThunderDB/proto"

	"gitlab.com/thunderdb/ThunderDB/blockproducer/types"
	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
)

// TxIndex indexes the tx blockchain receive.
type txIndex struct {
	mu sync.Mutex

	billingHashIndex map[hash.Hash]*types.TxBilling
	// lastBillingIndex indexes last appearing txBilling of DatabaseID
	// to ensure the nonce of txbilling monotone increasing
	lastBillingIndex map[*proto.DatabaseID]uint64
}

// newTxIndex creates a new TxIndex.
func newTxIndex() *txIndex {
	txIndex := txIndex{
		billingHashIndex: make(map[hash.Hash]*types.TxBilling),
		lastBillingIndex: make(map[*proto.DatabaseID]uint64),
	}
	return &txIndex
}

// addTxBilling adds a checked TxBilling in the TxIndex.
func (ti *txIndex) addTxBilling(tb *types.TxBilling) (err error) {
	ti.mu.Lock()
	defer ti.mu.Unlock()

	if v, ok := ti.billingHashIndex[*tb.TxHash]; ok {
		// TODO(lambda): ensure whether the situation will happen
		if v == nil {
			err = ErrCorruptedIndex
		}
	}

	ti.billingHashIndex[*tb.TxHash] = tb

	return nil
}

// updateLatTxBilling updates the last billing index of specific databaseID
func (ti *txIndex) updateLastTxBilling(databaseID *proto.DatabaseID, sequenceID uint64) (err error) {
	ti.mu.Lock()
	defer ti.mu.Unlock()

	if v, ok := ti.lastBillingIndex[databaseID]; ok {
		if v >= sequenceID {
			return ErrSmallerSequenceID
		} else {
			ti.lastBillingIndex[databaseID] = sequenceID
		}
	} else {
		ti.lastBillingIndex[databaseID] = sequenceID
	}
	return nil
}

// fetchUnpackedTxBillings fetch all txbillings in index
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

// hasTxBilling look up the specific txbilling in index
func (ti *txIndex) hasTxBilling(h *hash.Hash) bool {
	_, ok := ti.billingHashIndex[*h]
	return ok
}

// getTxBilling look up the specific txbilling in index
func (ti *txIndex) getTxBilling(h *hash.Hash) *types.TxBilling {
	val, _ := ti.billingHashIndex[*h]
	return val
}

// lastSequenceID look up the last sequenceID of specific databaseID
func (ti *txIndex) lastSequenceID(databaseID *proto.DatabaseID) (uint64, error) {
	if seqID, ok := ti.lastBillingIndex[databaseID]; ok {
		return seqID, nil
	} else {
		return 0, ErrNoSuchTxBilling
	}
}
