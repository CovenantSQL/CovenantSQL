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
	"context"
	"database/sql"
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
	metaResponseIndex = [4]byte{'R', 'E', 'S', 'P'}
	metaAckIndex      = [4]byte{'Q', 'A', 'C', 'K'}
	leveldbConf       = opt.Options{}

	// Atomic counters for stats
	cachedBlockCount int32
)

func init() {
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
	// bdb stores state, profile and block
	bdb *leveldb.DB
	// tdb stores ack/request/response
	tdb *leveldb.DB
	bi  *blockIndex
	ai  *ackIndex
	st  *x.State
	cl  *rpc.Caller
	rt  *runtime
	ctx context.Context // ctx is the root context of Chain

	blocks    chan *types.Block
	heights   chan int32
	responses chan *types.ResponseHeader
	acks      chan *types.AckHeader

	// DBAccount info
	databaseID   proto.DatabaseID
	tokenType    types.TokenType
	gasPrice     uint64
	updatePeriod uint64

	// Cached fileds, may need to renew some of this fields later.
	//
	// pk is the private key of the local miner.
	pk *asymmetric.PrivateKey
	// addr is the AccountAddress generate from public key.
	addr *proto.AccountAddress
}

// NewChain creates a new sql-chain struct.
func NewChain(c *Config) (chain *Chain, err error) {
	return NewChainWithContext(context.Background(), c)
}

// NewChainWithContext creates a new sql-chain struct with context.
func NewChainWithContext(ctx context.Context, c *Config) (chain *Chain, err error) {
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

	log.WithField("db", c.DatabaseID).Debugf("create new chain bdb %s", bdbFile)

	// Open LevelDB for ack/request/response
	tdbFile := c.ChainFilePrefix + "-ack-req-resp.ldb"
	tdb, err := leveldb.OpenFile(tdbFile, &leveldbConf)
	if err != nil {
		err = errors.Wrapf(err, "open leveldb %s", tdbFile)
		return
	}

	log.WithField("db", c.DatabaseID).Debugf("create new chain tdb %s", tdbFile)

	// Open storage
	var strg xi.Storage
	if strg, err = xs.NewSqlite(c.DataFile); err != nil {
		return
	}

	// Cache local private key
	var (
		pk   *asymmetric.PrivateKey
		addr proto.AccountAddress
	)
	if pk, err = kms.GetLocalPrivateKey(); err != nil {
		err = errors.Wrap(err, "failed to cache private key")
		return
	}
	addr, err = crypto.PubKeyHash(pk.PubKey())
	if err != nil {
		log.WithError(err).WithField("db", c.DatabaseID).Warning("failed to generate addr in NewChain")
		return
	}

	// Create chain state
	chain = &Chain{
		bdb:          bdb,
		tdb:          tdb,
		bi:           newBlockIndex(),
		ai:           newAckIndex(),
		st:           x.NewState(sql.IsolationLevel(c.IsolationLevel), c.Server, strg),
		cl:           rpc.NewCaller(),
		rt:           newRunTime(ctx, c),
		ctx:          ctx,
		blocks:       make(chan *types.Block),
		heights:      make(chan int32, 1),
		responses:    make(chan *types.ResponseHeader),
		acks:         make(chan *types.AckHeader),
		tokenType:    c.TokenType,
		gasPrice:     c.GasPrice,
		updatePeriod: c.UpdatePeriod,
		databaseID:   c.DatabaseID,

		pk:   pk,
		addr: &addr,
	}

	if err = chain.pushBlock(c.Genesis); err != nil {
		return nil, err
	}

	return
}

// LoadChain loads the chain state from the specified database and rebuilds a memory index.
func LoadChain(c *Config) (chain *Chain, err error) {
	return LoadChainWithContext(context.Background(), c)
}

