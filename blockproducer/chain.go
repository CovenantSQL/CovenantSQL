/*
 * Copyright 2018 The ThunderDB Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the “License”);
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an “AS IS” BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package blockproducer

import (
	"time"

	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
	"gitlab.com/thunderdb/ThunderDB/proto"

	"github.com/coreos/bbolt"
	"gitlab.com/thunderdb/ThunderDB/blockproducer/types"
	"gitlab.com/thunderdb/ThunderDB/crypto/kms"
	"gitlab.com/thunderdb/ThunderDB/utils/log"
)

var (
	metaBucket               = [4]byte{0x0, 0x0, 0x0, 0x0}
	metaStateKey             = []byte("thunderdb-state")
	metaBlockIndexBucket     = []byte("thunderdb-block-index-bucket")
	metaTxBillingIndexBucket = []byte("thunderdb-tx-billing-index-bucket")

	accountAddress = proto.AccountAddress(hash.Hash{104, 36, 47, 65, 118, 196, 75, 11, 253, 85, 3, 128, 41, 109,
		167, 180, 119, 64, 83, 185, 214, 103, 74, 9, 125, 14, 139, 16, 107, 112, 144, 55})
)

// Chain defines the main chain
type Chain struct {
	db *bolt.DB
	bi *blockIndex
	ti *txIndex
	rt *runtime
	st *state
}

func NewChain(cfg *config) (*Chain, error) {
	// open db file
	db, err := bolt.Open(cfg.dataFile, 0600, nil)
	if err != nil {
		return nil, err
	}

	// create bucket for meta data
	err = db.Update(func(tx *bolt.Tx) (err error) {
		bucket, err := tx.CreateBucketIfNotExists(metaBucket[:])

		if err != nil {
			return
		}

		_, err = bucket.CreateBucketIfNotExists(metaBlockIndexBucket)
		if err != nil {
			return
		}

		_, err = bucket.CreateBucketIfNotExists(metaTxBillingIndexBucket)
		if err != nil {
			return
		}
		return
	})
	if err != nil {
		return nil, err
	}

	// create chain
	chain := &Chain{
		db: db,
		bi: newBlockIndex(cfg),
		ti: newTxIndex(),
		rt: newRuntime(cfg, accountAddress),
		st: &state{
			node:   nil,
			Head:   cfg.genesis.SignedHeader.BlockHash,
			Height: -1,
		},
	}

	// TODO(lambda): push genesis block into the chain

	// TODO(lambda): start the service
	return chain, nil
}

func LoadChain(cfg *config) (chain *Chain, err error) {
	// open db file
	db, err := bolt.Open(cfg.dataFile, 0600, nil)

	if err != nil {
		return nil, err
	}

	chain = &Chain{
		db: db,
		bi: newBlockIndex(cfg),
		ti: newTxIndex(),
		rt: newRuntime(cfg, accountAddress),
		st: &state{},
	}

	err = chain.db.View(func(tx *bolt.Tx) error {
		meta := tx.Bucket(metaBucket[:])
		err = chain.st.deserialize(meta.Get(metaStateKey))

		if err != nil {
			return err
		}

		var last *blockNode
		var index int32
		blocks := meta.Bucket(metaBlockIndexBucket)
		nodes := make([]blockNode, blocks.Stats().KeyN)

		err = blocks.ForEach(func(k, v []byte) (err error) {
			block := &types.Block{}

			if err = block.Deserialize(v); err != nil {
				return err
			}

			log.Debugf("Read new block: %+v", block)
			parent := (*blockNode)(nil)

			if last == nil {
				// TODO(lambda): check genesis block
			} else if block.SignedHeader.ParentHash.IsEqual(&last.hash) {
				if err = block.SignedHeader.Verify(); err != nil {
					return err
				}

				parent = last
			} else {
				parent = chain.bi.lookupBlock(block.SignedHeader.BlockHash)

				if parent == nil {
					return ErrParentNotFound
				}
			}

			nodes[index].initBlockNode(block, parent)
			last = &nodes[index]
			index++
			return err
		})
		if err != nil {
			return err
		}

		txbillings := meta.Bucket(metaTxBillingIndexBucket)
		err = txbillings.ForEach(func(k, v []byte) error {
			txbilling := types.TxBilling{}
			err = txbilling.Deserialize(v)
			if err != nil {
				return err
			}
			chain.ti.AddTxBilling(&txbilling)
			return err
		})

		return nil
	})
	if err != nil {
		return nil, err
	}

	return chain, nil
}

func (c *Chain) pushBlock(b *types.Block) error {
	node := newBlockNode(b, c.st.node)
	state := state{
		node:   node,
		Head:   node.hash,
		Height: node.height,
	}

	encBlock, err := b.Serialize()
	if err != nil {
		return err
	}

	encState, err := c.st.serialize()
	if err != err {
		return err
	}

	err = c.db.Update(func(tx *bolt.Tx) error {
		err = tx.Bucket(metaBucket[:]).Put(metaStateKey, encState)
		if err != nil {
			return err
		}
		err = tx.Bucket(metaBucket[:]).Bucket(metaBlockIndexBucket).Put(node.indexKey(), encBlock)
		if err != nil {
			return err
		}
		return err
	})
	if err != nil {
		return err
	}
	c.st = &state
	c.bi.addBlock(node)
	return nil
}

func (c *Chain) pushTxBilling(tb *types.TxBilling) error {
	encTx, err := tb.Serialize()
	if err != nil {
		return err
	}

	err = c.db.Update(func(tx *bolt.Tx) error {
		err = tx.Bucket(metaBucket[:]).Bucket(metaTxBillingIndexBucket).Put(tb.TxHash[:], encTx)
		return err
	})
	if err != nil {
		return err
	}
	c.ti.hashIndex[*tb.TxHash] = tb
	return nil
}

func (c *Chain) produceBlock(now time.Time) error {
	priv, err := kms.GetLocalPrivateKey()
	if err != nil {
		return err
	}

	b := types.Block{
		SignedHeader: types.SignedHeader{
			Header: types.Header{
				Version:    blockVersion,
				Producer:   c.rt.accountAddress,
				ParentHash: c.st.Head,
				Timestamp:  now,
			},
		},
		TxBillings: c.ti.FetchTxBillings(),
	}

	err = b.PackAndSignBlock(priv)

	if err != nil {
		return err
	}
	return nil
}
