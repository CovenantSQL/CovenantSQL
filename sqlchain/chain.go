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

package sqlchain

import (
	"encoding/binary"
	"time"

	bolt "github.com/coreos/bbolt"
	log "github.com/sirupsen/logrus"
	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
	"gitlab.com/thunderdb/ThunderDB/crypto/kms"
	"gitlab.com/thunderdb/ThunderDB/rpc"
	ct "gitlab.com/thunderdb/ThunderDB/sqlchain/types"
	wt "gitlab.com/thunderdb/ThunderDB/worker/types"
)

var (
	metaBucket              = [4]byte{0x0, 0x0, 0x0, 0x0}
	metaStateKey            = []byte("thunderdb-state")
	metaBlockIndexBucket    = []byte("thunderdb-block-index-bucket")
	metaHeightIndexBucket   = []byte("thunderdb-query-height-index-bucket")
	metaRequestIndexBucket  = []byte("thunderdb-query-reqeust-index-bucket")
	metaResponseIndexBucket = []byte("thunderdb-query-response-index-bucket")
	metaAckIndexBucket      = []byte("thunderdb-query-ack-index-bucket")
)

// heightToKey converts a height in int32 to a key in bytes.
func heightToKey(h int32) (key []byte) {
	key = make([]byte, 4)
	binary.BigEndian.PutUint32(key, uint32(h))
	return
}

// keyToHeight converts a height back from a key in bytes.
func keyToHeight(k []byte) int32 {
	return int32(binary.BigEndian.Uint32(k))
}

// Chain represents a sql-chain.
type Chain struct {
	db *bolt.DB
	bi *blockIndex
	qi *queryIndex
	cl *rpc.Caller
	rt *runtime
	st *state

	// Only for test
	tIsMyTurn bool
}

// NewChain creates a new sql-chain struct.
func NewChain(c *Config) (chain *Chain, err error) {
	err = c.Genesis.VerifyAsGenesis()

	if err != nil {
		return
	}

	// Open DB file
	db, err := bolt.Open(c.DataFile, 0600, nil)

	if err != nil {
		return
	}

	// Create buckets for chain meta
	if err = db.Update(func(tx *bolt.Tx) (err error) {
		bucket, err := tx.CreateBucketIfNotExists(metaBucket[:])

		if err != nil {
			return
		}

		if _, err = bucket.CreateBucketIfNotExists(metaBlockIndexBucket); err != nil {
			return
		}

		_, err = bucket.CreateBucketIfNotExists(metaHeightIndexBucket)
		return
	}); err != nil {
		return
	}

	// Create chain state
	chain = &Chain{
		db: db,
		bi: newBlockIndex(c),
		qi: newQueryIndex(),
		cl: rpc.NewCaller(),
		rt: newRunTime(c),
		st: &state{
			node:   nil,
			Head:   c.Genesis.SignedHeader.GenesisHash,
			Height: -1,
		},
	}

	if err = chain.pushBlock(c.Genesis); err != nil {
		return nil, err
	}

	// Start service
	chain.rt.startService(chain)

	return
}

