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
	rt "runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/CovenantSQL/CovenantSQL/crypto"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	x "github.com/CovenantSQL/CovenantSQL/xenomint"
	xi "github.com/CovenantSQL/CovenantSQL/xenomint/interfaces"
	xs "github.com/CovenantSQL/CovenantSQL/xenomint/sqlite"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

const (
	minBlockCacheTTL = int32(30)
)

var (
	metaState         = [4]byte{'S', 'T', 'A', 'T'}
	metaBlockIndex    = [4]byte{'B', 'L', 'C', 'K'}
	metaRequestIndex  = [4]byte{'R', 'E', 'Q', 'U'}
	metaResponseIndex = [4]byte{'R', 'E', 'S', 'P'}
	metaAckIndex      = [4]byte{'Q', 'A', 'C', 'K'}
	leveldbConf       = opt.Options{}

	// Atomic counters for stats
	cachedBlockCount int32
)

func init() {
	leveldbConf.BlockSize = 4 * 1024 * 1024
	leveldbConf.Compression = opt.SnappyCompression
}

func statBlock(b *types.Block) {
	atomic.AddInt32(&cachedBlockCount, 1)
	rt.SetFinalizer(b, func(_ *types.Block) {
		atomic.AddInt32(&cachedBlockCount, -1)
	})
}

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

// keyWithSymbolToHeight converts a height back from a key(ack/resp/req/block) in bytes.
// ack key:
// ['Q', 'A', 'C', 'K', height, hash]
// resp key:
// ['R', 'E', 'S', 'P', height, hash]
// req key:
// ['R', 'E', 'Q', 'U', height, hash]
// block key:
// ['B', 'L', 'C', 'K', height, hash]
func keyWithSymbolToHeight(k []byte) int32 {
	if len(k) < 8 {
		return -1
	}
	return int32(binary.BigEndian.Uint32(k[4:]))
}

// Chain represents a sql-chain.
type Chain struct {
	// bdb stores state and block
	bdb *leveldb.DB
	// tdb stores ack/request/response
	tdb *leveldb.DB
	bi  *blockIndex
	ai  *ackIndex
	st  *x.State
	cl  *rpc.Caller
	rt  *runtime

	stopCh    chan struct{}
	blocks    chan *types.Block
	heights   chan int32
	responses chan *types.ResponseHeader
	acks      chan *types.AckHeader

	// DBAccount info
	tokenType    types.TokenType
	gasPrice     uint64
	updatePeriod uint64

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

	// Cached fileds, may need to renew some of this fields later.
	//
	// pk is the private key of the local miner.
	pk *asymmetric.PrivateKey
}

// NewChain creates a new sql-chain struct.
func NewChain(c *Config) (chain *Chain, err error) {
	// TODO(leventeliu): this is a rough solution, you may also want to clean database file and
	// force rebuilding.
	var fi os.FileInfo
	if fi, err = os.Stat(c.ChainFilePrefix + "-block-state.ldb"); err == nil && fi.Mode().IsDir() {
		return LoadChain(c)
	}

	err = c.Genesis.VerifyAsGenesis()
	if err != nil {
		return
	}

	// Open LevelDB for block and state
	bdbFile := c.ChainFilePrefix + "-block-state.ldb"
	bdb, err := leveldb.OpenFile(bdbFile, &leveldbConf)
	if err != nil {
		err = errors.Wrapf(err, "open leveldb %s", bdbFile)
		return
	}

	log.Debugf("Create new chain bdb %s", bdbFile)

	// Open LevelDB for ack/request/response
	tdbFile := c.ChainFilePrefix + "-ack-req-resp.ldb"
	tdb, err := leveldb.OpenFile(tdbFile, &leveldbConf)
	if err != nil {
		err = errors.Wrapf(err, "open leveldb %s", tdbFile)
		return
	}

	log.Debugf("Create new chain tdb %s", tdbFile)

	// Open x.State
	var (
		strg  xi.Storage
		state *x.State
	)
	if strg, err = xs.NewSqlite(c.DataFile); err != nil {
		return
	}
	if state, err = x.NewState(c.Server, strg); err != nil {
		return
	}

	// Cache local private key
	var pk *asymmetric.PrivateKey
	if pk, err = kms.GetLocalPrivateKey(); err != nil {
		err = errors.Wrap(err, "failed to cache private key")
		return
	}

	// Create chain state
	chain = &Chain{
		bdb:          bdb,
		tdb:          tdb,
		bi:           newBlockIndex(),
		ai:           newAckIndex(),
		st:           state,
		cl:           rpc.NewCaller(),
		rt:           newRunTime(c),
		stopCh:       make(chan struct{}),
		blocks:       make(chan *types.Block),
		heights:      make(chan int32, 1),
		responses:    make(chan *types.ResponseHeader),
		acks:         make(chan *types.AckHeader),
		tokenType:    c.TokenType,
		gasPrice:     c.GasPrice,
		updatePeriod: c.UpdatePeriod,

		// Observer related
		observers:           make(map[proto.NodeID]int32),
		observerReplicators: make(map[proto.NodeID]*observerReplicator),
		replCh:              make(chan struct{}),

		pk: pk,
	}

	if err = chain.pushBlock(c.Genesis); err != nil {
		return nil, err
	}

	return
}

