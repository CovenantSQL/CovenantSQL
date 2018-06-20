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

	bolt "github.com/coreos/bbolt"
	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
	"gitlab.com/thunderdb/ThunderDB/utils"
)

var (
	metaBucket           = [4]byte{0x0, 0x0, 0x0, 0x0}
	metaStateKey         = []byte("thunderdb-state")
	metaBlockIndexBucket = []byte("thunderdb-block-index-bucket")
)

// State represents a snapshot of current best chain.
type State struct {
	node   *blockNode
	Head   hash.Hash
	Height int32
}

func (s *State) marshal() ([]byte, error) {
	buffer := bytes.NewBuffer(nil)

	if err := utils.WriteElements(buffer, binary.BigEndian,
		s.Head,
		s.Height,
	); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func (s *State) unmarshal(b []byte) (err error) {
	reader := bytes.NewReader(b)
	return utils.ReadElements(reader, binary.BigEndian,
		&s.Head,
		&s.Height,
	)
}

// Chain represents a sql-chain.
type Chain struct {
	cfg          *Config
	db           *bolt.DB
	index        *blockIndex
	pendingBlock *Block
	state        *State
}

// NewChain creates a new sql-chain struct.
func NewChain(cfg *Config) (chain *Chain, err error) {
	err = cfg.Genesis.VerifyAsGenesis()

	if err != nil {
		return
	}

	// Open DB file
	db, err := bolt.Open(cfg.DataDir, 0600, nil)

	if err != nil {
		return
	}

	// Create buckets for chain meta
	err = db.Update(func(tx *bolt.Tx) (err error) {
		bucket, err := tx.CreateBucketIfNotExists(metaBucket[:])

		if err != nil {
			return
		}

		_, err = bucket.CreateBucketIfNotExists(metaBlockIndexBucket)
		return
	})

	if err != nil {
		return
	}

	// Create chain state
	chain = &Chain{
		cfg:          cfg,
		db:           db,
		index:        newBlockIndex(cfg),
		pendingBlock: &Block{},
		state: &State{
			node:   nil,
			Head:   cfg.Genesis.SignedHeader.RootHash,
			Height: -1,
		},
	}

	err = chain.PushBlock(cfg.Genesis.SignedHeader)

	if err != nil {
		return nil, err
	}

	return
}

func blockIndexKey(blockHash *hash.Hash, height uint32) []byte {
	indexKey := make([]byte, hash.HashSize+4)
	binary.BigEndian.PutUint32(indexKey[0:4], height)
	copy(indexKey[4:hash.HashSize], blockHash[:])
	return indexKey
}

// LoadChain loads the chain state from the specified database and rebuilds a memory index.
func LoadChain(cfg *Config) (chain *Chain, err error) {
	// Open DB file
	db, err := bolt.Open(cfg.DataDir, 0600, nil)

	if err != nil {
		return
	}

	// Create chain state
	chain = &Chain{
		cfg:          cfg,
		db:           db,
		index:        newBlockIndex(cfg),
		pendingBlock: &Block{},
		state:        &State{},
	}

	err = chain.db.View(func(tx *bolt.Tx) (err error) {
		// Read state struct
		bucket := tx.Bucket(metaBucket[:])
		err = chain.state.unmarshal(bucket.Get(metaStateKey))

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
			header := &SignedHeader{}
			err = header.unmarshal(v)

			if err != nil {
				return err
			}

			parent := (*blockNode)(nil)

			if lastNode == nil {
				if err = header.VerifyAsGenesis(); err != nil {
					return
				}
			} else if header.ParentHash == lastNode.hash {
				if err = header.Verify(); err != nil {
					return
				}

				parent = lastNode
			} else {
				parent = chain.index.LookupNode(&header.ParentHash)

				if parent == nil {
					return ErrParentNotFound
				}
			}

			nodes[index].initBlockNode(header, parent)
			lastNode = &nodes[index]
			index++
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return
}

// PushBlock pushes the signed block header to extend the current main chain.
func (c *Chain) PushBlock(block *SignedHeader) (err error) {
	// Pushed block must extend the best chain
	if block.Header.ParentHash != hash.Hash(c.state.Head) {
		return ErrInvalidBlock
	}

	// Update best state
	c.state.node = newBlockNode(block, c.state.node)
	c.state.Head = [32]byte(block.BlockHash)
	c.state.Height++

	// Update index
	c.index.AddBlock(c.state.node)

	// Write to db
	return c.db.Update(func(tx *bolt.Tx) (err error) {
		buffer, err := block.marshal()

		if err != nil {
			return err
		}

		key := c.state.node.indexKey()
		err = tx.Bucket(metaBucket[:]).Bucket(metaBlockIndexBucket).Put(key, buffer)

		if err != nil {
			return err
		}

		buffer, err = c.state.marshal()

		if err != nil {
			return err
		}

		err = tx.Bucket(metaBucket[:]).Put(metaStateKey, buffer)

		return
	})
}
