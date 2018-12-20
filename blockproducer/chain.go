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

package blockproducer

import (
	"context"
	"fmt"
	"math"
	"os"
	"sync"
	"time"

	pi "github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	"github.com/CovenantSQL/CovenantSQL/chainbus"
	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/merkle"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	xi "github.com/CovenantSQL/CovenantSQL/xenomint/interfaces"
	"github.com/pkg/errors"
)

// Chain defines the main chain.
type Chain struct {
	// Routine controlling components
	ctx    context.Context
	cancel context.CancelFunc
	wg     *sync.WaitGroup
	// RPC components
	server *rpc.Server
	cl     *rpc.Caller
	// Other components
	st xi.Storage
	bs chainbus.Bus
	// Channels for incoming blocks and transactions
	pendingBlocks chan *types.BPBlock
	pendingTxs    chan pi.Transaction
	// The following fields are read-only in runtime
	address     proto.AccountAddress
	genesisTime time.Time
	period      time.Duration
	tick        time.Duration

	sync.RWMutex // protects following fields
	peers        *proto.Peers
	nodeID       proto.NodeID
	confirms     uint32
	serversNum   uint32
	locSvIndex   uint32
	nextHeight   uint32
	offset       time.Duration
	lastIrre     *blockNode
	immutable    *metaState
	headIndex    int
	headBranch   *branch
	branches     []*branch
	txPool       map[hash.Hash]pi.Transaction
}

// NewChain creates a new blockchain.
func NewChain(cfg *Config) (c *Chain, err error) {
	return NewChainWithContext(context.Background(), cfg)
}

