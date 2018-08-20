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
	bolt "github.com/coreos/bbolt"
	ci "gitlab.com/thunderdb/ThunderDB/chain/interfaces"
)

var (
	metaBucket        = [4]byte{0x0, 0x0, 0x0, 0x0}
	metaTxIndexBucket = []byte("covenantsql-tx-index-bucket")
)

// TxPersistence defines a persistence storage for blockchain transactions.
type TxPersistence struct {
	db *bolt.DB
}

// NewTxPersistence returns a new TxPersistence instance using the given bolt database as
// underlying storage engine.
func NewTxPersistence(db *bolt.DB) (ins *TxPersistence, err error) {
	// Initialize buckets
	if err = db.Update(func(tx *bolt.Tx) (err error) {
		meta, err := tx.CreateBucketIfNotExists(metaBucket[:])
		if err != nil {
			return
		}
		_, err = meta.CreateBucketIfNotExists(metaTxIndexBucket)
		return
	}); err != nil {
		// Create instance if succeed
		ins = &TxPersistence{db: db}
	}
	return
}

// PutTransaction serializes and puts the transaction tx into the storage.
func (p *TxPersistence) PutTransaction(tx ci.Transaction) (err error) {
	var key, value []byte
	key = tx.GetPersistenceKey()
	if value, err = tx.Serialize(); err != nil {
		return
	}
	return p.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(metaBucket[:]).Bucket(metaTxIndexBucket).Put(key, value)
	})
}

// GetTransaction gets the transaction binary representation from the storage with key and
// deserialize to tx.
//
// It is important that tx must provide an interface with corresponding concrete value, or the
// deserialization will cause unexpected error.
func (p *TxPersistence) GetTransaction(key []byte, tx ci.Transaction) (err error) {
	var value []byte
	if err = p.db.View(func(tx *bolt.Tx) error {
		value = tx.Bucket(metaBucket[:]).Bucket(metaTxIndexBucket).Get(key)
		return nil
	}); err != nil {
		return
	}
	return tx.Deserialize(value)
}

// DelTransaction deletes the transaction from the storage with key.
func (p *TxPersistence) DelTransaction(key []byte) (err error) {
	return p.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(metaBucket[:]).Bucket(metaTxIndexBucket).Delete(key)
	})
}

// PutTransactionAndUpdateIndex serializes and puts the transaction from the storage with key
// and updates transaction index ti in a single database transaction.
func (p *TxPersistence) PutTransactionAndUpdateIndex(tx ci.Transaction, ti *TxIndex) (err error) {
	var (
		key = tx.GetPersistenceKey()
		val []byte
	)
	if val, err = tx.Serialize(); err != nil {
		return
	}
	return p.db.Update(func(dbtx *bolt.Tx) (err error) {
		if err = dbtx.Bucket(metaBucket[:]).Bucket(metaTxIndexBucket).Put(key, val); err != nil {
			return
		}
		ti.StoreTx(tx)
		return
	})
}

// DelTransactionAndUpdateIndex deletes the transaction from the storage with key and updates
// transaction index ti in a single database transaction.
func (p *TxPersistence) DelTransactionAndUpdateIndex(
	ikey interface{}, pkey []byte, ti *TxIndex) (err error,
) {
	return p.db.Update(func(dbtx *bolt.Tx) (err error) {
		if err = dbtx.Bucket(metaBucket[:]).Bucket(metaTxIndexBucket).Delete(pkey); err != nil {
			return
		}
		ti.DelTx(ikey)
		return
	})
}
