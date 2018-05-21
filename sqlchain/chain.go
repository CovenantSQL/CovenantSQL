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

package sqlchain

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"

	"github.com/btcsuite/btcd/btcec"
	bolt "github.com/coreos/bbolt"
	"github.com/thunderdb/ThunderDB/crypto/hash"
)

var (
	metaBucket           = [4]byte{0x0, 0x0, 0x0, 0x0}
	metaStateKey         = []byte("thunderdb-state")
	metaBlockIndexBucket = []byte("thunderdb-block-index-bucket")
)

// State represents a snapshot of current best chain.
type State struct {
	node   *blockNode
	Head   [32]byte
	Height int32
}

// Chain represents a sql-chain.
type Chain struct {
	cfg          *Config
	db           *bolt.DB
	index        *blockIndex
	pendingBlock *Block
	state        *State
}

func init() {
	gob.Register(btcec.KoblitzCurve{})
}

// NewChain creates a new sql-chain struct.
func NewChain(cfg *Config) (chain *Chain, err error) {
	db, err := bolt.Open(cfg.DataDir, 0600, nil)

	if err != nil {
		return nil, err
	}

	chain = &Chain{
		cfg:          cfg,
		db:           db,
		index:        newBlockIndex(cfg),
		pendingBlock: &Block{},
		state:        &State{},
	}

	if err = chain.InitChain(); err != nil {
		return nil, err
	}

	return chain, err
}

// TODO(leventeliu): implement this function
func verifyGenesis() bool {
	return true
}

func blockIndexKey(blockHash *hash.Hash, height uint32) []byte {
	indexKey := make([]byte, hash.HashSize+4)
	binary.BigEndian.PutUint32(indexKey[0:4], height)
	copy(indexKey[4:hash.HashSize], blockHash[:])
	return indexKey
}

// InitChain initializes the chain state from the specified database and rebuilds a memory index.
func (c *Chain) InitChain() (err error) {
	initialized := false

	err = c.db.View(func(tx *bolt.Tx) (err error) {
		bucket := tx.Bucket(metaBucket[:])

		if bucket != nil && bucket.Get(metaStateKey) != nil &&
			bucket.Bucket(metaBlockIndexBucket) != nil {
			initialized = true
		}

		return nil
	})

	if err != nil {
		return err
	}

	if !initialized {
		return c.createGenesis()
	}

	return c.db.View(func(tx *bolt.Tx) (err error) {
		// Read state struct
		bucket := tx.Bucket(metaBucket[:])
		buffer := bytes.NewBuffer(bucket.Get(metaStateKey))
		dec := gob.NewDecoder(buffer)
		err = dec.Decode(c.state)

		if err != nil {
			return err
		}

		// Rebuild memory index
		blockCount := int32(0)
		bi := bucket.Bucket(metaBlockIndexBucket)
		cursor := bi.Cursor()

		for k, _ := cursor.First(); k != nil; k, _ = cursor.Next() {
			blockCount++
		}

		lastNode := (*blockNode)(nil)
		index := int32(0)
		nodes := make([]blockNode, blockCount)
		cursor = bi.Cursor()

		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			header := SignedHeader{}
			buffer := bytes.NewBuffer(v)
			dec := gob.NewDecoder(buffer)
			err := dec.Decode(&header)
			parent := (*blockNode)(nil)

			if err != nil {
				return err
			}

			if lastNode == nil {
				if !verifyGenesis() {
					return fmt.Errorf("initChain: failed to verify genesis block")
				}
			} else if header.ParentHash == lastNode.hash {
				parent = lastNode
			} else {
				parent = c.index.LookupNode(&header.ParentHash)

				if parent == nil {
					return fmt.Errorf("initChain: could not find parent node")
				}
			}

			nodes[index].initBlockNode(&header, parent)
			index++
		}

		return nil
	})
}

// TODO(leventeliu): implement this method
func (c *Chain) createGenesis() (err error) {
	return nil
}

// PushBlock pushes the signed block header to extend the current main chain.
func (c *Chain) PushBlock(block *SignedHeader) (err error) {
	// Pushed block must extend the best chain
	if block.Header.ParentHash != hash.Hash(c.state.Head) {
		return fmt.Errorf("pushBlock: new block must extend the best chain")
	}

	// Update best state
	c.state.node = newBlockNode(block, c.state.node)
	c.state.Head = [32]byte(block.BlockHash)
	c.state.Height++

	// Update index
	c.index.AddBlock(c.state.node)

	// Write to db
	return c.db.Update(func(tx *bolt.Tx) (err error) {
		var buffer bytes.Buffer
		enc := gob.NewEncoder(&buffer)
		err = enc.Encode(block)

		if err != nil {
			return err
		}

		key := blockIndexKey(&block.BlockHash, uint32(c.state.Height))
		err = tx.Bucket(metaBucket[:]).Bucket(metaBlockIndexBucket).Put(key, buffer.Bytes())

		if err != nil {
			return err
		}

		buffer.Reset()
		err = enc.Encode(c.state)

		if err != nil {
			return err
		}

		err = tx.Bucket(metaBucket[:]).Put(metaStateKey, buffer.Bytes())

		return nil
	})
}