// NewChainWithContext creates a new blockchain with context.
func NewChainWithContext(ctx context.Context, cfg *Config) (c *Chain, err error) {
	var (
		existed bool
		ok      bool
		ierr    error

		cld context.Context
		ccl context.CancelFunc
		l   = uint32(len(cfg.Peers.Servers))
		t   float64
		m   uint32

		st        xi.Storage
		irre      *blockNode
		heads     []*blockNode
		immutable *metaState
		txPool    map[hash.Hash]pi.Transaction

		branches  []*branch
		br, head  *branch
		headIndex int

		pubKey     *asymmetric.PublicKey
		addr       proto.AccountAddress
		locSvIndex int32

		bus = chainbus.New()
	)

	if fi, err := os.Stat(cfg.DataFile); err == nil && fi.Mode().IsRegular() {
		existed = true
	}

	// Open storage
	if st, ierr = openStorage(fmt.Sprintf("file:%s", cfg.DataFile)); ierr != nil {
		err = errors.Wrap(ierr, "failed to open storage")
		return
	}

	// Storage genesis
	if !existed {
		// TODO(leventeliu): reuse chain.switchBranch to construct initial state.
		var init = newMetaState()
		for _, v := range cfg.Genesis.Transactions {
			if ierr = init.apply(v); ierr != nil {
				err = errors.Wrap(ierr, "failed to initialize immutable state")
				return
			}
		}
		var sps []storageProcedure
		sps = append(sps, addBlock(0, cfg.Genesis))
		for k, v := range init.dirty.accounts {
			if v != nil {
				sps = append(sps, updateAccount(&v.Account))
			} else {
				sps = append(sps, deleteAccount(k))
			}
		}
		for k, v := range init.dirty.databases {
			if v != nil {
				sps = append(sps, updateShardChain(&v.SQLChainProfile))
			} else {
				sps = append(sps, deleteShardChain(k))
			}
		}
		for k, v := range init.dirty.provider {
			if v != nil {
				sps = append(sps, updateProvider(&v.ProviderProfile))
			} else {
				sps = append(sps, deleteProvider(k))
			}
		}
		sps = append(sps, updateIrreversible(cfg.Genesis.SignedHeader.BlockHash))
		if ierr = store(st, sps, nil); ierr != nil {
			err = errors.Wrap(ierr, "failed to initialize storage")
			return
		}
	}

	// Load from database and rebuild branches
	if irre, heads, immutable, txPool, ierr = loadDatabase(st); ierr != nil {
		err = errors.Wrap(ierr, "failed to load data from storage")
		return
	}
	for _, v := range heads {
		log.WithFields(log.Fields{
			"irre_hash":  irre.hash.Short(4),
			"irre_count": irre.count,
			"head_hash":  v.hash.Short(4),
			"head_count": v.count,
		}).Debug("checking head")
		if v.hasAncestor(irre) {
			if br, ierr = fork(irre, v, immutable, txPool); ierr != nil {
				err = errors.Wrapf(ierr, "failed to rebuild branch with head %s", v.hash.Short(4))
				return
			}
			branches = append(branches, br)
		}
	}
	if len(branches) == 0 {
		err = ErrNoAvailableBranch
		return
	}

	// Set head branch
	for i, v := range branches {
		if head == nil || v.head.count > head.head.count {
			headIndex = i
			head = v
		}
	}

	// Get accountAddress
	if pubKey, err = kms.GetLocalPublicKey(); err != nil {
		return
	}
	if addr, err = crypto.PubKeyHash(pubKey); err != nil {
		return
	}

	// Setup peer list
	if locSvIndex, ok = cfg.Peers.Find(cfg.NodeID); !ok {
		err = ErrLocalNodeNotFound
		return
	}
	if t = cfg.ConfirmThreshold; t <= 0.0 {
		t = float64(2) / 3.0
	}
	if m = uint32(math.Ceil(float64(l)*t + 1)); m > l {
		m = l
	}

	// create chain
	cld, ccl = context.WithCancel(ctx)
	c = &Chain{
		ctx:    cld,
		cancel: ccl,
		wg:     &sync.WaitGroup{},

		server: cfg.Server,
		cl:     rpc.NewCaller(),

		st: st,
		bs: bus,

		pendingBlocks: make(chan *types.BPBlock),
		pendingTxs:    make(chan pi.Transaction),

		address:     addr,
		genesisTime: cfg.Genesis.SignedHeader.Timestamp,
		period:      cfg.Period,
		tick:        cfg.Tick,

		peers:      cfg.Peers,
		nodeID:     cfg.NodeID,
		confirms:   m,
		serversNum: l,
		locSvIndex: uint32(locSvIndex),
		nextHeight: head.head.height + 1,
		offset:     time.Duration(0), // TODO(leventeliu): initialize offset

		lastIrre:   irre,
		immutable:  immutable,
		headIndex:  headIndex,
		headBranch: head,
		branches:   branches,
		txPool:     txPool,
	}
	log.WithFields(log.Fields{
		"index":     c.locSvIndex,
		"bp_number": c.serversNum,
		"period":    c.period.String(),
		"tick":      c.tick.String(),
		"height":    c.head().height,
	}).Debug("current chain state")
	return
}

// checkBlock has following steps: 1. check parent block 2. checkTx 2. merkle tree 3. Hash 4. Signature.
func (c *Chain) checkBlock(b *types.BPBlock) (err error) {
	rootHash := merkle.NewMerkle(b.GetTxHashes()).GetRoot()
	if !b.SignedHeader.MerkleRoot.IsEqual(rootHash) {
		return ErrInvalidMerkleTreeRoot
	}

	enc, err := b.SignedHeader.BPHeader.MarshalHash()
	if err != nil {
		return err
	}
	h := hash.THashH(enc)
	if !b.BlockHash().IsEqual(&h) {
		return ErrInvalidHash
	}

	return nil
}

func (c *Chain) pushBlock(b *types.BPBlock) (err error) {
	var ierr error
	if ierr = c.checkBlock(b); ierr != nil {
		err = errors.Wrap(ierr, "failed to check block")
		return
	}
	if ierr = c.applyBlock(b); ierr != nil {
		err = errors.Wrap(ierr, "failed to apply block")
		return
	}
	return
}