// LoadChain loads the chain state from the specified database and rebuilds a memory index.
func LoadChain(c *Config) (chain *Chain, err error) {
	// Open LevelDB for block and state
	bdbFile := c.ChainFilePrefix + "-block-state.ldb"
	bdb, err := leveldb.OpenFile(bdbFile, &leveldbConf)
	if err != nil {
		err = errors.Wrapf(err, "open leveldb %s", bdbFile)
		return
	}

	// Open LevelDB for ack/request/response
	tdbFile := c.ChainFilePrefix + "-ack-req-resp.ldb"
	tdb, err := leveldb.OpenFile(tdbFile, &leveldbConf)
	if err != nil {
		err = errors.Wrapf(err, "open leveldb %s", tdbFile)
		return
	}

	// Open x.State
	var (
		strg   xi.Storage
		xstate *x.State
	)
	if strg, err = xs.NewSqlite(c.DataFile); err != nil {
		return
	}
	if xstate, err = x.NewState(c.Server, strg); err != nil {
		return
	}

	// Cache local private key
	var pk *asymmetric.PrivateKey
	if pk, err = kms.GetLocalPrivateKey(); err != nil {
		err = errors.Wrap(err, "failed to cache private key")
		return
	}

	// Create chain state
	chain = &Chain{
		bdb:          bdb,
		tdb:          tdb,
		bi:           newBlockIndex(),
		ai:           newAckIndex(),
		st:           xstate,
		cl:           rpc.NewCaller(),
		rt:           newRunTime(c),
		stopCh:       make(chan struct{}),
		blocks:       make(chan *types.Block),
		heights:      make(chan int32, 1),
		responses:    make(chan *types.ResponseHeader),
		acks:         make(chan *types.AckHeader),
		tokenType:    c.TokenType,
		gasPrice:     c.GasPrice,
		updatePeriod: c.UpdatePeriod,

		// Observer related
		observers:           make(map[proto.NodeID]int32),
		observerReplicators: make(map[proto.NodeID]*observerReplicator),
		replCh:              make(chan struct{}),

		pk: pk,
	}

	// Read state struct
	stateEnc, err := chain.bdb.Get(metaState[:], nil)
	if err != nil {
		return nil, err
	}
	st := &state{}
	if err = utils.DecodeMsgPack(stateEnc, st); err != nil {
		return nil, err
	}

	log.WithFields(log.Fields{
		"peer":  chain.rt.getPeerInfoString(),
		"state": st,
	}).Debug("Loading state from database")

	// Read blocks and rebuild memory index
	var (
		id        uint64
		index     int32
		last      *blockNode
		blockIter = chain.bdb.NewIterator(util.BytesPrefix(metaBlockIndex[:]), nil)
	)
	defer blockIter.Release()
	for index = 0; blockIter.Next(); index++ {
		var (
			k     = blockIter.Key()
			v     = blockIter.Value()
			block = &types.Block{}

			current, parent *blockNode
		)

		if err = utils.DecodeMsgPack(v, block); err != nil {
			err = errors.Wrapf(err, "decoding failed at height %d with key %s",
				keyWithSymbolToHeight(k), string(k))
			return
		}
		log.WithFields(log.Fields{
			"peer":  chain.rt.getPeerInfoString(),
			"block": block.BlockHash().String(),
		}).Debug("Loading block from database")

		if last == nil {
			if err = block.VerifyAsGenesis(); err != nil {
				err = errors.Wrap(err, "genesis verification failed")
				return
			}
			// Set constant fields from genesis block
			chain.rt.setGenesis(block)
		} else if block.ParentHash().IsEqual(&last.hash) {
			if err = block.Verify(); err != nil {
				err = errors.Wrapf(err, "block verification failed at height %d with key %s",
					keyWithSymbolToHeight(k), string(k))
				return
			}
			parent = last
		} else {
			if parent = chain.bi.lookupNode(block.ParentHash()); parent == nil {
				return nil, ErrParentNotFound
			}
		}

		// Update id
		if nid, ok := block.CalcNextID(); ok && nid > id {
			id = nid
		}

		current = &blockNode{}
		current.initBlockNode(chain.rt.getHeightFromTime(block.Timestamp()), block, parent)
		chain.bi.addBlock(current)
		last = current
	}
	if err = blockIter.Error(); err != nil {
		err = errors.Wrap(err, "load block")
		return
	}

	// Set chain state
	st.node = last
	chain.rt.setHead(st)
	chain.st.InitTx(id)
	chain.pruneBlockCache()

	// Read queries and rebuild memory index
	respIter := chain.tdb.NewIterator(util.BytesPrefix(metaResponseIndex[:]), nil)
	defer respIter.Release()
	for respIter.Next() {
		k := respIter.Key()
		v := respIter.Value()
		h := keyWithSymbolToHeight(k)
		var resp = &types.SignedResponseHeader{}
		if err = utils.DecodeMsgPack(v, resp); err != nil {
			err = errors.Wrapf(err, "load resp, height %d, index %s", h, string(k))
			return
		}
		log.WithFields(log.Fields{
			"height": h,
			"header": resp.Hash().String(),
		}).Debug("Loaded new resp header")
	}
	if err = respIter.Error(); err != nil {
		err = errors.Wrap(err, "load resp")
		return
	}

	ackIter := chain.tdb.NewIterator(util.BytesPrefix(metaAckIndex[:]), nil)
	defer ackIter.Release()
	for ackIter.Next() {
		k := ackIter.Key()
		v := ackIter.Value()
		h := keyWithSymbolToHeight(k)
		var ack = &types.SignedAckHeader{}
		if err = utils.DecodeMsgPack(v, ack); err != nil {
			err = errors.Wrapf(err, "load ack, height %d, index %s", h, string(k))
			return
		}
		log.WithFields(log.Fields{
			"height": h,
			"header": ack.Hash().String(),
		}).Debug("Loaded new ack header")
	}
	if err = respIter.Error(); err != nil {
		err = errors.Wrap(err, "load ack")
		return
	}

	return
}

