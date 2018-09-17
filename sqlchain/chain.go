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

package sqlchain

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"reflect"
	"sync"
	"time"

	pt "github.com/CovenantSQL/CovenantSQL/blockproducer/types"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/kayak"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	ct "github.com/CovenantSQL/CovenantSQL/sqlchain/types"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	wt "github.com/CovenantSQL/CovenantSQL/worker/types"
	"github.com/coreos/bbolt"
)

var (
	metaBucket              = [4]byte{0x0, 0x0, 0x0, 0x0}
	metaStateKey            = []byte("covenantsql-state")
	metaBlockIndexBucket    = []byte("covenantsql-block-index-bucket")
	metaHeightIndexBucket   = []byte("covenantsql-query-height-index-bucket")
	metaRequestIndexBucket  = []byte("covenantsql-query-request-index-bucket")
	metaResponseIndexBucket = []byte("covenantsql-query-response-index-bucket")
	metaAckIndexBucket      = []byte("covenantsql-query-ack-index-bucket")
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

	stopCh    chan struct{}
	blocks    chan *ct.Block
	heights   chan int32
	responses chan *wt.ResponseHeader
	acks      chan *wt.AckHeader

	// observerLock defines the lock of observer update operations.
	observerLock sync.Mutex
	// observers defines the observer nodes of current chain.
	observers map[proto.NodeID]int32
	// observerReplicators defines the observer states of current chain.
	observerReplicators map[proto.NodeID]*observerReplicator
	// replCh defines the replication trigger channel for replication check.
	replCh chan struct{}
	// replWg defines the waitGroups for running replications.
	replWg sync.WaitGroup
}