func (c *Chain) produceBlock(now time.Time) (err error) {
	var (
		priv *asymmetric.PrivateKey
		b    *types.BPBlock
	)

	if priv, err = kms.GetLocalPrivateKey(); err != nil {
		return
	}
	if b, err = c.produceAndStoreBlock(now, priv); err != nil {
		return
	}
	log.WithField("block", b).Debug("produced new block")

	for _, s := range c.getPeers().Servers {
		if !s.IsEqual(&c.nodeID) {
			// Bind NodeID to subroutine
			func(id proto.NodeID) {
				c.goFuncWithTimeout(func(ctx context.Context) {
					var (
						req = &types.AdviseNewBlockReq{
							Envelope: proto.Envelope{
								// TODO(lambda): Add fields.
							},
							Block: b,
						}
						resp = &types.AdviseNewBlockResp{}
						err  = c.cl.CallNodeWithContext(
							ctx, id, route.MCCAdviseNewBlock.String(), req, resp)
					)
					log.WithFields(log.Fields{
						"local":       c.peerInfo(),
						"remote":      id,
						"block_time":  b.Timestamp(),
						"block_hash":  b.BlockHash().Short(4),
						"parent_hash": b.ParentHash().Short(4),
					}).WithError(err).Debug("broadcasting new block to other peers")
				}, c.period)
			}(s)
		}
	}

	return err
}

func (c *Chain) fetchBlock(h hash.Hash) (b *types.BPBlock, err error) {
	var (
		enc []byte
		out = &types.BPBlock{}
	)
	if err = c.st.Reader().QueryRow(
		`SELECT "encoded" FROM "blocks" WHERE "hash"=?`, h.String(),
	).Scan(&enc); err != nil {
		return
	}
	if err = utils.DecodeMsgPack(enc, out); err != nil {
		return
	}
	b = out
	return
}

func (c *Chain) fetchBlockByHeight(h uint32) (b *types.BPBlock, count uint32, err error) {
	var node = c.head().ancestor(h)
	if node == nil {
		err = ErrNoSuchBlock
		return
	} else if node.block != nil {
		b = node.block
		count = node.count
		return
	}
	// Not cached, read from database
	if b, err = c.fetchBlock(node.hash); err != nil {
		return
	}
	count = node.count
	return
}

func (c *Chain) fetchBlockByCount(count uint32) (b *types.BPBlock, height uint32, err error) {
	var node = c.head().ancestorByCount(count)
	if node == nil {
		err = ErrNoSuchBlock
		return
	} else if node.block != nil {
		b = node.block
		height = node.height
		return
	}
	// Not cached, read from database
	if b, err = c.fetchBlock(node.hash); err != nil {
		return
	}
	height = node.height
	return
}

func (c *Chain) fetchLastBlock() (b *types.BPBlock, count uint32, height uint32, err error) {
	var node = c.head()
	if node == nil {
		err = ErrNoSuchBlock
		return
	} else if node.block != nil {
		b = node.block
		height = node.height
		count = node.count
		return
	}
	// Not cached, read from database
	if b, err = c.fetchBlock(node.hash); err != nil {
		return
	}
	height = node.height
	count = node.count
	return
}

// advanceNextHeight does the check and runs block producing if its my turn.
func (c *Chain) advanceNextHeight(now time.Time) {
	log.WithFields(log.Fields{
		"next_height": c.getNextHeight(),
		"bp_number":   c.serversNum,
		"node_index":  c.locSvIndex,
	}).Info("check turns")
	defer c.increaseNextHeight()

	if !c.isMyTurn() {
		return
	}

	log.WithField("height", c.getNextHeight()).Info("producing a new block")
	if err := c.produceBlock(now); err != nil {
		log.WithField("now", now.Format(time.RFC3339Nano)).WithError(err).Errorln(
			"failed to produce block")
	}
}

func (c *Chain) syncHeads() {
	for {
		var h = c.heightOfTime(c.now())
		if c.getNextHeight() > h {
			break
		}
		for c.getNextHeight() <= h {
			// TODO(leventeliu): use the test mode flag to bypass the long-running synchronizing
			// on startup by now, need better solution here.
			if !conf.GConf.IsTestMode {
				log.WithFields(log.Fields{
					"next_height": c.getNextHeight(),
					"height":      h,
				}).Debug("synchronizing head blocks")
				c.syncCurrentHead(c.ctx)
			}
			c.increaseNextHeight()
		}
	}
}

