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
	"expvar"
	"fmt"
	"math"
	"os"
	"sync"
	"time"

	mw "github.com/zserge/metric"

	pi "github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	"github.com/CovenantSQL/CovenantSQL/chainbus"
	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	"github.com/CovenantSQL/CovenantSQL/types"
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
	caller *rpc.Caller
	// Other components
	storage  xi.Storage
	chainBus chainbus.Bus
	// Channels for incoming blocks and transactions
	pendingBlocks    chan *types.BPBlock
	pendingAddTxReqs chan *types.AddTxReq
	// The following fields are read-only in runtime
	address     proto.AccountAddress
	genesisTime time.Time
	period      time.Duration
	tick        time.Duration

	sync.RWMutex // protects following fields
	bpInfos      []*blockProducerInfo
	localBPInfo  *blockProducerInfo
	localNodeID  proto.NodeID
	confirms     uint32
	nextHeight   uint32
	offset       time.Duration
	lastIrre     *blockNode
	immutable    *metaState
	headIndex    int
	headBranch   *branch
	branches     []*branch
	txPool       map[hash.Hash]pi.Transaction
	mode         RunMode
}

// NewChain creates a new blockchain.
func NewChain(cfg *Config) (c *Chain, err error) {
	// Normally, NewChain() should only be called once in app.
	// So, we just check expvar without a lock
	if expvar.Get("height") == nil {
		expvar.Publish("height", mw.NewGauge("5m1s"))
	}
	return NewChainWithContext(context.Background(), cfg)
}