// LoadChainWithContext loads the chain state from the specified database and rebuilds
// a memory index with context.
func LoadChainWithContext(ctx context.Context, c *Config) (chain *Chain, err error) {
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
	var strg xi.Storage
	if strg, err = xs.NewSqlite(c.DataFile); err != nil {
		return
	}

	// Cache local private key
	var (
		pk   *asymmetric.PrivateKey
		addr proto.AccountAddress
	)
	if pk, err = kms.GetLocalPrivateKey(); err != nil {
		err = errors.Wrap(err, "failed to cache private key")
		return
	}
	addr, err = crypto.PubKeyHash(pk.PubKey())
	if err != nil {
		log.WithError(err).WithField("db", c.DatabaseID).Warning("failed to generate addr in LoadChain")
		return
	}

	// Create chain state
	chain = &Chain{
		bdb:          bdb,
		tdb:          tdb,
		bi:           newBlockIndex(),
		ai:           newAckIndex(),
		st:           x.NewState(sql.IsolationLevel(c.IsolationLevel), c.Server, strg),
		cl:           rpc.NewCaller(),
		rt:           newRunTime(ctx, c),
		ctx:          ctx,
		blocks:       make(chan *types.Block),
		heights:      make(chan int32, 1),
		responses:    make(chan *types.ResponseHeader),
		acks:         make(chan *types.AckHeader),
		tokenType:    c.TokenType,
		gasPrice:     c.GasPrice,
		updatePeriod: c.UpdatePeriod,
		databaseID:   c.DatabaseID,

		pk:   pk,
		addr: &addr,
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
		"db":    c.DatabaseID,
	}).Debug("loading state from database")

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
			"db":    c.DatabaseID,
		}).Debug("loading block from database")

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
	chain.st.SetSeq(id)
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
			"db":     c.DatabaseID,
		}).Debug("loaded new resp header")
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
			"db":     c.DatabaseID,
		}).Debug("loaded new ack header")
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
		if ierr = c.AddResponse(v.Response); ierr != nil {
			log.WithFields(log.Fields{
				"index":      i,
				"producer":   b.Producer(),
				"block_hash": b.BlockHash(),
				"db":         c.databaseID,
			}).WithError(ierr).Warn("failed to add response to ackIndex")
		}
	}
	for i, v := range b.Acks {
		if ierr = c.remove(v); ierr != nil {
			log.WithFields(log.Fields{
				"index":      i,
				"producer":   b.Producer(),
				"block_hash": b.BlockHash(),
				"db":         c.databaseID,
			}).WithError(ierr).Warn("failed to remove Ack from ackIndex")
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
			"db":         c.databaseID,
		}).Info("pushed new block")
	}

	return
}

// pushAckedQuery pushes a acknowledged, signed and verified query into the chain.
func (c *Chain) pushAckedQuery(ack *types.SignedAckHeader) (err error) {
	log.WithField("db", c.databaseID).Debugf("push ack %s", ack.Hash().String())
	h := c.rt.getHeightFromTime(ack.GetResponseTimestamp())
	k := heightToKey(h)
	var enc *bytes.Buffer

	if enc, err = utils.EncodeMsgPack(ack); err != nil {
		return
	}

	tdbKey := utils.ConcatAll(metaAckIndex[:], k, ack.Hash().AsBytes())

	if err = c.register(ack); err != nil {
		err = errors.Wrapf(err, "register ack %v at height %d", ack.Hash(), h)
		return
	}

	if err = c.tdb.Put(tdbKey, enc.Bytes(), nil); err != nil {
		err = errors.Wrapf(err, "put ack %d %s", h, ack.Hash().String())
		return
	}

	return
}

// produceBlock prepares, signs and advises the pending block to the other peers.
func (c *Chain) produceBlock(now time.Time) (err error) {
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
			if c.rt.ctx.Err() != nil {
				err = c.rt.ctx.Err()
				return
			}
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
	select {
	case c.blocks <- block:
	case <-c.rt.ctx.Done():
		err = c.rt.ctx.Err()
		return
	}
	log.WithFields(log.Fields{
		"peer":            c.rt.getPeerInfoString(),
		"time":            c.rt.getChainTimeString(),
		"curr_turn":       c.rt.getNextTurn(),
		"using_timestamp": now.Format(time.RFC3339Nano),
		"block_hash":      block.BlockHash().String(),
		"db":              c.databaseID,
	}).Debug("produced new block")
	// Advise new block to the other peers
	var (
		req = &MuxAdviseNewBlockReq{
			Envelope: proto.Envelope{
				// TODO(leventeliu): Add fields.
			},
			DatabaseID: c.databaseID,
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
				if err := c.cl.CallNodeWithContext(
					c.rt.ctx, id, route.SQLCAdviseNewBlock.String(), req, resp,
				); err != nil {
					log.WithFields(log.Fields{
						"peer":            c.rt.getPeerInfoString(),
						"time":            c.rt.getChainTimeString(),
						"curr_turn":       c.rt.getNextTurn(),
						"using_timestamp": now.Format(time.RFC3339Nano),
						"block_hash":      block.BlockHash().String(),
						"db":              c.databaseID,
					}).WithError(err).Error("failed to advise new block")
				}
			}(s)
		}
	}
	wg.Wait()

	return
}