// Start starts the chain by step:
// 1. sync the chain
// 2. goroutine for getting blocks
// 3. goroutine for getting txes.
func (c *Chain) Start() {
	// Start blocks/txs processing goroutines
	c.goFunc(c.processBlocks)
	c.goFunc(c.processTxs)
	// Synchronize heads to current block period
	c.syncHeads()
	// TODO(leventeliu): subscribe ChainBus.
	// ...
	// Start main cycle and service
	c.goFunc(c.mainCycle)
	c.startService(c)
}

func (c *Chain) processBlocks(ctx context.Context) {
	for {
		select {
		case block := <-c.pendingBlocks:
			err := c.pushBlock(block)
			if err != nil {
				log.WithFields(log.Fields{
					"block_hash":        block.BlockHash(),
					"block_parent_hash": block.ParentHash(),
					"block_timestamp":   block.Timestamp(),
				}).Debug(err)
			}
		case <-ctx.Done():
			log.WithError(c.ctx.Err()).Info("abort block processing")
			return
		}
	}
}

func (c *Chain) addTx(tx pi.Transaction) {
	// Simple non-blocking broadcasting
	for _, v := range c.getPeers().Servers {
		if !v.IsEqual(&c.nodeID) {
			// Bind NodeID to subroutine
			func(id proto.NodeID) {
				c.goFuncWithTimeout(func(ctx context.Context) {
					var (
						req = &types.AddTxReq{
							Envelope: proto.Envelope{
								// TODO(lambda): Add fields.
							},
							Tx: tx,
						}
						resp = &types.AddTxResp{}
						err  = c.cl.CallNodeWithContext(
							ctx, id, route.MCCAddTx.String(), req, resp)
					)
					log.WithFields(log.Fields{
						"local":   c.peerInfo(),
						"remote":  id,
						"tx_hash": tx.Hash().Short(4),
						"tx_type": tx.GetTransactionType(),
					}).WithError(err).Debug("broadcasting transaction to other peers")
				}, c.period)
			}(v)
		}
	}

	select {
	case c.pendingTxs <- tx:
	case <-c.ctx.Done():
		log.WithError(c.ctx.Err()).Warn("add transaction aborted")
	}
}

func (c *Chain) processTx(tx pi.Transaction) {
	if err := tx.Verify(); err != nil {
		log.WithError(err).Error("failed to verify transaction")
		return
	}
	if err := c.storeTx(tx); err != nil {
		log.WithError(err).Error("failed to add transaction")
	}
}

func (c *Chain) processTxs(ctx context.Context) {
	for {
		select {
		case tx := <-c.pendingTxs:
			c.processTx(tx)
		case <-ctx.Done():
			log.WithError(c.ctx.Err()).Info("abort transaction processing")
			return
		}
	}
}

func (c *Chain) mainCycle(ctx context.Context) {
	var timer = time.NewTimer(0)
	defer func() {
		if !timer.Stop() {
			<-timer.C
		}
	}()
	for {
		select {
		case <-timer.C:
			c.syncCurrentHead(ctx) // Try to fetch block at height `nextHeight-1`
			var t, d = c.nextTick()
			if d <= 0 {
				// Try to produce block at `nextHeight` if it's my turn, and increase height by 1
				c.advanceNextHeight(t)
			} else {
				log.WithFields(log.Fields{
					"peer":        c.peerInfo(),
					"next_height": c.getNextHeight(),
					"head_height": c.head().height,
					"head_block":  c.head().hash.Short(4),
					"now_time":    t.Format(time.RFC3339Nano),
					"duration":    d,
				}).Debug("main cycle")
			}
			timer.Reset(d)
		case <-ctx.Done():
			log.WithError(ctx.Err()).Info("abort main cycle")
			return
		}
	}
}