// pushBlock pushes the signed block header to extend the current main chain.
func (c *Chain) pushBlock(b *types.Block) (err error) {
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
	t, err := c.bdb.OpenTransaction()
	if err = t.Put(metaState[:], encState.Bytes(), nil); err != nil {
		err = errors.Wrapf(err, "put %s", string(metaState[:]))
		t.Discard()
		return
	}
	blockKey := utils.ConcatAll(metaBlockIndex[:], node.indexKey())
	if err = t.Put(blockKey, encBlock.Bytes(), nil); err != nil {
		err = errors.Wrapf(err, "put %s", string(node.indexKey()))
		t.Discard()
		return
	}
	if err = t.Commit(); err != nil {
		err = errors.Wrapf(err, "commit error")
		t.Discard()
		return
	}
	c.rt.setHead(st)
	c.bi.addBlock(node)

	// Keep track of the queries from the new block
	var ierr error
	for i, v := range b.QueryTxs {
		if ierr = c.addResponse(v.Response); ierr != nil {
			log.WithFields(log.Fields{
				"index":      i,
				"producer":   b.Producer(),
				"block_hash": b.BlockHash(),
			}).WithError(ierr).Warn("Failed to add response to ackIndex")
		}
	}
	for i, v := range b.Acks {
		if ierr = c.remove(v); ierr != nil {
			log.WithFields(log.Fields{
				"index":      i,
				"producer":   b.Producer(),
				"block_hash": b.BlockHash(),
			}).WithError(ierr).Warn("Failed to remove Ack from ackIndex")
		}
	}

	if err == nil {
		log.WithFields(log.Fields{
			"peer":       c.rt.getPeerInfoString()[:14],
			"time":       c.rt.getChainTimeString(),
			"block":      b.BlockHash().String()[:8],
			"producer":   b.Producer()[:8],
			"queryCount": len(b.QueryTxs),
			"ackCount":   len(b.Acks),
			"blockTime":  b.Timestamp().Format(time.RFC3339Nano),
			"height":     c.rt.getHeightFromTime(b.Timestamp()),
			"head": fmt.Sprintf("%s <- %s",
				func() string {
					if st.node.parent != nil {
						return st.node.parent.hash.String()[:8]
					}
					return "|"
				}(), st.Head.String()[:8]),
			"headHeight": c.rt.getHead().Height,
		}).Info("Pushed new block")
	}

	return
}