// NewChain creates a new sql-chain struct.
func NewChain(c *Config) (chain *Chain, err error) {
	// TODO(leventeliu): this is a rough solution, you may also want to clean database file and
	// force rebuilding.
	var fi os.FileInfo
	if fi, err = os.Stat(c.DataFile); err == nil && fi.Mode().IsRegular() {
		return LoadChain(c)
	}

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
		db:        db,
		bi:        newBlockIndex(c),
		qi:        newQueryIndex(),
		cl:        rpc.NewCaller(),
		rt:        newRunTime(c),
		stopCh:    make(chan struct{}),
		blocks:    make(chan *ct.Block),
		heights:   make(chan int32, 1),
		responses: make(chan *wt.ResponseHeader),
		acks:      make(chan *wt.AckHeader),

		// Observer related
		observers:           make(map[proto.NodeID]int32),
		observerReplicators: make(map[proto.NodeID]*observerReplicator),
		replCh:              make(chan struct{}),
	}

	if err = chain.pushBlock(c.Genesis); err != nil {
		return nil, err
	}

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
		db:        db,
		bi:        newBlockIndex(c),
		qi:        newQueryIndex(),
		cl:        rpc.NewCaller(),
		rt:        newRunTime(c),
		stopCh:    make(chan struct{}),
		blocks:    make(chan *ct.Block),
		heights:   make(chan int32, 1),
		responses: make(chan *wt.ResponseHeader),
		acks:      make(chan *wt.AckHeader),

		// Observer related
		observers:           make(map[proto.NodeID]int32),
		observerReplicators: make(map[proto.NodeID]*observerReplicator),
		replCh:              make(chan struct{}),
	}

	err = chain.db.View(func(tx *bolt.Tx) (err error) {
		// Read state struct
		meta := tx.Bucket(metaBucket[:])
		metaEnc := meta.Get(metaStateKey)
		if metaEnc == nil {
			return ErrMetaStateNotFound
		}
		st := &state{}
		if err = utils.DecodeMsgPack(metaEnc, st); err != nil {
			return err
		}

		log.WithFields(log.Fields{
			"peer":  chain.rt.getPeerInfoString(),
			"state": st,
		}).Debug("Loading state from database")

		// Read blocks and rebuild memory index
		var last *blockNode
		var index int32
		blocks := meta.Bucket(metaBlockIndexBucket)
		nodes := make([]blockNode, blocks.Stats().KeyN)

		if err = blocks.ForEach(func(k, v []byte) (err error) {
			block := &ct.Block{}

			if err = utils.DecodeMsgPack(v, block); err != nil {
				return
			}

			log.WithFields(log.Fields{
				"peer":  chain.rt.getPeerInfoString(),
				"block": block.BlockHash().String(),
			}).Debug("Loading block from database")
			parent := (*blockNode)(nil)

			if last == nil {
				if err = block.VerifyAsGenesis(); err != nil {
					return
				}

				// Set constant fields from genesis block
				chain.rt.setGenesis(block)
			} else if block.ParentHash().IsEqual(&last.hash) {
				if err = block.Verify(); err != nil {
					return
				}

				parent = last
			} else {
				parent = chain.bi.lookupNode(block.BlockHash())

				if parent == nil {
					return ErrParentNotFound
				}
			}

			height := chain.rt.getHeightFromTime(block.Timestamp())
			nodes[index].initBlockNode(height, block, parent)
			chain.bi.addBlock(&nodes[index])
			last = &nodes[index]
			index++
			return
		}); err != nil {
			return
		}

		// Set chain state
		st.node = last
		chain.rt.setHead(st)

		// Read queries and rebuild memory index
		heights := meta.Bucket(metaHeightIndexBucket)

		if err = heights.ForEach(func(k, v []byte) (err error) {
			h := keyToHeight(k)

			if resps := heights.Bucket(k).Bucket(
				metaResponseIndexBucket); resps != nil {
				if err = resps.ForEach(func(k []byte, v []byte) (err error) {
					var resp = &wt.SignedResponseHeader{}
					if err = utils.DecodeMsgPack(v, resp); err != nil {
						return
					}
					log.WithFields(log.Fields{
						"height": h,
						"header": resp.HeaderHash.String(),
					}).Debug("Loaded new resp header")
					return chain.qi.addResponse(h, resp)
				}); err != nil {
					return
				}
			}

			if acks := heights.Bucket(k).Bucket(metaAckIndexBucket); acks != nil {
				if err = acks.ForEach(func(k []byte, v []byte) (err error) {
					var ack = &wt.SignedAckHeader{}
					if err = utils.DecodeMsgPack(v, ack); err != nil {
						return
					}
					log.WithFields(log.Fields{
						"height": h,
						"header": ack.HeaderHash.String(),
					}).Debug("Loaded new ack header")
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

	return
}

// pushBlock pushes the signed block header to extend the current main chain.
func (c *Chain) pushBlock(b *ct.Block) (err error) {
	// Prepare and encode
	h := c.rt.getHeightFromTime(b.Timestamp())
	node := newBlockNode(h, b, c.rt.getHead().node)
	st := &state{
		node:   node,
		Head:   node.hash,
		Height: node.height,
	}
	var encBlock, encState *bytes.Buffer

	if encBlock, err = utils.EncodeMsgPack(b); err != nil {
		return
	}

	if encState, err = utils.EncodeMsgPack(st); err != nil {
		return
	}

	// Update in transaction
	err = c.db.Update(func(tx *bolt.Tx) (err error) {
		if err = tx.Bucket(metaBucket[:]).Put(metaStateKey, encState.Bytes()); err != nil {
			return
		}

		if err = tx.Bucket(metaBucket[:]).Bucket(metaBlockIndexBucket).Put(
			node.indexKey(), encBlock.Bytes()); err != nil {
			return
		}

		c.rt.setHead(st)
		c.bi.addBlock(node)
		c.qi.setSignedBlock(h, b)
		return
	})

	if err == nil {
		log.WithFields(log.Fields{
			"peer":        c.rt.getPeerInfoString()[:14],
			"time":        c.rt.getChainTimeString(),
			"block":       b.BlockHash().String()[:8],
			"producer":    b.Producer()[:8],
			"querycount":  len(b.Queries),
			"blocktime":   b.Timestamp().Format(time.RFC3339Nano),
			"blockheight": c.rt.getHeightFromTime(b.Timestamp()),
			"headblock": fmt.Sprintf("%s <- %s",
				func() string {
					if st.node.parent != nil {
						return st.node.parent.hash.String()[:8]
					}
					return "|"
				}(), st.Head.String()[:8]),
			"headheight": c.rt.getHead().Height,
		}).Info("Pushed new block")
	}

	return
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
	var enc *bytes.Buffer

	if enc, err = utils.EncodeMsgPack(resp); err != nil {
		return
	}

	return c.db.Update(func(tx *bolt.Tx) (err error) {
		heightBucket, err := ensureHeight(tx, k)

		if err != nil {
			return
		}

		if err = heightBucket.Bucket(metaResponseIndexBucket).Put(
			resp.HeaderHash[:], enc.Bytes()); err != nil {
			return
		}

		// Always put memory changes which will not be affected by rollback after DB operations
		return c.qi.addResponse(h, resp)
	})
}

// pushAckedQuery pushes a acknowledged, signed and verified query into the chain.
func (c *Chain) pushAckedQuery(ack *wt.SignedAckHeader) (err error) {
	h := c.rt.getHeightFromTime(ack.SignedResponseHeader().Timestamp)
	k := heightToKey(h)
	var enc *bytes.Buffer

	if enc, err = utils.EncodeMsgPack(ack); err != nil {
		return
	}

	return c.db.Update(func(tx *bolt.Tx) (err error) {
		b, err := ensureHeight(tx, k)

		if err != nil {
			return
		}

		// TODO(leventeliu): this doesn't seem right to use an error to detect key existence.
		if err = b.Bucket(metaAckIndexBucket).Put(
			ack.HeaderHash[:], enc.Bytes(),
		); err != nil {
			return
		}

		// Always put memory changes which will not be affected by rollback after DB operations
		if err = c.qi.addAck(h, ack); err != nil {
			return
		}

		return
	})
}

// produceBlock prepares, signs and advises the pending block to the orther peers.
func (c *Chain) produceBlock(now time.Time) (err error) {
	// Retrieve local key pair
	priv, err := kms.GetLocalPrivateKey()

	if err != nil {
		return
	}

	// Pack and sign block
	block := &ct.Block{
		SignedHeader: ct.SignedHeader{
			Header: ct.Header{
				Version:     0x01000000,
				Producer:    c.rt.getServer().ID,
				GenesisHash: c.rt.genesisHash,
				ParentHash:  c.rt.getHead().Head,
				// MerkleRoot: will be set by Block.PackAndSignBlock(PrivateKey)
				Timestamp: now,
			},
			// BlockHash/Signee/Signature: will be set by Block.PackAndSignBlock(PrivateKey)
		},
		Queries: c.qi.markAndCollectUnsignedAcks(c.rt.getNextTurn()),
	}

	if err = block.PackAndSignBlock(priv); err != nil {
		return
	}

	// Send to pending list
	c.blocks <- block
	log.WithFields(log.Fields{
		"peer":            c.rt.getPeerInfoString(),
		"time":            c.rt.getChainTimeString(),
		"curr_turn":       c.rt.getNextTurn(),
		"using_timestamp": now.Format(time.RFC3339Nano),
		"block_hash":      block.BlockHash().String(),
	}).Debug("Produced new block")

	// Advise new block to the other peers
	req := &MuxAdviseNewBlockReq{
		Envelope: proto.Envelope{
			// TODO(leventeliu): Add fields.
		},
		DatabaseID: c.rt.databaseID,
		AdviseNewBlockReq: AdviseNewBlockReq{
			Block: block,
		},
	}
	peers := c.rt.getPeers()
	wg := &sync.WaitGroup{}

	for _, s := range peers.Servers {
		if s.ID != c.rt.getServer().ID {
			wg.Add(1)
			go func(id proto.NodeID) {
				defer wg.Done()
				resp := &MuxAdviseNewBlockResp{}
				if err := c.cl.CallNode(
					id, route.SQLCAdviseNewBlock.String(), req, resp); err != nil {
					log.WithFields(log.Fields{
						"peer":            c.rt.getPeerInfoString(),
						"time":            c.rt.getChainTimeString(),
						"curr_turn":       c.rt.getNextTurn(),
						"using_timestamp": now.Format(time.RFC3339Nano),
						"block_hash":      block.BlockHash().String(),
					}).WithError(err).Error(
						"Failed to advise new block")
				}
			}(s.ID)
		}
	}

	wg.Wait()

	// fire replication to observers
	c.startStopReplication()

	return
}

func (c *Chain) syncHead() {
	// Try to fetch if the the block of the current turn is not advised yet
	if h := c.rt.getNextTurn() - 1; c.rt.getHead().Height < h {
		var err error
		req := &MuxFetchBlockReq{
			Envelope: proto.Envelope{
				// TODO(leventeliu): Add fields.
			},
			DatabaseID: c.rt.databaseID,
			FetchBlockReq: FetchBlockReq{
				Height: h,
			},
		}
		resp := &MuxFetchBlockResp{}
		peers := c.rt.getPeers()
		succ := false

		for i, s := range peers.Servers {
			if s.ID != c.rt.getServer().ID {
				if err = c.cl.CallNode(
					s.ID, route.SQLCFetchBlock.String(), req, resp,
				); err != nil || resp.Block == nil {
					log.WithFields(log.Fields{
						"peer":        c.rt.getPeerInfoString(),
						"time":        c.rt.getChainTimeString(),
						"remote":      fmt.Sprintf("[%d/%d] %s", i, len(peers.Servers), s.ID),
						"curr_turn":   c.rt.getNextTurn(),
						"head_height": c.rt.getHead().Height,
						"head_block":  c.rt.getHead().Head.String(),
					}).WithError(err).Debug(
						"Failed to fetch block from peer")
				} else {
					c.blocks <- resp.Block
					log.WithFields(log.Fields{
						"peer":        c.rt.getPeerInfoString(),
						"time":        c.rt.getChainTimeString(),
						"remote":      fmt.Sprintf("[%d/%d] %s", i, len(peers.Servers), s.ID),
						"curr_turn":   c.rt.getNextTurn(),
						"head_height": c.rt.getHead().Height,
						"head_block":  c.rt.getHead().Head.String(),
					}).Debug(
						"Fetch block from remote peer successfully")
					succ = true
					break
				}
			}
		}

		if !succ {
			log.WithFields(log.Fields{
				"peer":        c.rt.getPeerInfoString(),
				"time":        c.rt.getChainTimeString(),
				"curr_turn":   c.rt.getNextTurn(),
				"head_height": c.rt.getHead().Height,
				"head_block":  c.rt.getHead().Head.String(),
			}).Debug(
				"Cannot get block from any peer")
		}
	}
}

// runCurrentTurn does the check and runs block producing if its my turn.
func (c *Chain) runCurrentTurn(now time.Time) {
	defer func() {
		c.rt.setNextTurn()
		c.qi.advanceBarrier(c.rt.getMinValidHeight())
		// Info the block processing goroutine that the chain height has grown, so please return
		// any stashed blocks for further check.
		c.heights <- c.rt.getHead().Height
	}()

	log.WithFields(log.Fields{
		"peer":            c.rt.getPeerInfoString(),
		"time":            c.rt.getChainTimeString(),
		"curr_turn":       c.rt.getNextTurn(),
		"head_height":     c.rt.getHead().Height,
		"head_block":      c.rt.getHead().Head.String(),
		"using_timestamp": now.Format(time.RFC3339Nano),
	}).Debug("Run current turn")

	if c.rt.getHead().Height < c.rt.getNextTurn()-1 {
		log.WithFields(log.Fields{
			"peer":            c.rt.getPeerInfoString(),
			"time":            c.rt.getChainTimeString(),
			"curr_turn":       c.rt.getNextTurn(),
			"head_height":     c.rt.getHead().Height,
			"head_block":      c.rt.getHead().Head.String(),
			"using_timestamp": now.Format(time.RFC3339Nano),
		}).Error("A block will be skipped")
	}

	if !c.rt.isMyTurn() {
		return
	}

	if err := c.produceBlock(now); err != nil {
		log.WithFields(log.Fields{
			"peer":            c.rt.getPeerInfoString(),
			"time":            c.rt.getChainTimeString(),
			"curr_turn":       c.rt.getNextTurn(),
			"using_timestamp": now.Format(time.RFC3339Nano),
		}).WithError(err).Error(
			"Failed to produce block")
	}
}

// mainCycle runs main cycle of the sql-chain.
func (c *Chain) mainCycle() {
	defer func() {
		c.rt.wg.Done()
		// Signal worker goroutines to stop
		close(c.stopCh)
	}()

	for {
		select {
		case <-c.rt.stopCh:
			return
		default:
			c.syncHead()

			if t, d := c.rt.nextTick(); d > 0 {
				log.WithFields(log.Fields{
					"peer":            c.rt.getPeerInfoString(),
					"time":            c.rt.getChainTimeString(),
					"next_turn":       c.rt.getNextTurn(),
					"head_height":     c.rt.getHead().Height,
					"head_block":      c.rt.getHead().Head.String(),
					"using_timestamp": t.Format(time.RFC3339Nano),
					"duration":        d,
				}).Debug("Main cycle")
				time.Sleep(d)
			} else {
				c.runCurrentTurn(t)
			}
		}
	}
}

// sync synchronizes blocks and queries from the other peers.
func (c *Chain) sync() (err error) {
	log.WithFields(log.Fields{
		"peer": c.rt.getPeerInfoString(),
		"time": c.rt.getChainTimeString(),
	}).Debug("Synchronizing chain state")

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

func (c *Chain) processBlocks() {
	rsCh := make(chan struct{})
	rsWG := &sync.WaitGroup{}
	returnStash := func(stash []*ct.Block) {
		defer rsWG.Done()
		for _, block := range stash {
			select {
			case c.blocks <- block:
			case <-rsCh:
				return
			}
		}
	}

	defer func() {
		close(rsCh)
		rsWG.Wait()
		c.rt.wg.Done()
	}()

	var stash []*ct.Block
	for {
		select {
		case h := <-c.heights:
			// Return all stashed blocks to pending channel
			log.WithFields(log.Fields{
				"height": h,
				"stashs": len(stash),
			}).Debug("Read new height from channel")
			if stash != nil {
				rsWG.Add(1)
				go returnStash(stash)
				stash = nil
			}
		case block := <-c.blocks:
			height := c.rt.getHeightFromTime(block.Timestamp())
			log.WithFields(log.Fields{
				"peer":         c.rt.getPeerInfoString(),
				"time":         c.rt.getChainTimeString(),
				"curr_turn":    c.rt.getNextTurn(),
				"head_height":  c.rt.getHead().Height,
				"head_block":   c.rt.getHead().Head.String(),
				"block_height": height,
				"block_hash":   block.BlockHash().String(),
			}).Debug("Processing new block")

			if height > c.rt.getNextTurn()-1 {
				// Stash newer blocks for later check
				stash = append(stash, block)
			} else {
				// Process block
				if height < c.rt.getNextTurn()-1 {
					// TODO(leventeliu): check and add to fork list.
				} else {
					if err := c.CheckAndPushNewBlock(block); err != nil {
						log.WithFields(log.Fields{
							"peer":         c.rt.getPeerInfoString(),
							"time":         c.rt.getChainTimeString(),
							"curr_turn":    c.rt.getNextTurn(),
							"head_height":  c.rt.getHead().Height,
							"head_block":   c.rt.getHead().Head.String(),
							"block_height": height,
							"block_hash":   block.BlockHash().String(),
						}).Error("Failed to check and push new block")
					}
				}
			}
			// fire replication to observers
			c.startStopReplication()
		case <-c.stopCh:
			return
		}
	}
}

func (c *Chain) processResponses() {
	// TODO(leventeliu): implement that
	defer c.rt.wg.Done()
	for {
		select {
		case <-c.stopCh:
			return
		}
	}
}

func (c *Chain) processAcks() {
	// TODO(leventeliu): implement that
	defer c.rt.wg.Done()
	for {
		select {
		case <-c.stopCh:
			return
		}
	}
}

// Start starts the main process of the sql-chain.
func (c *Chain) Start() (err error) {
	if err = c.sync(); err != nil {
		return
	}

	c.rt.wg.Add(1)
	go c.processBlocks()
	c.rt.wg.Add(1)
	go c.processResponses()
	c.rt.wg.Add(1)
	go c.processAcks()
	c.rt.wg.Add(1)
	go c.mainCycle()
	c.rt.wg.Add(1)
	go c.replicationCycle()
	c.rt.startService(c)
	return
}

// Stop stops the main process of the sql-chain.
func (c *Chain) Stop() (err error) {
	// Stop main process
	log.WithFields(log.Fields{
		"peer": c.rt.getPeerInfoString(),
		"time": c.rt.getChainTimeString(),
	}).Debug("Stopping chain")
	c.rt.stop()
	log.WithFields(log.Fields{
		"peer": c.rt.getPeerInfoString(),
		"time": c.rt.getChainTimeString(),
	}).Debug("Chain service stopped")
	// Close database file
	err = c.db.Close()
	log.WithFields(log.Fields{
		"peer": c.rt.getPeerInfoString(),
		"time": c.rt.getChainTimeString(),
	}).Debug("Chain database closed")
	return
}

// FetchBlock fetches the block at specified height from local cache.
func (c *Chain) FetchBlock(height int32) (b *ct.Block, err error) {
	if n := c.rt.getHead().node.ancestor(height); n != nil {
		k := n.indexKey()
		err = c.db.View(func(tx *bolt.Tx) (err error) {
			if v := tx.Bucket(metaBucket[:]).Bucket(metaBlockIndexBucket).Get(k); v != nil {
				b = &ct.Block{}
				err = utils.DecodeMsgPack(v, b)
			}

			return
		})
	}

	return
}

// FetchAckedQuery fetches the acknowledged query from local cache.
func (c *Chain) FetchAckedQuery(height int32, header *hash.Hash) (
	ack *wt.SignedAckHeader, err error,
) {
	if ack, err = c.qi.getAck(height, header); err != nil {
		err = c.db.View(func(tx *bolt.Tx) (err error) {
			for i := height - c.rt.queryTTL; i <= height; i++ {
				if b := tx.Bucket(metaBucket[:]).Bucket(metaHeightIndexBucket).Bucket(
					heightToKey(height)); b != nil {
					if v := b.Bucket(metaAckIndexBucket).Get(header[:]); v != nil {
						dec := &wt.SignedAckHeader{}

						if err = utils.DecodeMsgPack(v, dec); err != nil {
							ack = dec
							break
						}
					}
				}
			}

			return
		})
	}

	return
}

// syncAckedQuery uses RPC call to synchronize an acknowledged query from a remote node.
func (c *Chain) syncAckedQuery(height int32, header *hash.Hash, id proto.NodeID) (
	ack *wt.SignedAckHeader, err error,
) {
	req := &MuxFetchAckedQueryReq{
		Envelope: proto.Envelope{
			// TODO(leventeliu): Add fields.
		},
		DatabaseID: c.rt.databaseID,
		FetchAckedQueryReq: FetchAckedQueryReq{
			Height:                height,
			SignedAckedHeaderHash: header,
		},
	}
	resp := &MuxFetchAckedQueryResp{}

	if err = c.cl.CallNode(id, route.SQLCFetchAckedQuery.String(), req, resp); err != nil {
		log.WithFields(log.Fields{
			"peer": c.rt.getPeerInfoString(),
			"time": c.rt.getChainTimeString(),
		}).WithError(err).Error(
			"Failed to fetch acked query")
		return
	}

	if err = c.VerifyAndPushAckedQuery(resp.Ack); err != nil {
		return
	}

	ack = resp.Ack
	return
}

// queryOrSyncAckedQuery tries to query an acknowledged query from local index, and also tries to
// synchronize it from a remote node if not found locally.
func (c *Chain) queryOrSyncAckedQuery(height int32, header *hash.Hash, id proto.NodeID) (
	ack *wt.SignedAckHeader, err error,
) {
	if ack, err = c.FetchAckedQuery(height, header); err != nil || ack != nil || id == c.rt.getServer().ID {
		return
	}

	return c.syncAckedQuery(height, header, id)
}

// CheckAndPushNewBlock implements ChainRPCServer.CheckAndPushNewBlock.
func (c *Chain) CheckAndPushNewBlock(block *ct.Block) (err error) {
	height := c.rt.getHeightFromTime(block.Timestamp())
	head := c.rt.getHead()
	peers := c.rt.getPeers()
	total := int32(len(peers.Servers))
	next := func() int32 {
		if total > 0 {
			return (head.Height + 1) % total
		}
		return -1
	}()
	log.WithFields(log.Fields{
		"peer":        c.rt.getPeerInfoString(),
		"time":        c.rt.getChainTimeString(),
		"block":       block.BlockHash().String(),
		"producer":    block.Producer(),
		"blocktime":   block.Timestamp().Format(time.RFC3339Nano),
		"blockheight": height,
		"blockparent": block.ParentHash().String(),
		"headblock":   head.Head.String(),
		"headheight":  head.Height,
	}).WithError(err).Debug("Checking new block from other peer")

	if head.Height == height && head.Head.IsEqual(block.BlockHash()) {
		// Maybe already set by FetchBlock
		return nil
	} else if !block.ParentHash().IsEqual(&head.Head) {
		// Pushed block must extend the best chain
		return ErrInvalidBlock
	}

	// Verify block signatures
	if err = block.Verify(); err != nil {
		return
	}

	// Short circuit the checking process if it's a self-produced block
	if block.Producer() == c.rt.server.ID {
		return c.pushBlock(block)
	}

	// Check block producer
	index, found := peers.Find(block.Producer())

	if !found {
		return ErrUnknownProducer
	}

	if index != next {
		log.WithFields(log.Fields{
			"peer":     c.rt.getPeerInfoString(),
			"time":     c.rt.getChainTimeString(),
			"expected": next,
			"actual":   index,
		}).WithError(err).Error(
			"Failed to check new block")
		return ErrInvalidProducer
	}

	// TODO(leventeliu): check if too many periods are skipped or store block for future use.
	// if height-c.rt.getHead().Height > X {
	// 	...
	// }

	// Check queries
	for _, q := range block.Queries {
		var ok bool

		if ok, err = c.qi.checkAckFromBlock(height, block.BlockHash(), q); err != nil {
			return
		}

		if !ok {
			if _, err = c.syncAckedQuery(height, q, block.Producer()); err != nil {
				return
			}

			if _, err = c.qi.checkAckFromBlock(height, block.BlockHash(), q); err != nil {
				return
			}
		}
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

// UpdatePeers updates peer list of the sql-chain.
func (c *Chain) UpdatePeers(peers *kayak.Peers) error {
	return c.rt.updatePeers(peers)
}

// getBilling returns a billing request from the blocks within height range [low, high].
func (c *Chain) getBilling(low, high int32) (req *pt.BillingRequest, err error) {
	// Height `n` is ensured (or skipped) if `Next Turn` > `n` + 1
	if c.rt.getNextTurn() <= high+1 {
		err = ErrUnavailableBillingRang
		return
	}

	var (
		n                   *blockNode
		addr                proto.AccountAddress
		ack                 *wt.SignedAckHeader
		lowBlock, highBlock *ct.Block
		billings            = make(map[proto.AccountAddress]*proto.AddrAndGas)
	)

	if head := c.rt.getHead(); head != nil {
		n = head.node
	}

	for ; n != nil && n.height > high; n = n.parent {
	}

	for ; n != nil && n.height >= low; n = n.parent {
		if lowBlock == nil {
			lowBlock = n.block
		}

		highBlock = n.block

		if addr, err = utils.PubKeyHash(n.block.Signee()); err != nil {
			return
		}

		if billing, ok := billings[addr]; ok {
			billing.GasAmount = c.rt.producingReward
		} else {
			producer := n.block.Producer()
			billings[addr] = &proto.AddrAndGas{
				AccountAddress: addr,
				RawNodeID:      *producer.ToRawNodeID(),
				GasAmount:      c.rt.producingReward,
			}
		}

		for _, v := range n.block.Queries {
			if ack, err = c.queryOrSyncAckedQuery(n.height, v, n.block.Producer()); err != nil {
				return
			}

			if addr, err = utils.PubKeyHash(ack.SignedResponseHeader().Signee); err != nil {
				return
			}

			if billing, ok := billings[addr]; ok {
				billing.GasAmount += c.rt.price[ack.SignedRequestHeader().QueryType] *
					ack.SignedRequestHeader().BatchCount
			} else {
				billings[addr] = &proto.AddrAndGas{
					AccountAddress: addr,
					RawNodeID:      *ack.SignedResponseHeader().NodeID.ToRawNodeID(),
					GasAmount:      c.rt.producingReward,
				}
			}
		}
	}

	if lowBlock == nil || highBlock == nil {
		err = ErrUnavailableBillingRang
		return
	}

	// Make request
	gasAmounts := make([]*proto.AddrAndGas, 0, len(billings))

	for _, v := range billings {
		gasAmounts = append(gasAmounts, v)
	}

	req = &pt.BillingRequest{
		Header: pt.BillingRequestHeader{
			DatabaseID: c.rt.databaseID,
			LowBlock:   *lowBlock.BlockHash(),
			LowHeight:  low,
			HighBlock:  *highBlock.BlockHash(),
			HighHeight: high,
			GasAmounts: gasAmounts,
		},
	}
	return
}

func (c *Chain) collectBillingSignatures(billings *pt.BillingRequest) {
	defer c.rt.wg.Done()
	// Process sign billing responses, note that range iterating over channel will only break if
	// the channle is closed
	req := &MuxSignBillingReq{
		Envelope: proto.Envelope{
			// TODO(leventeliu): Add fields.
		},
		DatabaseID: c.rt.databaseID,
		SignBillingReq: SignBillingReq{
			BillingRequest: *billings,
		},
	}
	proWG := &sync.WaitGroup{}
	respC := make(chan *SignBillingResp)
	defer proWG.Wait()
	proWG.Add(1)
	go func() {
		defer proWG.Done()
		for resp := range respC {
			req.Signees = append(req.Signees, resp.Signee)
			req.Signatures = append(req.Signatures, resp.Signature)
		}

		var (
			bp  proto.NodeID
			err error
		)

		defer log.WithFields(log.Fields{
			"peer":             c.rt.getPeerInfoString(),
			"time":             c.rt.getChainTimeString(),
			"signees_count":    len(req.Signees),
			"signatures_count": len(req.Signatures),
			"bp":               bp,
		}).WithError(err).Debug(
			"Sent billing request")

		if bp, err = rpc.GetCurrentBP(); err != nil {
			return
		}

		resp := &pt.BillingResponse{}

		if err = c.cl.CallNode(bp, "MCC.AdviseBillingRequest", req, resp); err != nil {
			return
		}
	}()

	// Send requests and push responses to response channel
	peers := c.rt.getPeers()
	rpcWG := &sync.WaitGroup{}
	defer func() {
		rpcWG.Wait()
		close(respC)
	}()

	for _, s := range peers.Servers {
		if s.ID != c.rt.getServer().ID {
			rpcWG.Add(1)
			go func(id proto.NodeID) {
				defer rpcWG.Done()
				resp := &MuxSignBillingResp{}

				if err := c.cl.CallNode(id, "SQLC.SignBilling", req, resp); err != nil {
					log.WithFields(log.Fields{
						"peer": c.rt.getPeerInfoString(),
						"time": c.rt.getChainTimeString(),
					}).WithError(err).Error(
						"Failed to send sign billing request")
				}

				respC <- &resp.SignBillingResp
			}(s.ID)
		}
	}
}

// LaunchBilling launches a new billing process for the blocks within height range [low, high]
// (inclusive).
func (c *Chain) LaunchBilling(low, high int32) (err error) {
	var (
		req *pt.BillingRequest
		h   *hash.Hash
	)

	defer log.WithFields(log.Fields{
		"peer": c.rt.getPeerInfoString(),
		"time": c.rt.getChainTimeString(),
		"low":  low,
		"high": high,
	}).WithError(err).Debug("Launched billing process")

	if req, err = c.getBilling(low, high); err != nil {
		return
	}

	if h, err = req.PackRequestHeader(); err != nil {
		return
	}

	req.RequestHash = *h
	c.rt.wg.Add(1)
	go c.collectBillingSignatures(req)
	return
}

// SignBilling signs a billing request.
func (c *Chain) SignBilling(req *pt.BillingRequest) (
	pub *asymmetric.PublicKey, sig *asymmetric.Signature, err error,
) {
	var (
		h   *hash.Hash
		loc *pt.BillingRequest
	)
	defer log.WithFields(log.Fields{
		"peer": c.rt.getPeerInfoString(),
		"time": c.rt.getChainTimeString(),
		"low":  req.Header.LowHeight,
		"high": req.Header.HighHeight,
	}).WithError(err).Debug("Processing sign billing request")

	// Verify billing results
	if h, err = req.PackRequestHeader(); err != nil {
		return
	}

	if !req.RequestHash.IsEqual(h) {
		err = ErrBillingNotMatch
		return
	}

	if loc, err = c.getBilling(req.Header.LowHeight, req.Header.HighHeight); err != nil {
		return
	}

	if !req.Header.LowBlock.IsEqual(&loc.Header.LowBlock) ||
		!req.Header.HighBlock.IsEqual(&loc.Header.HighBlock) {
		err = ErrBillingNotMatch
		return
	}

	reqMap := make(map[proto.AccountAddress]*proto.AddrAndGas)
	locMap := make(map[proto.AccountAddress]*proto.AddrAndGas)

	for _, v := range req.Header.GasAmounts {
		reqMap[v.AccountAddress] = v
	}

	for _, v := range loc.Header.GasAmounts {
		locMap[v.AccountAddress] = v
	}

	if !reflect.DeepEqual(reqMap, locMap) {
		err = ErrBillingNotMatch
		return
	}

	// Sign block with private key
	priv, err := kms.GetLocalPrivateKey()

	if err != nil {
		return
	}

	if sig, err = req.SignRequestHeader(priv); err != nil {
		return
	}

	pub = priv.PubKey()
	return
}

func (c *Chain) addSubscription(nodeID proto.NodeID, startHeight int32) (err error) {
	// send previous height and transactions using AdviseAckedQuery/AdviseNewBlock RPC method
	// add node to subscriber list
	c.observerLock.Lock()
	defer c.observerLock.Unlock()
	c.observers[nodeID] = startHeight
	c.startStopReplication()
	return
}

func (c *Chain) cancelSubscription(nodeID proto.NodeID) (err error) {
	// remove node from subscription list
	c.observerLock.Lock()
	defer c.observerLock.Unlock()
	delete(c.observers, nodeID)
	c.startStopReplication()
	return
}

func (c *Chain) startStopReplication() {
	if c.replCh != nil {
		select {
		case c.replCh <- struct{}{}:
		case <-c.stopCh:
		default:
		}
	}
}

func (c *Chain) populateObservers() {
	c.observerLock.Lock()
	defer c.observerLock.Unlock()

	// handle replication threads
	for nodeID, startHeight := range c.observers {
		if replicator, exists := c.observerReplicators[nodeID]; exists {
			// already started
			if startHeight >= 0 {
				replicator.setNewHeight(startHeight)
				c.observers[nodeID] = int32(-1)
			}
		} else {
			// start new replication routine
			c.replWg.Add(1)
			replicator := newObserverReplicator(nodeID, startHeight, c)
			c.observerReplicators[nodeID] = replicator
			go replicator.run()
		}
	}

	// stop replicators
	for nodeID, replicator := range c.observerReplicators {
		if _, exists := c.observers[nodeID]; !exists {
			replicator.stop()
			delete(c.observerReplicators, nodeID)
		}
	}
}

func (c *Chain) replicationCycle() {
	defer func() {
		c.replWg.Wait()
		c.rt.wg.Done()
	}()

	for {
		select {
		case <-c.replCh:
			// populateObservers
			c.populateObservers()
			// send triggers to replicators
			for _, replicator := range c.observerReplicators {
				replicator.tick()
			}
		case <-c.stopCh:
			return
		}
	}
}