func (c *Chain) syncCurrentHead(ctx context.Context) {
	var (
		h        = c.getNextHeight() - 1
		cld, ccl = context.WithTimeout(ctx, c.tick)
		wg       = &sync.WaitGroup{}
	)

	defer func() {
		wg.Wait()
		ccl()
	}()

	if c.head().height >= h {
		return
	}

	// Initiate blocking gossip calls to fetch block of the current height,
	// with timeout of one tick.
	for _, v := range c.getPeers().Servers {
		if !v.IsEqual(&c.nodeID) {
			wg.Add(1)
			go func(id proto.NodeID) {
				defer wg.Done()
				var (
					err error
					req = &types.FetchBlockReq{
						Envelope: proto.Envelope{
							// TODO(lambda): Add fields.
						},
						Height: h,
					}
					resp = &types.FetchBlockResp{}
				)
				if err = c.cl.CallNodeWithContext(
					cld, id, route.MCCFetchBlock.String(), req, resp,
				); err != nil {
					log.WithFields(log.Fields{
						"local":  c.peerInfo(),
						"remote": id,
						"height": h,
					}).WithError(err).Warn("failed to fetch block")
					return
				}
				log.WithFields(log.Fields{
					"local":  c.peerInfo(),
					"remote": id,
					"height": h,
					"parent": resp.Block.ParentHash().Short(4),
					"hash":   resp.Block.BlockHash().Short(4),
				}).Debug("fetched new block from remote peer")
				select {
				case c.pendingBlocks <- resp.Block:
				case <-cld.Done():
					log.WithError(cld.Err()).Warn("add pending block aborted")
				}
			}(v)
		}
	}
}

// Stop stops the main process of the sql-chain.
func (c *Chain) Stop() (err error) {
	// Stop main process
	log.WithFields(log.Fields{"peer": c.peerInfo()}).Debug("stopping chain")
	c.stop()
	log.WithFields(log.Fields{"peer": c.peerInfo()}).Debug("chain service stopped")
	c.st.Close()
	log.WithFields(log.Fields{"peer": c.peerInfo()}).Debug("chain database closed")
	close(c.pendingBlocks)
	close(c.pendingTxs)
	return
}

func (c *Chain) storeTx(tx pi.Transaction) (err error) {
	var k = tx.Hash()
	c.Lock()
	defer c.Unlock()
	if _, ok := c.txPool[k]; ok {
		err = ErrExistedTx
		return
	}

	return store(c.st, []storageProcedure{addTx(tx)}, func() {
		c.txPool[k] = tx
		for _, v := range c.branches {
			v.addTx(tx)
		}
	})
}

func (c *Chain) nextNonce(addr proto.AccountAddress) (n pi.AccountNonce, err error) {
	c.RLock()
	defer c.RUnlock()
	return c.headBranch.preview.nextNonce(addr)
}

func (c *Chain) loadAccountCovenantBalance(addr proto.AccountAddress) (balance uint64, ok bool) {
	c.RLock()
	defer c.RUnlock()
	return c.immutable.loadAccountCovenantBalance(addr)
}

func (c *Chain) loadAccountStableBalance(addr proto.AccountAddress) (balance uint64, ok bool) {
	c.RLock()
	defer c.RUnlock()
	return c.immutable.loadAccountStableBalance(addr)
}

func (c *Chain) loadSQLChainProfile(databaseID proto.DatabaseID) (profile *types.SQLChainProfile, ok bool) {
	c.RLock()
	defer c.RUnlock()
	profileObj, ok := c.immutable.loadSQLChainObject(databaseID)
	profile = &profileObj.SQLChainProfile
	return
}