// LoadChain loads the chain state from the specified database and rebuilds a memory index.
func LoadChain(c *Config) (chain *Chain, err error) {
	// Open DB file
	db, err := bolt.Open(c.DataFile, 0600, nil)

	if err != nil {
		return
	}

	// Create chain state
	chain = &Chain{
		db: db,
		bi: newBlockIndex(c),
		qi: newQueryIndex(),
		cl: rpc.NewCaller(),
		rt: newRunTime(c),
		st: &state{},
	}

	err = chain.db.View(func(tx *bolt.Tx) (err error) {
		// Read state struct
		meta := tx.Bucket(metaBucket[:])
		err = chain.st.UnmarshalBinary(meta.Get(metaStateKey))

		if err != nil {
			return err
		}

		// Read blocks and rebuild memory index
		var last *blockNode
		var index int32
		blocks := meta.Bucket(metaBlockIndexBucket)
		nodes := make([]blockNode, blocks.Stats().KeyN)

		if err = blocks.ForEach(func(k, v []byte) (err error) {
			block := &ct.Block{}

			if err = block.UnmarshalBinary(v); err != nil {
				return
			}

			log.Debugf("Read new block: %+v", block)
			parent := (*blockNode)(nil)

			if last == nil {
				if err = block.SignedHeader.VerifyAsGenesis(); err != nil {
					return
				}

				// Set constant fields from genesis block
				chain.rt.setGenesis(block)
			} else if block.SignedHeader.ParentHash.IsEqual(&last.hash) {
				if err = block.SignedHeader.Verify(); err != nil {
					return
				}

				parent = last
			} else {
				parent = chain.bi.lookupNode(&block.SignedHeader.BlockHash)

				if parent == nil {
					return ErrParentNotFound
				}
			}

			nodes[index].initBlockNode(block, parent)
			last = &nodes[index]
			index++
			return
		}); err != nil {
			return
		}

		// Read queries and rebuild memory index
		heights := meta.Bucket(metaHeightIndexBucket)
		resp := &wt.SignedResponseHeader{}
		ack := &wt.SignedAckHeader{}

		if err = heights.ForEach(func(k, v []byte) (err error) {
			h := keyToHeight(k)

			if resps := heights.Bucket(k).Bucket(
				metaResponseIndexBucket); resps != nil {
				if err = resps.ForEach(func(k []byte, v []byte) (err error) {
					if err = resp.UnmarshalBinary(v); err != nil {
						return
					}

					return chain.qi.addResponse(h, resp)
				}); err != nil {
					return
				}
			}

			if acks := heights.Bucket(k).Bucket(metaAckIndexBucket); acks != nil {
				if err = acks.ForEach(func(k []byte, v []byte) (err error) {
					if err = ack.UnmarshalBinary(v); err != nil {
						return
					}

					return chain.qi.addAck(h, ack)
				}); err != nil {
					return
				}
			}

			return
		}); err != nil {
			return
		}

		return
	})

	// Start service
	chain.rt.startService(chain)

	return
}

// pushBlock pushes the signed block header to extend the current main chain.
func (c *Chain) pushBlock(b *ct.Block) (err error) {
	// Prepare and encode
	h := c.rt.getHeightFromTime(b.SignedHeader.Timestamp)
	node := newBlockNode(b, c.st.node)
	st := &state{
		node:   node,
		Head:   node.hash,
		Height: node.height,
	}
	var encBlock, encState []byte

	if encBlock, err = b.MarshalBinary(); err != nil {
		return
	}

	if encState, err = st.MarshalBinary(); err != nil {
		return
	}

	// Update in transaction
	return c.db.Update(func(tx *bolt.Tx) (err error) {
		if err = tx.Bucket(metaBucket[:]).Put(metaStateKey, encState); err != nil {
			return
		}

		if err = tx.Bucket(metaBucket[:]).Bucket(metaBlockIndexBucket).Put(
			node.indexKey(), encBlock); err != nil {
			return
		}

		c.st = st
		c.bi.addBlock(node)
		c.qi.setSignedBlock(h, b)

		return
	})
}

func ensureHeight(tx *bolt.Tx, k []byte) (hb *bolt.Bucket, err error) {
	b := tx.Bucket(metaBucket[:]).Bucket(metaHeightIndexBucket)

	if hb = b.Bucket(k); hb == nil {
		// Create and initialize bucket in new height
		if hb, err = b.CreateBucket(k); err != nil {
			return
		}

		if _, err = hb.CreateBucket(metaRequestIndexBucket); err != nil {
			return
		}

		if _, err = hb.CreateBucket(metaResponseIndexBucket); err != nil {
			return
		}

		if _, err = hb.CreateBucket(metaAckIndexBucket); err != nil {
			return
		}
	}

	return
}