func (c *Chain) syncHead() {
	// Try to fetch if the block of the current turn is not advised yet
	if h := c.rt.getNextTurn() - 1; c.rt.getHead().Height < h {
		var err error
		req := &MuxFetchBlockReq{
			Envelope: proto.Envelope{
				// TODO(leventeliu): Add fields.
			},
			DatabaseID: c.databaseID,
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
						"db":          c.databaseID,
					}).WithError(err).Debug(
						"Failed to fetch block from peer")
				} else {
					statBlock(resp.Block)
					select {
					case c.blocks <- resp.Block:
					case <-c.rt.ctx.Done():
						err = c.rt.ctx.Err()
						return
					}
					log.WithFields(log.Fields{
						"peer":        c.rt.getPeerInfoString(),
						"time":        c.rt.getChainTimeString(),
						"remote":      fmt.Sprintf("[%d/%d] %s", i, len(peers.Servers), s),
						"curr_turn":   c.rt.getNextTurn(),
						"head_height": c.rt.getHead().Height,
						"head_block":  c.rt.getHead().Head.String(),
						"db":          c.databaseID,
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
				"db":          c.databaseID,
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
		"db":              c.databaseID,
	}).Debug("run current turn")

	if c.rt.getHead().Height < c.rt.getNextTurn()-1 {
		log.WithFields(log.Fields{
			"peer":            c.rt.getPeerInfoString(),
			"time":            c.rt.getChainTimeString(),
			"curr_turn":       c.rt.getNextTurn(),
			"head_height":     c.rt.getHead().Height,
			"head_block":      c.rt.getHead().Head.String(),
			"using_timestamp": now.Format(time.RFC3339Nano),
			"db":              c.databaseID,
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
			"db":              c.databaseID,
		}).WithError(err).Error(
			"Failed to produce block")
	}
}

// mainCycle runs main cycle of the sql-chain.
func (c *Chain) mainCycle(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
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
				//	"db":              c.databaseID,
				//}).Debug("main cycle")
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
		"db":   c.databaseID,
	}).Debug("synchronizing chain state")

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

func (c *Chain) processBlocks(ctx context.Context) {
	var (
		cld, ccl = context.WithCancel(ctx)
		wg       = &sync.WaitGroup{}
	)

	returnStash := func(stash []*types.Block) {
		defer wg.Done()
		for _, block := range stash {
			select {
			case c.blocks <- block:
			case <-cld.Done():
				return
			}
		}
	}

	defer func() {
		ccl()
		wg.Wait()
	}()

	var (
		stash []*types.Block
	)
	for {
		select {
		case h := <-c.heights:
			// Return all stashed blocks to pending channel
			log.WithFields(log.Fields{
				"height": h,
				"stashs": len(stash),
				"db":     c.databaseID,
			}).Debug("read new height from channel")
			if stash != nil {
				wg.Add(1)
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
				"db":           c.databaseID,
			}).Debug("processing new block")

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
							"db":           c.databaseID,
						}).WithError(err).Error("Failed to check and push new block")
					} else {
						head := c.rt.getHead()
						currentCount := uint64(head.node.count)
						if currentCount%c.updatePeriod == 0 {
							ub, err := c.billing(head.node)
							if err != nil {
								log.WithError(err).WithField("db", c.databaseID).Error("billing failed")
							}
							// allocate nonce
							nonceReq := &types.NextAccountNonceReq{}
							nonceResp := &types.NextAccountNonceResp{}
							nonceReq.Addr = *c.addr
							if err = rpc.RequestBP(route.MCCNextAccountNonce.String(), nonceReq, nonceResp); err != nil {
								// allocate nonce failed
								log.WithError(err).WithField("db", c.databaseID).Warning("allocate nonce for transaction failed")
							}
							ub.Nonce = nonceResp.Nonce
							if err = ub.Sign(c.pk); err != nil {
								log.WithError(err).WithField("db", c.databaseID).Warning("sign tx failed")
							}

							addTxReq := &types.AddTxReq{TTL: 1}
							addTxResp := &types.AddTxResp{}
							addTxReq.Tx = ub
							log.WithField("db", c.databaseID).Debugf("nonce in processBlocks: %d, addr: %s",
								addTxReq.Tx.GetAccountNonce(), addTxReq.Tx.GetAccountAddress())
							if err = rpc.RequestBP(route.MCCAddTx.String(), addTxReq, addTxResp); err != nil {
								log.WithError(err).WithField("db", c.databaseID).Warning("send tx failed")
							}
						}
					}
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