func (c *Chain) switchBranch(bl *types.BPBlock, origin int, head *branch) (err error) {
	var (
		irre     *blockNode
		newIrres []*blockNode
		sps      []storageProcedure
		up       storageCallback
		height   = c.heightOfTime(bl.Timestamp())
	)

	// Find new irreversible blocks
	//
	// NOTE(leventeliu):
	// May have multiple new irreversible blocks here if peer list shrinks. May also have
	// no new irreversible block at all if peer list expands.
	irre = head.head.lastIrreversible(c.confirms)
	newIrres = irre.fetchNodeList(c.lastIrre.count)

	// Apply irreversible blocks to create dirty map on immutable cache
	//
	// TODO(leventeliu): use old metaState for now, better use separated dirty cache.
	for _, b := range newIrres {
		for _, tx := range b.block.Transactions {
			if err := c.immutable.apply(tx); err != nil {
				log.WithError(err).Fatal("failed to apply block to immutable database")
			}
		}
	}

	// Prepare storage procedures to update immutable database
	sps = append(sps, addBlock(height, bl))
	for k, v := range c.immutable.dirty.accounts {
		if v != nil {
			sps = append(sps, updateAccount(&v.Account))
		} else {
			sps = append(sps, deleteAccount(k))
		}
	}
	for k, v := range c.immutable.dirty.databases {
		if v != nil {
			sps = append(sps, updateShardChain(&v.SQLChainProfile))
		} else {
			sps = append(sps, deleteShardChain(k))
		}
	}
	for k, v := range c.immutable.dirty.provider {
		if v != nil {
			sps = append(sps, updateProvider(&v.ProviderProfile))
		} else {
			sps = append(sps, deleteProvider(k))
		}
	}
	for _, n := range newIrres {
		sps = append(sps, deleteTxs(n.block.Transactions))
	}
	sps = append(sps, updateIrreversible(irre.hash))

	// Prepare callback to update cache
	up = func() {
		// Update last irreversible block
		c.lastIrre = irre
		// Apply irreversible blocks to immutable database
		c.immutable.commit()
		// Prune branches
		var (
			idx int
			brs = make([]*branch, 0, len(c.branches))
		)
		for i, b := range c.branches {
			if i == origin {
				// Current branch
				brs = append(brs, head)
				idx = len(brs) - 1
			} else if b.head.hasAncestor(irre) {
				brs = append(brs, b)
			} else {
				log.WithFields(log.Fields{
					"branch": func() string {
						if i == c.headIndex {
							return "[head]"
						}
						return fmt.Sprintf("[%04d]", i)
					}(),
				}).Debugf("pruning branch")
			}
		}
		// Replace current branches
		c.headBranch = head
		c.headIndex = idx
		c.branches = brs
		// Clear packed transactions
		for _, b := range newIrres {
			for _, br := range c.branches {
				br.clearPackedTxs(b.block.Transactions)
			}
			for _, tx := range b.block.Transactions {
				delete(c.txPool, tx.Hash())
			}
		}
	}

	// Write to immutable database and update cache
	if err = store(c.st, sps, up); err != nil {
		c.immutable.clean()
	}
	// TODO(leventeliu): trigger ChainBus.Publish.
	// ...
	return
}

func (c *Chain) stat() {
	c.RLock()
	defer c.RUnlock()
	for i, v := range c.branches {
		var buff string
		if i == c.headIndex {
			buff += "[head] "
		} else {
			buff += fmt.Sprintf("[%04d] ", i)
		}
		buff += v.sprint(c.lastIrre.count)
		log.WithFields(log.Fields{
			"branch": buff,
		}).Info("runtime state")
	}
}

func (c *Chain) applyBlock(bl *types.BPBlock) (err error) {
	var (
		ok     bool
		ierr   error
		br     *branch
		parent *blockNode
		head   *blockNode
		height = c.heightOfTime(bl.Timestamp())
	)

	defer c.stat()
	c.Lock()
	defer c.Unlock()

	for i, v := range c.branches {
		// Grow a branch
		if v.head.hash.IsEqual(bl.ParentHash()) {
			head = newBlockNode(height, bl, v.head)
			if br, ierr = v.applyBlock(head); ierr != nil {
				err = errors.Wrapf(ierr, "failed to apply block %s", head.hash.Short(4))
				return
			}
			// Grow a branch while the current branch is not changed
			if br.head.count <= c.headBranch.head.count {
				return store(c.st,
					[]storageProcedure{addBlock(height, bl)},
					func() { c.branches[i] = br },
				)
			}
			// Switch branch or grow current branch
			return c.switchBranch(bl, i, br)
		}
	}

	for _, v := range c.branches {
		if n := v.head.ancestor(height); n != nil && n.hash.IsEqual(bl.BlockHash()) {
			// Return silently if block exists in the current branch
			return
		}
		// Fork and create new branch
		if parent, ok = v.head.canForkFrom(bl.SignedHeader.ParentHash, c.lastIrre.count); ok {
			head = newBlockNode(height, bl, parent)
			if br, ierr = fork(c.lastIrre, head, c.immutable, c.txPool); ierr != nil {
				err = errors.Wrapf(ierr, "failed to fork from %s", parent.hash.Short(4))
				return
			}
			return store(c.st,
				[]storageProcedure{addBlock(height, bl)},
				func() { c.branches = append(c.branches, br) },
			)
		}
	}

	err = ErrParentNotFound
	return
}