// pushResponedQuery pushes a responsed, signed and verified query into the chain.
func (c *Chain) pushResponedQuery(resp *wt.SignedResponseHeader) (err error) {
	h := c.rt.getHeightFromTime(resp.Request.Timestamp)
	k := heightToKey(h)
	var enc []byte

	if enc, err = resp.MarshalBinary(); err != nil {
		return
	}

	return c.db.Update(func(tx *bolt.Tx) (err error) {
		heightBucket, err := ensureHeight(tx, k)

		if err != nil {
			return
		}

		if err = heightBucket.Bucket(metaResponseIndexBucket).Put(
			resp.HeaderHash[:], enc); err != nil {
			return
		}

		// Always put memory changes which will not be affected by rollback after DB operations
		return c.qi.addResponse(h, resp)
	})
}

// pushAckedQuery pushed a acknowledged, signed and verified query into the chain.
func (c *Chain) pushAckedQuery(ack *wt.SignedAckHeader) (err error) {
	h := c.rt.getHeightFromTime(ack.SignedResponseHeader().Timestamp)
	k := heightToKey(h)
	var enc []byte

	if enc, err = ack.MarshalBinary(); err != nil {
		return
	}

	return c.db.Update(func(tx *bolt.Tx) (err error) {
		b, err := ensureHeight(tx, k)

		if err != nil {
			return
		}

		// TODO(leventeliu): this doesn't seem right to use an error to detect key existence.
		if err = b.Bucket(metaAckIndexBucket).Put(
			ack.HeaderHash[:], enc,
		); err != nil && err != bolt.ErrIncompatibleValue {
			return
		}

		// Always put memory changes which will not be affected by rollback after DB operations
		if err = c.qi.addAck(h, ack); err != nil {
			return
		}

		return
	})
}

// isMyTurn returns whether it's my turn to produce block or not.
//
// TODO(leventliu): need implementation.
func (c *Chain) isMyTurn() bool {
	return c.tIsMyTurn
}

// produceBlock prepares, signs and advises the pending block to the orther peers.
func (c *Chain) produceBlock(now time.Time) (err error) {
	// Retrieve local key pair
	priv, err := kms.GetLocalPrivateKey()

	if err != nil {
		return
	}

	pub, err := kms.GetLocalPublicKey()

	if err != nil {
		return
	}

	// Pack and sign block
	block := &ct.Block{
		SignedHeader: ct.SignedHeader{
			Header: ct.Header{
				Version:     0x01000000,
				Producer:    c.rt.server.ID,
				GenesisHash: c.rt.genesisHash,
				ParentHash:  c.st.Head,
				// MerkleRoot: will be set by Block.PackAndSignBlock(PrivateKey)
				Timestamp: now,
			},
			// BlockHash: will be set by Block.PackAndSignBlock(PrivateKey)
			Signee: pub,
			// Signature: will be set by Block.PackAndSignBlock(PrivateKey)
		},
		Queries: c.qi.markAndCollectUnsignedAcks(c.rt.getNextTurn()),
	}

	if err = block.PackAndSignBlock(priv); err != nil {
		return
	}

	if err = c.pushBlock(block); err != nil {
		return
	}

	// TODO(leventeliu): advise new block
	// ...

	return
}

// runCurrentTurn does the check and runs block producing if its my turn.
func (c *Chain) runCurrentTurn(now time.Time) {
	defer c.rt.setNextTurn()

	if !c.isMyTurn() {
		return
	}

	if err := c.produceBlock(now); err != nil {
		log.Errorf("Failed to produce block: err = %v, now = %s", err, now.Format(time.RFC3339Nano))
		c.Stop()
	}
}

// mainCycle runs main cycle of the sql-chain.
func (c *Chain) mainCycle() {
	defer c.rt.wg.Done()

	for {
		select {
		case <-c.rt.stopCh:
			return
		default:
			if t, d := c.rt.nextTick(); d > 0 {
				time.Sleep(d)
			} else {
				c.runCurrentTurn(t)
			}
		}
	}
}

