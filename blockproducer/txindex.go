package blockproducer

import (
	"sync"

	"gitlab.com/thunderdb/ThunderDB/blockproducer/types"
	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
)

// TxIndex indexes the tx blockchain receive.
type txIndex struct {
	mu sync.Mutex

	hashIndex map[hash.Hash]*types.TxBilling
}

// newTxIndex creates a new TxIndex.
func newTxIndex() *txIndex {
	txIndex := txIndex{
		hashIndex: make(map[hash.Hash]*types.TxBilling),
	}
	return &txIndex
}

// AddTxBilling adds a checked TxBilling in the TxIndex.
func (ti *txIndex) addTxBilling(tb *types.TxBilling) (err error) {
	ti.mu.Lock()
	defer ti.mu.Unlock()

	if v, ok := ti.hashIndex[*tb.TxHash]; ok {
		// TODO(lambda): ensure whether the situation will happen
		if v == nil {
			err = ErrCorruptedIndex
		}
	}

	ti.hashIndex[*tb.TxHash] = tb

	return nil
}

func (ti *txIndex) fetchTxBillings() []*types.TxBilling {
	ti.mu.Lock()
	defer ti.mu.Unlock()

	txes := make([]*types.TxBilling, 0, 1024)

	for _, t := range ti.hashIndex {
		if t != nil && t.SignedBlock == nil {
			txes = append(txes, t)
		}
	}
	return txes
}

func (ti *txIndex) hasTxBilling(h *hash.Hash) bool {
	_, ok := ti.hashIndex[*h]
	return ok
}