func (c *Chain) produceAndStoreBlock(
	now time.Time, priv *asymmetric.PrivateKey) (out *types.BPBlock, err error,
) {
	var (
		bl   *types.BPBlock
		br   *branch
		ierr error
	)

	defer c.stat()
	c.Lock()
	defer c.Unlock()

	// Try to produce new block
	if br, bl, ierr = c.headBranch.produceBlock(
		c.heightOfTime(now), now, c.address, priv,
	); ierr != nil {
		err = errors.Wrapf(ierr, "failed to produce block at head %s",
			c.headBranch.head.hash.Short(4))
		return
	}
	if ierr = c.switchBranch(bl, c.headIndex, br); ierr != nil {
		err = errors.Wrapf(ierr, "failed to switch branch #%d:%s",
			c.headIndex, c.headBranch.head.hash.Short(4))
		return
	}
	out = bl
	return
}

// now returns the current coordinated chain time.
func (c *Chain) now() time.Time {
	c.RLock()
	defer c.RUnlock()
	return time.Now().UTC().Add(c.offset)
}

func (c *Chain) startService(chain *Chain) {
	c.server.RegisterService(route.BlockProducerRPCName, &ChainRPCService{chain: chain})
}

// nextTick returns the current clock reading and the duration till the next turn. If duration
// is less or equal to 0, use the clock reading to run the next cycle - this avoids some problem
// caused by concurrent time synchronization.
func (c *Chain) nextTick() (t time.Time, d time.Duration) {
	var h uint32
	h, t = func() (nt uint32, t time.Time) {
		c.RLock()
		defer c.RUnlock()
		nt = c.nextHeight
		t = time.Now().UTC().Add(c.offset)
		return
	}()
	d = c.genesisTime.Add(time.Duration(h) * c.period).Sub(t)
	if d > c.tick {
		d = c.tick
	}
	return
}

func (c *Chain) isMyTurn() bool {
	c.RLock()
	defer c.RUnlock()
	return c.nextHeight%c.serversNum == c.locSvIndex
}

// increaseNextHeight prepares the chain state for the next turn.
func (c *Chain) increaseNextHeight() {
	c.Lock()
	defer c.Unlock()
	c.nextHeight++
}

func (c *Chain) peerInfo() string {
	var index, bpNum, nodeID = func() (uint32, uint32, proto.NodeID) {
		c.RLock()
		defer c.RUnlock()
		return c.locSvIndex, c.serversNum, c.nodeID
	}()
	return fmt.Sprintf("[%d/%d] %s", index, bpNum, nodeID)
}

// heightOfTime calculates the heightOfTime with this sql-chain config of a given time reading.
func (c *Chain) heightOfTime(t time.Time) uint32 {
	return uint32(t.Sub(c.genesisTime) / c.period)
}

func (c *Chain) getNextHeight() uint32 {
	c.RLock()
	defer c.RUnlock()
	return c.nextHeight
}

func (c *Chain) getPeers() *proto.Peers {
	c.RLock()
	defer c.RUnlock()
	var peers = c.peers.Clone()
	return &peers
}

func (c *Chain) head() *blockNode {
	c.RLock()
	defer c.RUnlock()
	return c.headBranch.head
}

func (c *Chain) stop() {
	c.cancel()
	c.wg.Wait()
}

func (c *Chain) goFunc(f func(ctx context.Context)) {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		f(c.ctx)
	}()
}

func (c *Chain) goFuncWithTimeout(f func(ctx context.Context), timeout time.Duration) {
	c.wg.Add(1)
	go func() {
		var ctx, ccl = context.WithTimeout(c.ctx, timeout)
		defer func() {
			ccl()
			c.wg.Done()
		}()
		f(ctx)
	}()
}