// pushAckedQuery pushes a acknowledged, signed and verified query into the chain.
func (c *Chain) pushAckedQuery(ack *types.SignedAckHeader) (err error) {
	log.Debugf("push ack %s", ack.Hash().String())
	h := c.rt.getHeightFromTime(ack.SignedResponseHeader().Timestamp)
	k := heightToKey(h)
	var enc *bytes.Buffer

	if enc, err = utils.EncodeMsgPack(ack); err != nil {
		return
	}

	tdbKey := utils.ConcatAll(metaAckIndex[:], k, ack.Hash().AsBytes())

	if err = c.tdb.Put(tdbKey, enc.Bytes(), nil); err != nil {
		err = errors.Wrapf(err, "put ack %d %s", h, ack.Hash().String())
		return
	}

	if err = c.register(ack); err != nil {
		err = errors.Wrapf(err, "register ack %v at height %d", ack.Hash(), h)
		return
	}

	return
}

// produceBlockV2 prepares, signs and advises the pending block to the other peers.
func (c *Chain) produceBlockV2(now time.Time) (err error) {
	var (
		frs []*types.Request
		qts []*x.QueryTracker
	)
	if frs, qts, err = c.st.CommitEx(); err != nil {
		return
	}
	var block = &types.Block{
		SignedHeader: types.SignedHeader{
			Header: types.Header{
				Version:     0x01000000,
				Producer:    c.rt.getServer(),
				GenesisHash: c.rt.genesisHash,
				ParentHash:  c.rt.getHead().Head,
				// MerkleRoot: will be set by BPBlock.PackAndSignBlock(PrivateKey)
				Timestamp: now,
			},
		},
		FailedReqs: frs,
		QueryTxs:   make([]*types.QueryAsTx, len(qts)),
		Acks:       c.ai.acks(c.rt.getHeightFromTime(now)),
	}
	statBlock(block)
	for i, v := range qts {
		// TODO(leventeliu): maybe block waiting at a ready channel instead?
		for !v.Ready() {
			time.Sleep(1 * time.Millisecond)
		}
		block.QueryTxs[i] = &types.QueryAsTx{
			// TODO(leventeliu): add acks for billing.
			Request:  v.Req,
			Response: &v.Resp.Header,
		}
	}
	// Sign block
	if err = block.PackAndSignBlock(c.pk); err != nil {
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
	var (
		req = &MuxAdviseNewBlockReq{
			Envelope: proto.Envelope{
				// TODO(leventeliu): Add fields.
			},
			DatabaseID: c.rt.databaseID,
			AdviseNewBlockReq: AdviseNewBlockReq{
				Block: block,
				Count: func() int32 {
					if nd := c.bi.lookupNode(block.BlockHash()); nd != nil {
						return nd.count
					}
					if pn := c.bi.lookupNode(block.ParentHash()); pn != nil {
						return pn.count + 1
					}
					return -1
				}(),
			},
		}
		peers = c.rt.getPeers()
		wg    = &sync.WaitGroup{}
	)
	for _, s := range peers.Servers {
		if s != c.rt.getServer() {
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
			}(s)
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
			if s != c.rt.getServer() {
				if err = c.cl.CallNode(
					s, route.SQLCFetchBlock.String(), req, resp,
				); err != nil || resp.Block == nil {
					log.WithFields(log.Fields{
						"peer":        c.rt.getPeerInfoString(),
						"time":        c.rt.getChainTimeString(),
						"remote":      fmt.Sprintf("[%d/%d] %s", i, len(peers.Servers), s),
						"curr_turn":   c.rt.getNextTurn(),
						"head_height": c.rt.getHead().Height,
						"head_block":  c.rt.getHead().Head.String(),
					}).WithError(err).Debug(
						"Failed to fetch block from peer")
				} else {
					statBlock(resp.Block)
					c.blocks <- resp.Block
					log.WithFields(log.Fields{
						"peer":        c.rt.getPeerInfoString(),
						"time":        c.rt.getChainTimeString(),
						"remote":      fmt.Sprintf("[%d/%d] %s", i, len(peers.Servers), s),
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
		c.stat()
		c.pruneBlockCache()
		c.rt.setNextTurn()
		c.ai.advance(c.rt.getMinValidHeight())
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

	if err := c.produceBlockV2(now); err != nil {
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
				//log.WithFields(log.Fields{
				//	"peer":            c.rt.getPeerInfoString(),
				//	"time":            c.rt.getChainTimeString(),
				//	"next_turn":       c.rt.getNextTurn(),
				//	"head_height":     c.rt.getHead().Height,
				//	"head_block":      c.rt.getHead().Head.String(),
				//	"using_timestamp": t.Format(time.RFC3339Nano),
				//	"duration":        d,
				//}).Debug("Main cycle")
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
	returnStash := func(stash []*types.Block) {
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

	var stash []*types.Block
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
						}).WithError(err).Error("Failed to check and push new block")
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
	// Close LevelDB file
	var ierr error
	if ierr = c.bdb.Close(); ierr != nil && err == nil {
		err = ierr
	}
	log.WithFields(log.Fields{
		"peer": c.rt.getPeerInfoString(),
		"time": c.rt.getChainTimeString(),
	}).WithError(ierr).Debug("Chain database closed")
	if ierr = c.tdb.Close(); ierr != nil && err == nil {
		err = ierr
	}
	log.WithFields(log.Fields{
		"peer": c.rt.getPeerInfoString(),
		"time": c.rt.getChainTimeString(),
	}).WithError(ierr).Debug("Chain database closed")
	// Close state
	if ierr = c.st.Close(false); ierr != nil && err == nil {
		err = ierr
	}
	log.WithFields(log.Fields{
		"peer": c.rt.getPeerInfoString(),
		"time": c.rt.getChainTimeString(),
	}).WithError(ierr).Debug("Chain state storage closed")
	return
}

// FetchBlock fetches the block at specified height from local cache.
func (c *Chain) FetchBlock(height int32) (b *types.Block, err error) {
	if n := c.rt.getHead().node.ancestor(height); n != nil {
		k := utils.ConcatAll(metaBlockIndex[:], n.indexKey())
		var v []byte
		v, err = c.bdb.Get(k, nil)
		if err != nil {
			err = errors.Wrapf(err, "fetch block %s", string(k))
			return
		}

		b = &types.Block{}
		statBlock(b)
		err = utils.DecodeMsgPack(v, b)
		if err != nil {
			err = errors.Wrapf(err, "fetch block %s", string(k))
			return
		}
	}

	return
}

// CheckAndPushNewBlock implements ChainRPCServer.CheckAndPushNewBlock.
func (c *Chain) CheckAndPushNewBlock(block *types.Block) (err error) {
	height := c.rt.getHeightFromTime(block.Timestamp())
	head := c.rt.getHead()
	peers := c.rt.getPeers()
	total := int32(len(peers.Servers))
	next := func() int32 {
		if total > 0 {
			return (c.rt.getNextTurn() - 1) % total
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
	if block.Producer() == c.rt.server {
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

	// Replicate local state from the new block
	if err = c.st.ReplayBlock(block); err != nil {
		return
	}

	return c.pushBlock(block)
}

// VerifyAndPushAckedQuery verifies a acknowledged and signed query, and pushed it if valid.
func (c *Chain) VerifyAndPushAckedQuery(ack *types.SignedAckHeader) (err error) {
	// TODO(leventeliu): check ack.
	if c.rt.queryTimeIsExpired(ack.SignedResponseHeader().Timestamp) {
		err = errors.Wrapf(ErrQueryExpired, "Verify ack query, min valid height %d, ack height %d", c.rt.getMinValidHeight(), c.rt.getHeightFromTime(ack.Timestamp))
		return
	}

	if err = ack.Verify(); err != nil {
		return
	}

	return c.pushAckedQuery(ack)
}

// UpdatePeers updates peer list of the sql-chain.
func (c *Chain) UpdatePeers(peers *proto.Peers) error {
	return c.rt.updatePeers(peers)
}

// getBilling returns a billing request from the blocks within height range [low, high].
func (c *Chain) getBilling(low, high int32) (req *types.BillingRequest, err error) {
	// Height `n` is ensured (or skipped) if `Next Turn` > `n` + 1
	if c.rt.getNextTurn() <= high+1 {
		err = ErrUnavailableBillingRang
		return
	}

	var (
		n                   *blockNode
		addr                proto.AccountAddress
		lowBlock, highBlock *types.Block
		billings            = make(map[proto.AccountAddress]*proto.AddrAndGas)
	)

	if head := c.rt.getHead(); head != nil {
		n = head.node
	}

	for ; n != nil && n.height > high; n = n.parent {
	}

	for ; n != nil && n.height >= low; n = n.parent {
		// TODO(leventeliu): block maybe released, use persistence version in this case.
		if n.block == nil {
			continue
		}

		if lowBlock == nil {
			lowBlock = n.block
		}

		highBlock = n.block

		if addr, err = crypto.PubKeyHash(n.block.Signee()); err != nil {
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

		for _, v := range n.block.Acks {
			if addr, err = crypto.PubKeyHash(v.SignedResponseHeader().Signee); err != nil {
				return
			}

			if billing, ok := billings[addr]; ok {
				billing.GasAmount += c.rt.price[v.SignedRequestHeader().QueryType] *
					v.SignedRequestHeader().BatchCount
			} else {
				billings[addr] = &proto.AddrAndGas{
					AccountAddress: addr,
					RawNodeID:      *v.SignedResponseHeader().NodeID.ToRawNodeID(),
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

	req = &types.BillingRequest{
		Header: types.BillingRequestHeader{
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

func (c *Chain) collectBillingSignatures(billings *types.BillingRequest) {
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

		bpReq := &types.AdviseBillingReq{
			Req: billings,
		}

		var (
			bpNodeID proto.NodeID
			err      error
		)

		for resp := range respC {
			if err = bpReq.Req.AddSignature(resp.Signee, resp.Signature, false); err != nil {
				// consume all rpc result
				for range respC {
				}

				return
			}
		}

		defer log.WithFields(log.Fields{
			"peer":             c.rt.getPeerInfoString(),
			"time":             c.rt.getChainTimeString(),
			"signees_count":    len(req.Signees),
			"signatures_count": len(req.Signatures),
			"bp":               bpNodeID,
		}).WithError(err).Debug(
			"Sent billing request")

		if bpNodeID, err = rpc.GetCurrentBP(); err != nil {
			return
		}

		var resp interface{}
		if err = c.cl.CallNode(bpNodeID, route.MCCAdviseBillingRequest.String(), bpReq, resp); err != nil {
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
		if s != c.rt.getServer() {
			rpcWG.Add(1)
			go func(id proto.NodeID) {
				defer rpcWG.Done()
				resp := &MuxSignBillingResp{}

				if err := c.cl.CallNode(id, route.SQLCSignBilling.String(), req, resp); err != nil {
					log.WithFields(log.Fields{
						"peer": c.rt.getPeerInfoString(),
						"time": c.rt.getChainTimeString(),
					}).WithError(err).Error(
						"Failed to send sign billing request")
				}

				respC <- &resp.SignBillingResp
			}(s)
		}
	}
}

// LaunchBilling launches a new billing process for the blocks within height range [low, high]
// (inclusive).
func (c *Chain) LaunchBilling(low, high int32) (err error) {
	var (
		req *types.BillingRequest
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

	if _, err = req.PackRequestHeader(); err != nil {
		return
	}

	c.rt.wg.Add(1)
	go c.collectBillingSignatures(req)
	return
}

// SignBilling signs a billing request.
func (c *Chain) SignBilling(req *types.BillingRequest) (
	pub *asymmetric.PublicKey, sig *asymmetric.Signature, err error,
) {
	var (
		loc *types.BillingRequest
	)
	defer log.WithFields(log.Fields{
		"peer": c.rt.getPeerInfoString(),
		"time": c.rt.getChainTimeString(),
		"low":  req.Header.LowHeight,
		"high": req.Header.HighHeight,
	}).WithError(err).Debug("Processing sign billing request")

	// Verify billing results
	if err = req.VerifySignatures(); err != nil {
		return
	}
	if loc, err = c.getBilling(req.Header.LowHeight, req.Header.HighHeight); err != nil {
		return
	}
	if err = req.Compare(loc); err != nil {
		return
	}
	pub, sig, err = req.SignRequestHeader(c.pk, false)
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

// Query queries req from local chain state and returns the query results in resp.
func (c *Chain) Query(req *types.Request) (resp *types.Response, err error) {
	var ref *x.QueryTracker
	if ref, resp, err = c.st.Query(req); err != nil {
		return
	}
	if err = resp.Sign(c.pk); err != nil {
		return
	}
	if err = c.addResponse(&resp.Header); err != nil {
		return
	}
	ref.UpdateResp(resp)
	return
}

// Replay replays a write log from other peer to replicate storage state.
func (c *Chain) Replay(req *types.Request, resp *types.Response) (err error) {
	switch req.Header.QueryType {
	case types.ReadQuery:
		return
	case types.WriteQuery:
		return c.st.Replay(req, resp)
	default:
		err = ErrInvalidRequest
	}
	if err = c.addResponse(&resp.Header); err != nil {
		return
	}
	return
}

func (c *Chain) addResponse(resp *types.SignedResponseHeader) (err error) {
	return c.ai.addResponse(c.rt.getHeightFromTime(resp.Request.Timestamp), resp)
}

func (c *Chain) register(ack *types.SignedAckHeader) (err error) {
	return c.ai.register(c.rt.getHeightFromTime(ack.SignedRequestHeader().Timestamp), ack)
}

func (c *Chain) remove(ack *types.SignedAckHeader) (err error) {
	return c.ai.remove(c.rt.getHeightFromTime(ack.SignedRequestHeader().Timestamp), ack)
}

func (c *Chain) pruneBlockCache() {
	var (
		head    = c.rt.getHead().node
		lastCnt int32
	)
	if head == nil {
		return
	}
	lastCnt = head.count - c.rt.blockCacheTTL
	// Move to last count position
	for ; head != nil && head.count > lastCnt; head = head.parent {
	}
	// Prune block references
	for ; head != nil && head.block != nil; head = head.parent {
		head.block = nil
	}
}

func (c *Chain) stat() {
	var (
		ic = atomic.LoadInt32(&multiIndexCount)
		rc = atomic.LoadInt32(&responseCount)
		tc = atomic.LoadInt32(&ackTrackerCount)
		bc = atomic.LoadInt32(&cachedBlockCount)
	)
	// Print chain stats
	log.WithFields(log.Fields{
		"database_id":           c.rt.databaseID,
		"multiIndex_count":      ic,
		"response_header_count": rc,
		"query_tracker_count":   tc,
		"cached_block_count":    bc,
	}).Info("Chain mem stats")
	// Print xeno stats
	c.st.Stat(c.rt.databaseID)
}