// NewChainWithContext creates a new blockchain with context.
func NewChainWithContext(ctx context.Context, cfg *Config) (c *Chain, err error) {
	var (
		existed bool
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

		pubKey      *asymmetric.PublicKey
		addr        proto.AccountAddress
		bpInfos     []*blockProducerInfo
		localBPInfo *blockProducerInfo

		bus = chainbus.New()
	)

	// Verify genesis block in config
	if cfg.Genesis == nil {
		err = ErrNilGenesis
		return
	}
	if ierr = cfg.Genesis.VerifyHash(); ierr != nil {
		err = errors.Wrap(ierr, "failed to verify genesis block hash")
		return
	}

	// Open storage
	if fi, err := os.Stat(cfg.DataFile); err == nil && fi.Mode().IsRegular() {
		existed = true
	}
	if st, ierr = openStorage(fmt.Sprintf("file:%s", cfg.DataFile)); ierr != nil {
		err = errors.Wrap(ierr, "failed to open storage")
		return
	}
	defer func() {
		if err != nil {
			st.Close()
		}
	}()

	// Create initial state from genesis block and store
	if !existed {
		var init = newMetaState()
		for _, v := range cfg.Genesis.Transactions {
			if ierr = init.apply(v); ierr != nil {
				err = errors.Wrap(ierr, "failed to initialize immutable state")
				return
			}
		}
		var sps = init.compileChanges(nil)
		sps = append(sps, addBlock(0, cfg.Genesis))
		sps = append(sps, updateIrreversible(cfg.Genesis.SignedHeader.DataHash))
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
	if persistedGenesis := irre.ancestorByCount(0); persistedGenesis == nil ||
		!persistedGenesis.hash.IsEqual(cfg.Genesis.BlockHash()) {
		err = ErrGenesisHashNotMatch
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
			if br, ierr = newBranch(irre, v, immutable, txPool); ierr != nil {
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
	if localBPInfo, bpInfos, err = buildBlockProducerInfos(cfg.NodeID, cfg.Peers, cfg.Mode == APINodeMode); err != nil {
		return
	}
	if t = cfg.ConfirmThreshold; t <= 0.0 {
		t = conf.DefaultConfirmThreshold
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
		caller: rpc.NewCaller(),

		storage:  st,
		chainBus: bus,

		pendingBlocks:    make(chan *types.BPBlock),
		pendingAddTxReqs: make(chan *types.AddTxReq),

		address:     addr,
		genesisTime: cfg.Genesis.SignedHeader.Timestamp,
		period:      cfg.Period,
		tick:        cfg.Tick,

		bpInfos:     bpInfos,
		localBPInfo: localBPInfo,
		localNodeID: cfg.NodeID,
		confirms:    m,
		nextHeight:  head.head.height + 1,
		offset:      time.Duration(0), // TODO(leventeliu): initialize offset

		lastIrre:   irre,
		immutable:  immutable,
		headIndex:  headIndex,
		headBranch: head,
		branches:   branches,
		txPool:     txPool,
		mode:       cfg.Mode,
	}
	log.WithFields(log.Fields{
		"local":  c.getLocalBPInfo(),
		"period": c.period,
		"tick":   c.tick,
		"height": c.head().height,
	}).Debug("current chain state")
	return
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

// Stop stops the main process of the sql-chain.
func (c *Chain) Stop() (err error) {
	// Stop main process
	var le = log.WithFields(log.Fields{
		"local": c.getLocalBPInfo(),
	})
	le.Debug("stopping chain")
	c.stop()
	le.Debug("chain service stopped")
	c.storage.Close()
	le.Debug("chain database closed")

	// FIXME(leventeliu): RPC server should provide an `unregister` method to detach chain service
	// instance. Add it to Chain.stop(), then working channels can be closed safely.
	// Otherwise a DATARACE (while closing a channel with a blocking write from RPC service) or
	// `write on closed channel` panic may occur.
	// Comment this out for now, IT IS A RESOURCE LEAK.
	//
	//close(c.pendingBlocks)
	//close(c.pendingTxs)

	return
}

func (c *Chain) pushBlock(b *types.BPBlock) (err error) {
	var ierr error
	if ierr = b.Verify(); ierr != nil {
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

	log.WithFields(log.Fields{
		"block_time":  b.Timestamp(),
		"block_hash":  b.BlockHash().Short(4),
		"parent_hash": b.ParentHash().Short(4),
	}).Debug("produced new block")

	// Broadcast to other block producers
	c.nonblockingBroadcastBlock(b)
	return
}

// advanceNextHeight does the check and runs block producing if its my turn.
func (c *Chain) advanceNextHeight(now time.Time, d time.Duration) {
	var elapsed = -d

	log.WithFields(log.Fields{
		"local":            c.getLocalBPInfo(),
		"enclosing_height": c.getNextHeight() - 1,
		"using_timestamp":  now.Format(time.RFC3339Nano),
		"elapsed_seconds":  elapsed.Seconds(),
	}).Info("enclosing current height and advancing to next height")

	defer c.increaseNextHeight()
	// Skip if it's not my turn
	if c.mode == APINodeMode || !c.isMyTurn() {
		return
	}
	// Normally, a block producing should start right after the new period, but more time may also
	// elapse since the last block synchronizing.
	if elapsed+c.tick > c.period { // TODO(leventeliu): add threshold config for `elapsed`.
		log.WithFields(log.Fields{
			"advanced_height": c.getNextHeight(),
			"using_timestamp": now.Format(time.RFC3339Nano),
			"elapsed_seconds": elapsed.Seconds(),
		}).Warn("too much time elapsed in the new period, skip this block")
		return
	}
	expvar.Get("height").(mw.Metric).Add(float64(c.getNextHeight()))
	log.WithField("height", c.getNextHeight()).Info("producing a new block")
	if err := c.produceBlock(now); err != nil {
		log.WithField("now", now.Format(time.RFC3339Nano)).WithError(err).Errorln(
			"failed to produce block")
	}
}

func (c *Chain) syncHeads() {
	for {
		var (
			now       = c.now()
			nowHeight uint32
		)
		if now.Before(c.genesisTime) {
			log.WithFields(log.Fields{
				"local": c.getLocalBPInfo(),
			}).Info("now time is before genesis time, waiting for genesis")
			break
		}
		if nowHeight = c.heightOfTime(c.now()); c.getNextHeight() > nowHeight {
			break
		}
		for c.getNextHeight() <= nowHeight {
			// TODO(leventeliu): use the test mode flag to bypass the long-running synchronizing
			// on startup by now, need better solution here.
			if conf.GConf.StartupSyncHoles {
				log.WithFields(log.Fields{
					"local":       c.getLocalBPInfo(),
					"next_height": c.getNextHeight(),
					"now_height":  nowHeight,
				}).Debug("synchronizing head blocks")
				c.blockingSyncCurrentHead(c.ctx, conf.BPStartupRequiredReachableCount)
			}
			c.increaseNextHeight()
		}
	}
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

func (c *Chain) addTx(req *types.AddTxReq) {
	select {
	case c.pendingAddTxReqs <- req:
	case <-c.ctx.Done():
		log.WithError(c.ctx.Err()).Warn("add transaction aborted")
	}
}

func (c *Chain) processAddTxReq(addTxReq *types.AddTxReq) {
	// Nil check
	if addTxReq == nil || addTxReq.Tx == nil {
		log.Warn("empty add tx request")
		return
	}

	var (
		ttl = addTxReq.TTL
		tx  = addTxReq.Tx

		txhash = tx.Hash()
		addr   = tx.GetAccountAddress()
		nonce  = tx.GetAccountNonce()

		le = log.WithFields(log.Fields{
			"hash":    txhash.Short(4),
			"address": addr,
			"nonce":   nonce,
			"type":    tx.GetTransactionType(),
		})

		base pi.AccountNonce
		err  error
	)

	// Existense check
	if ok := func() (ok bool) {
		c.RLock()
		defer c.RUnlock()
		_, ok = c.txPool[txhash]
		return
	}(); ok {
		le.Debug("tx already exists, abort processing")
		return
	}

	// Verify transaction
	if err = tx.Verify(); err != nil {
		le.WithError(err).Warn("failed to verify transaction")
		return
	}
	if base, err = c.immutableNextNonce(addr); err != nil {
		le.WithError(err).Warn("failed to load base nonce of transaction account")
		return
	}
	if nonce < base || nonce >= base+conf.MaxPendingTxsPerAccount {
		// TODO(leventeliu): should persist to somewhere for tx query?
		le.WithFields(log.Fields{
			"base_nonce":    base,
			"pending_limit": conf.MaxPendingTxsPerAccount,
		}).Warn("invalid transaction nonce")
		return
	}

	// Broadcast to other block producers
	if ttl > conf.MaxTxBroadcastTTL {
		ttl = conf.MaxTxBroadcastTTL
	}
	if ttl > 0 {
		c.nonblockingBroadcastTx(ttl-1, tx)
	}

	// Add to tx pool
	if err = c.storeTx(tx); err != nil {
		le.WithError(err).Error("failed to add transaction")
	}
}

func (c *Chain) processTxs(ctx context.Context) {
	for {
		select {
		case addTxReq := <-c.pendingAddTxReqs:
			c.processAddTxReq(addTxReq)
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
			// Try to fetch block at height `nextHeight-1` until enough peers are reachable
			if err := c.blockingSyncCurrentHead(ctx, c.getRequiredConfirms()); err != nil {
				log.WithError(err).Info("abort main cycle")
				timer.Reset(0)
				return
			}

			var t, d = c.nextTick()
			if d <= 0 {
				// Try to produce block at `nextHeight` if it's my turn, and increase height by 1
				c.advanceNextHeight(t, d)
			} else {
				log.WithFields(log.Fields{
					"peer":        c.getLocalBPInfo(),
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

func (c *Chain) blockingSyncCurrentHead(ctx context.Context, requiredReachable uint32) (err error) {
	var (
		ticker   *time.Ticker
		interval = 1 * time.Second
	)
	if c.tick < interval {
		interval = c.tick
	}
	ticker = time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if c.syncCurrentHead(ctx, requiredReachable) {
			return
		}
		select {
		case <-ticker.C:
		case <-ctx.Done():
			err = ctx.Err()
			return
		}
	}
}

// syncCurrentHead synchronizes a block at the current height of the local peer from the known
// remote peers. The return value `ok` indicates that there're at least `requiredReachable-1`
// replies from these gossip calls.
func (c *Chain) syncCurrentHead(ctx context.Context, requiredReachable uint32) (ok bool) {
	var currentHeight = c.getNextHeight() - 1
	if c.head().height >= currentHeight {
		ok = true
		return
	}

	// Initiate blocking gossip calls to fetch block of the current height,
	// with timeout of one tick.
	var (
		unreachable = c.blockingFetchBlock(ctx, currentHeight)
		serversNum  = c.getLocalBPInfo().total
	)

	if ok = unreachable+requiredReachable <= serversNum; !ok {
		log.WithFields(log.Fields{
			"peer":              c.getLocalBPInfo(),
			"sync_head_height":  currentHeight,
			"unreachable_count": unreachable,
		}).Warn("one or more block producers are currently unreachable")
	}
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

	return store(c.storage, []storageProcedure{addTx(tx)}, func() {
		c.txPool[k] = tx
		for _, v := range c.branches {
			v.addTx(tx)
		}
	})
}

func (c *Chain) replaceAndSwitchToBranch(
	newBlock *types.BPBlock, originBrIdx int, newBranch *branch) (err error,
) {
	var (
		lastIrre *blockNode
		newIrres []*blockNode
		sps      []storageProcedure
		up       storageCallback
		height   = c.heightOfTime(newBlock.Timestamp())

		resultTxPool = make(map[hash.Hash]pi.Transaction)
		expiredTxs   []pi.Transaction
	)

	// Find new irreversible blocks
	//
	// NOTE(leventeliu):
	// May have multiple new irreversible blocks here if peer list shrinks. May also have
	// no new irreversible block at all if peer list expands.
	lastIrre = newBranch.head.lastIrreversible(c.confirms)
	newIrres = lastIrre.fetchNodeList(c.lastIrre.count)

	// Apply irreversible blocks to create dirty map on immutable cache
	for k, v := range c.txPool {
		resultTxPool[k] = v
	}
	for _, b := range newIrres {
		for _, tx := range b.block.Transactions {
			if err := c.immutable.apply(tx); err != nil {
				log.WithError(err).Fatal("failed to apply block to immutable database")
			}
			delete(resultTxPool, tx.Hash()) // Remove confirmed transaction
		}
	}

	// Check tx expiration
	for k, v := range resultTxPool {
		if base, err := c.immutable.nextNonce(
			v.GetAccountAddress(),
		); err != nil || v.GetAccountNonce() < base {
			log.WithFields(log.Fields{
				"hash":    k.Short(4),
				"type":    v.GetTransactionType(),
				"account": v.GetAccountAddress(),
				"nonce":   v.GetAccountNonce(),

				"immutable_base_nonce": base,
			}).Debug("transaction expired")
			expiredTxs = append(expiredTxs, v)
			delete(resultTxPool, k) // Remove expired transaction
		}
	}

	// Prepare storage procedures to update immutable database
	sps = c.immutable.compileChanges(sps)
	sps = append(sps, addBlock(height, newBlock))
	sps = append(sps, buildBlockIndex(height, newBlock))
	for _, n := range newIrres {
		sps = append(sps, deleteTxs(n.block.Transactions))
	}
	if len(expiredTxs) > 0 {
		sps = append(sps, deleteTxs(expiredTxs))
	}
	sps = append(sps, updateIrreversible(lastIrre.hash))

	// Prepare callback to update cache
	up = func() {
		// Update last irreversible block
		c.lastIrre = lastIrre
		// Apply irreversible blocks to immutable database
		c.immutable.commit()
		// Prune branches
		var (
			idx int
			brs = make([]*branch, 0, len(c.branches))
		)
		for i, b := range c.branches {
			if i == originBrIdx {
				// Replace origin branch with new branch
				newBranch.preview.commit()
				brs = append(brs, newBranch)
				idx = len(brs) - 1
			} else if b.head.hasAncestor(lastIrre) {
				// Move to new branches slice
				brs = append(brs, b)
			} else {
				// Prune this branch
				log.WithFields(log.Fields{
					"branch": func() string {
						if i == c.headIndex {
							return "[head]"
						}
						return fmt.Sprintf("[%04d]", i)
					}(),
				}).Debug("pruning branch")
			}
		}
		// Replace current branches
		c.headBranch = newBranch
		c.headIndex = idx
		c.branches = brs
		// Clear transactions in each branch
		for _, b := range newIrres {
			for _, br := range c.branches {
				br.clearPackedTxs(b.block.Transactions)
			}
		}
		for _, br := range c.branches {
			br.clearUnpackedTxs(expiredTxs)
		}
		// Update txPool to result txPool (packed and expired transactions cleared!)
		c.txPool = resultTxPool
	}

	// Write to immutable database and update cache
	if err = store(c.storage, sps, up); err != nil {
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
				return store(c.storage,
					[]storageProcedure{addBlock(height, bl)},
					func() {
						br.preview.commit()
						c.branches[i] = br
					},
				)
			}
			// Switch branch or grow current branch
			return c.replaceAndSwitchToBranch(bl, i, br)
		}
	}

	for _, v := range c.branches {
		if n := v.head.ancestor(height); n != nil && n.hash.IsEqual(bl.BlockHash()) {
			// Return silently if block exists in the current branch
			return
		}
	}

	for _, v := range c.branches {
		// Fork and create new branch
		if parent, ok = v.head.hasAncestorWithMinCount(
			bl.SignedHeader.ParentHash, c.lastIrre.count,
		); ok {
			head = newBlockNode(height, bl, parent)
			if br, ierr = newBranch(c.lastIrre, head, c.immutable, c.txPool); ierr != nil {
				err = errors.Wrapf(ierr, "failed to fork from %s", parent.hash.Short(4))
				return
			}
			return store(c.storage,
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
	if ierr = c.replaceAndSwitchToBranch(bl, c.headIndex, br); ierr != nil {
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
	return time.Now().Add(c.offset).UTC()
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
		t = time.Now().Add(c.offset).UTC()
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
	return c.nextHeight%c.localBPInfo.total == c.localBPInfo.rank
}

// increaseNextHeight prepares the chain state for the next turn.
func (c *Chain) increaseNextHeight() {
	c.Lock()
	defer c.Unlock()
	c.nextHeight++
}

// heightOfTime calculates the heightOfTime with this sql-chain config of a given time reading.
func (c *Chain) heightOfTime(t time.Time) uint32 {
	return uint32(t.Sub(c.genesisTime) / c.period)
}

func (c *Chain) getRequiredConfirms() uint32 {
	c.RLock()
	defer c.RUnlock()
	return c.confirms
}

func (c *Chain) getNextHeight() uint32 {
	c.RLock()
	defer c.RUnlock()
	return c.nextHeight
}

func (c *Chain) getLocalBPInfo() *blockProducerInfo {
	c.RLock()
	defer c.RUnlock()
	return c.localBPInfo
}

// getRemoteBPInfos remove this node from the peer list
func (c *Chain) getRemoteBPInfos() (remoteBPInfos []*blockProducerInfo) {
	var localBPInfo, bpInfos = func() (*blockProducerInfo, []*blockProducerInfo) {
		c.RLock()
		defer c.RUnlock()
		return c.localBPInfo, c.bpInfos
	}()

	for _, info := range bpInfos {
		if info.nodeID.IsEqual(&localBPInfo.nodeID) {
			continue
		}
		remoteBPInfos = append(remoteBPInfos, info)
	}

	return remoteBPInfos
}

func (c *Chain) lastIrreversibleBlock() *blockNode {
	c.RLock()
	defer c.RUnlock()
	return c.lastIrre
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
			c.wg.Done()
			ccl()
		}()
		f(ctx)
	}()
}