// Start starts the main process of the sql-chain.
func (c *Chain) Start() (err error) {
	if err = c.sync(); err != nil {
		return
	}

	c.rt.goFunc(c.processBlocks)
	c.rt.goFunc(c.mainCycle)
	c.rt.startService(c)
	return
}

// Stop stops the main process of the sql-chain.
func (c *Chain) Stop() (err error) {
	// Stop main process
	log.WithFields(log.Fields{
		"peer": c.rt.getPeerInfoString(),
		"time": c.rt.getChainTimeString(),
		"db":   c.databaseID,
	}).Debug("stopping chain")
	c.rt.stop(c.databaseID)
	log.WithFields(log.Fields{
		"peer": c.rt.getPeerInfoString(),
		"time": c.rt.getChainTimeString(),
		"db":   c.databaseID,
	}).Debug("chain service and workers stopped")
	// Close LevelDB file
	var ierr error
	if ierr = c.bdb.Close(); ierr != nil && err == nil {
		err = ierr
	}
	log.WithFields(log.Fields{
		"peer": c.rt.getPeerInfoString(),
		"time": c.rt.getChainTimeString(),
		"db":   c.databaseID,
	}).WithError(ierr).Debug("chain database closed")
	if ierr = c.tdb.Close(); ierr != nil && err == nil {
		err = ierr
	}
	log.WithFields(log.Fields{
		"peer": c.rt.getPeerInfoString(),
		"time": c.rt.getChainTimeString(),
		"db":   c.databaseID,
	}).WithError(ierr).Debug("chain database closed")
	// Close state
	if ierr = c.st.Close(false); ierr != nil && err == nil {
		err = ierr
	}
	log.WithFields(log.Fields{
		"peer": c.rt.getPeerInfoString(),
		"time": c.rt.getChainTimeString(),
		"db":   c.databaseID,
	}).WithError(ierr).Debug("chain state storage closed")
	return
}

// FetchBlock fetches the block at specified height from local cache.
func (c *Chain) FetchBlock(height int32) (b *types.Block, err error) {
	if n := c.rt.getHead().node.ancestor(height); n != nil {
		b, err = c.fetchBlockByIndexKey(n.indexKey())
		if err != nil {
			return
		}
	}

	return
}

// FetchBlockByCount fetches the block at specified count from local cache.
func (c *Chain) FetchBlockByCount(count int32) (b *types.Block, realCount int32, height int32, err error) {
	var n *blockNode

	if count < 0 {
		n = c.rt.getHead().node
	} else {
		n = c.rt.getHead().node.ancestorByCount(count)
	}

	if n != nil {
		b, err = c.fetchBlockByIndexKey(n.indexKey())
		if err != nil {
			return
		}

		height = n.height
		realCount = n.count
	}

	return
}

func (c *Chain) fetchBlockByIndexKey(indexKey []byte) (b *types.Block, err error) {
	k := utils.ConcatAll(metaBlockIndex[:], indexKey)
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
		"db":          c.databaseID,
	}).WithError(err).Debug("checking new block from other peer")

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
			"db":       c.databaseID,
		}).WithError(err).Error(
			"Failed to check new block")
		return ErrInvalidProducer
	}

	// TODO(leventeliu): check if too many periods are skipped or store block for future use.
	// if height-c.rt.getHead().Height > X {
	// 	...
	// }

	// Replicate local state from the new block
	if err = c.st.ReplayBlockWithContext(c.rt.ctx, block); err != nil {
		return
	}

	return c.pushBlock(block)
}