// sync synchronizes blocks and queries from the other peers.
func (c *Chain) sync() (err error) {
	for {
		now := c.rt.now()
		height := c.rt.getHeightFromTime(now)

		if c.rt.getNextTurn() >= height {
			break
		}

		for c.rt.getNextTurn() <= height {
			// TODO(leventeliu): fetch blocks and queries.
			c.rt.setNextTurn()
		}
	}

	return
}

// Start starts the main process of the sql-chain.
func (c *Chain) Start() (err error) {
	if err = c.sync(); err != nil {
		return
	}

	c.rt.wg.Add(1)
	go c.mainCycle()
	return
}

// Stop stops the main process of the sql-chain.
func (c *Chain) Stop() (err error) {
	// Stop main process
	c.rt.stop()

	// Close database file
	err = c.db.Close()
	return
}

// FetchBlock fetches the block at specified height from local cache.
func (c *Chain) FetchBlock(height int32) (b *ct.Block, err error) {
	if n := c.st.node.ancestor(height); n != nil {
		k := n.indexKey()

		err = c.db.View(func(tx *bolt.Tx) (err error) {
			v := tx.Bucket(metaBucket[:]).Bucket(metaBlockIndexBucket).Get(k)
			err = b.UnmarshalBinary(v)
			return
		})
	}

	return
}

// FetchAckedQuery fetches the acknowledged query from local cache.
func (c *Chain) FetchAckedQuery(height int32, header *hash.Hash) (
	ack *wt.SignedAckHeader, err error,
) {
	return c.qi.getAck(height, header)
}

// CheckAndPushNewBlock implements ChainRPCServer.CheckAndPushNewBlock.
func (c *Chain) CheckAndPushNewBlock(block *ct.Block) (err error) {
	// Pushed block must extend the best chain
	if !block.SignedHeader.ParentHash.IsEqual(&c.st.Head) {
		return ErrInvalidBlock
	}

	// TODO(leventeliu): verify that block.SignedHeader.Producer is the rightful producer of the
	// block.

	// Check block existence
	if c.bi.hasBlock(&block.SignedHeader.BlockHash) {
		return ErrBlockExists
	}

	// Block must produced within [start, end)
	h := c.rt.getHeightFromTime(block.SignedHeader.Timestamp)

	if h != c.st.Height+1 {
		return ErrBlockTimestampOutOfPeriod
	}

	// Check queries
	for _, q := range block.Queries {
		var ok bool

		if ok, err = c.qi.checkAckFromBlock(h, &block.SignedHeader.BlockHash, q); err != nil {
			return
		}

		if !ok {
			// TODO(leventeliu): call RPC.FetchAckedQuery from block.SignedHeader.Producer
		}
	}

	// Verify block signatures
	if err = block.Verify(); err != nil {
		return
	}

	return c.pushBlock(block)
}

// VerifyAndPushResponsedQuery verifies a responsed and signed query, and pushed it if valid.
func (c *Chain) VerifyAndPushResponsedQuery(resp *wt.SignedResponseHeader) (err error) {
	// TODO(leventeliu): check resp.
	if c.rt.queryTimeIsExpired(resp.Timestamp) {
		return ErrQueryExpired
	}

	if err = resp.Verify(); err != nil {
		return
	}

	return c.pushResponedQuery(resp)
}

// VerifyAndPushAckedQuery verifies a acknowledged and signed query, and pushed it if valid.
func (c *Chain) VerifyAndPushAckedQuery(ack *wt.SignedAckHeader) (err error) {
	// TODO(leventeliu): check ack.
	if c.rt.queryTimeIsExpired(ack.SignedResponseHeader().Timestamp) {
		return ErrQueryExpired
	}

	if err = ack.Verify(); err != nil {
		return
	}

	return c.pushAckedQuery(ack)
}