// VerifyAndPushAckedQuery verifies a acknowledged and signed query, and pushed it if valid.
func (c *Chain) VerifyAndPushAckedQuery(ack *types.SignedAckHeader) (err error) {
	// TODO(leventeliu): check ack.
	if c.rt.queryTimeIsExpired(ack.GetResponseTimestamp()) {
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

// Query queries req from local chain state and returns the query results in resp.
func (c *Chain) Query(
	req *types.Request, isLeader bool) (tracker *x.QueryTracker, resp *types.Response, err error,
) {
	// TODO(leventeliu): we're using an external context passed by request. Make sure that
	// cancelling will be propagated to this context before chain instance stops.
	return c.st.QueryWithContext(req.GetContext(), req, isLeader)
}

// AddResponse addes a response to the ackIndex, awaiting for acknowledgement.
func (c *Chain) AddResponse(resp *types.SignedResponseHeader) (err error) {
	return c.ai.addResponse(c.rt.getHeightFromTime(resp.GetRequestTimestamp()), resp)
}

func (c *Chain) register(ack *types.SignedAckHeader) (err error) {
	return c.ai.register(c.rt.getHeightFromTime(ack.GetRequestTimestamp()), ack)
}

func (c *Chain) remove(ack *types.SignedAckHeader) (err error) {
	return c.ai.remove(c.rt.getHeightFromTime(ack.GetRequestTimestamp()), ack)
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
		tc = atomic.LoadInt32(&ackCount)
		bc = atomic.LoadInt32(&cachedBlockCount)
	)
	// Print chain stats
	log.WithFields(log.Fields{
		"database_id":           c.databaseID,
		"multiIndex_count":      ic,
		"response_header_count": rc,
		"query_tracker_count":   tc,
		"cached_block_count":    bc,
		"db":                    c.databaseID,
	}).Info("chain mem stats")
	// Print xeno stats
	c.st.Stat(c.databaseID)
}

func (c *Chain) billing(node *blockNode) (ub *types.UpdateBilling, err error) {
	log.WithField("db", c.databaseID).Debugf("begin to billing from count %d", node.count)
	var (
		i, j      uint64
		minerAddr proto.AccountAddress
		userAddr  proto.AccountAddress
		usersMap  = make(map[proto.AccountAddress]uint64)
		minersMap = make(map[proto.AccountAddress]map[proto.AccountAddress]uint64)
	)

	for i = 0; i < c.updatePeriod && node != nil; i++ {
		var block = node.block
		// Not cached, recover from storage
		if block == nil {
			if block, err = c.FetchBlock(node.height); err != nil {
				return
			}
		}
		for _, tx := range block.QueryTxs {
			minerAddr = tx.Response.ResponseAccount
			if userAddr, err = crypto.PubKeyHash(tx.Request.Header.Signee); err != nil {
				log.WithError(err).WithField("db", c.databaseID).Warning("billing fail: miner addr")
				return
			}

			if _, ok := minersMap[userAddr]; !ok {
				minersMap[userAddr] = make(map[proto.AccountAddress]uint64)
			}
			if tx.Request.Header.QueryType == types.ReadQuery {
				minersMap[userAddr][minerAddr] += tx.Response.RowCount
				usersMap[userAddr] += tx.Response.RowCount
			} else {
				minersMap[userAddr][minerAddr] += uint64(tx.Response.AffectedRows)
				usersMap[userAddr] += uint64(tx.Response.AffectedRows)
			}
		}

		for _, req := range block.FailedReqs {
			if minerAddr, err = crypto.PubKeyHash(block.Signee()); err != nil {
				log.WithError(err).WithField("db", c.databaseID).Warning("billing fail: miner addr")
				return
			}
			if userAddr, err = crypto.PubKeyHash(req.Header.Signee); err != nil {
				log.WithError(err).WithField("db", c.databaseID).Warning("billing fail: user addr")
				return
			}
			if _, ok := minersMap[userAddr][minerAddr]; !ok {
				minersMap[userAddr] = make(map[proto.AccountAddress]uint64)
			}

			minersMap[userAddr][minerAddr] += uint64(len(req.Payload.Queries))
			usersMap[userAddr] += uint64(len(req.Payload.Queries))
		}
		node = node.parent
	}

	ub = types.NewUpdateBilling(&types.UpdateBillingHeader{
		Users: make([]*types.UserCost, len(usersMap)),
	})

	i = 0
	j = 0
	for userAddr, cost := range usersMap {
		log.WithField("db", c.databaseID).Debugf("user %s, cost %d", userAddr.String(), cost)
		ub.Users[i] = &types.UserCost{
			User: userAddr,
			Cost: cost,
		}
		miners := minersMap[userAddr]
		ub.Users[i].Miners = make([]*types.MinerIncome, len(miners))

		for k1, v1 := range miners {
			ub.Users[i].Miners[j] = &types.MinerIncome{
				Miner:  k1,
				Income: v1,
			}
			j++
		}
		j = 0
		i++
	}
	ub.Receiver, err = c.databaseID.AccountAddress()
	return
}
