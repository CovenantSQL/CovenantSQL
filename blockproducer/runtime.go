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
	"sync"
	"time"

	pi "github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	xi "github.com/CovenantSQL/CovenantSQL/xenomint/interfaces"
)

// copy from /sqlchain/runtime.go
// rt define the runtime of main chain.
type rt struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     *sync.WaitGroup

	// chainInitTime is the initial cycle time, when the Genesis block is produced.
	chainInitTime time.Time

	accountAddress proto.AccountAddress
	server         *rpc.Server

	bpNum uint32
	// index is the index of the current server in the peer list.
	index uint32

	// period is the block producing cycle.
	period time.Duration
	// tick defines the maximum duration between each cycle.
	tick time.Duration

	// peersMutex protects following peers-relative fields.
	peersMutex sync.Mutex
	peers      *proto.Peers
	nodeID     proto.NodeID
	minComfirm uint32

	stateMutex sync.Mutex // Protects following fields.
	// nextTurn is the height of the next block.
	nextTurn uint32

	// timeMutex protects following time-relative fields.
	timeMutex sync.Mutex
	// offset is the time difference calculated by: coodinatedChainTime - time.Now().
	//
	// TODO(leventeliu): update offset in ping cycle.
	offset time.Duration

	// Cached state
	cacheMu   sync.RWMutex
	irre      *blockNode
	immutable *metaState
	currIdx   int
	current   *branch
	branches  []*branch
	txPool    map[hash.Hash]pi.Transaction
}

func (r *rt) addTx(st xi.Storage, tx pi.Transaction) (err error) {
	return store(st, []storageProcedure{addTx(tx)}, func() {
		var k = tx.Hash()
		r.cacheMu.Lock()
		defer r.cacheMu.Unlock()
		if _, ok := r.txPool[k]; !ok {
			r.txPool[k] = tx
		}
		for _, v := range r.branches {
			v.addTx(tx)
		}
	})
}

func (r *rt) switchBranch(st xi.Storage, bl *types.BPBlock, origin int, head *branch) (err error) {
	var (
		irre     *blockNode
		newIrres []*blockNode
		sps      []storageProcedure
		up       storageCallback
	)

	// Find new irreversible blocks
	//
	// NOTE(leventeliu):
	// May have multiple new irreversible blocks here if peer list shrinks. May also have
	// no new irreversible block at all if peer list expands.
	irre = head.head.lastIrreversible(r.minComfirm)
	newIrres = irre.blockNodeListFrom(r.irre.count)

	// Apply irreversible blocks to create dirty map on immutable cache
	//
	// TODO(leventeliu): use old metaState for now, better use separated dirty cache.
	for _, b := range newIrres {
		for _, tx := range b.block.Transactions {
			if err := r.immutable.apply(tx); err != nil {
				log.WithError(err).Fatal("Failed to apply block to immutable database")
			}
		}
	}

	// Prepare storage procedures to update immutable database
	sps = append(sps, addBlock(bl))
	for k, v := range r.immutable.dirty.accounts {
		if v != nil {
			sps = append(sps, updateAccount(&v.Account))
		} else {
			sps = append(sps, deleteAccount(k))
		}
	}
	for k, v := range r.immutable.dirty.databases {
		if v != nil {
			sps = append(sps, updateShardChain(&v.SQLChainProfile))
		} else {
			sps = append(sps, deleteShardChain(k))
		}
	}
	for k, v := range r.immutable.dirty.provider {
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
		r.irre = irre
		// Apply irreversible blocks to immutable database
		r.immutable.commit()
		// Prune branches
		var (
			idx int
			brs = make([]*branch, 0, len(r.branches))
		)
		for i, b := range r.branches {
			if i == origin {
				// Current branch
				brs = append(brs, head)
				idx = len(brs) - 1
			} else if b.head.hasAncestor(irre) {
				brs = append(brs, b)
			}
		}
		// Replace current branches
		r.current = head
		r.currIdx = idx
		r.branches = brs
		// Clear packed transactions
		for _, b := range newIrres {
			for _, br := range r.branches {
				br.clearPackedTxs(b.block.Transactions)
			}
			for _, tx := range b.block.Transactions {
				delete(r.txPool, tx.Hash())
			}
		}
	}

	// Write to immutable database and update cache
	if err = store(st, sps, up); err != nil {
		r.immutable.clean()
	}
	return
}

func (r *rt) applyBlock(st xi.Storage, bl *types.BPBlock) (err error) {
	var (
		ok     bool
		br     *branch
		parent *blockNode
	)
	r.cacheMu.Lock()
	defer r.cacheMu.Unlock()

	for i, v := range r.branches {
		// Grow a branch
		if v.head.hash.IsEqual(&bl.SignedHeader.ParentHash) {
			if br, err = v.applyBlock(bl); err != nil {
				return
			}

			// Grow a branch while the current branch is not changed
			if br.head.count <= r.current.head.count {
				return store(st, []storageProcedure{addBlock(bl)}, func() { r.branches[i] = br })
			}

			// Switch branch or grow current branch
			return r.switchBranch(st, bl, i, br)
		}
		// Fork and create new branch
		if parent, ok = v.head.findNodeAfterCount(bl.SignedHeader.ParentHash, r.irre.count); ok {
			var head = newBlockNodeEx(bl, parent)
			if br, err = fork(r.irre, head, r.immutable, r.txPool); err != nil {
				return
			}
			return store(st,
				[]storageProcedure{addBlock(bl)},
				func() { r.branches = append(r.branches, br) },
			)
		}
	}

	return
}

func (r *rt) produceBlock(
	st xi.Storage, priv *asymmetric.PrivateKey) (out *types.BPBlock, err error,
) {
	var (
		bl *types.BPBlock
		br *branch
	)
	r.cacheMu.Lock()
	defer r.cacheMu.Unlock()

	// Try to produce new block
	if br, bl, err = r.current.produceBlock(); err != nil {
		return
	}
	if err = bl.PackAndSignBlock(priv); err != nil {
		return
	}
	if err = r.switchBranch(st, bl, r.currIdx, br); err != nil {
		return
	}
	out = bl
	return
}

// now returns the current coordinated chain time.
func (r *rt) now() time.Time {
	r.timeMutex.Lock()
	defer r.timeMutex.Unlock()
	// TODO(lambda): why does sqlchain not need UTC
	return time.Now().UTC().Add(r.offset)
}

func newRuntime(ctx context.Context, cfg *Config, accountAddress proto.AccountAddress) *rt {
	var index uint32
	for i, s := range cfg.Peers.Servers {
		if cfg.NodeID.IsEqual(&s) {
			index = uint32(i)
		}
	}
	var (
		cld, ccl = context.WithCancel(ctx)
		l        = float64(len(cfg.Peers.Servers))
		t        float64
		m        float64
	)
	if t = cfg.ComfirmThreshold; t <= 0.0 {
		t = float64(2) / 3.0
	}
	if m = math.Ceil(l*t + 1); m > l {
		m = l
	}
	return &rt{
		ctx:            cld,
		cancel:         ccl,
		wg:             &sync.WaitGroup{},
		chainInitTime:  cfg.Genesis.SignedHeader.Timestamp,
		accountAddress: accountAddress,
		server:         cfg.Server,
		bpNum:          uint32(len(cfg.Peers.Servers)),
		index:          index,
		period:         cfg.Period,
		tick:           cfg.Tick,
		peers:          cfg.Peers,
		nodeID:         cfg.NodeID,
		minComfirm:     uint32(m),
		nextTurn:       1,
		offset:         time.Duration(0),
	}
}

func (r *rt) startService(chain *Chain) {
	r.server.RegisterService(route.BlockProducerRPCName, &ChainRPCService{chain: chain})
}

// nextTick returns the current clock reading and the duration till the next turn. If duration
// is less or equal to 0, use the clock reading to run the next cycle - this avoids some problem
// caused by concurrently time synchronization.
func (r *rt) nextTick() (t time.Time, d time.Duration) {
	t = r.now()
	d = r.chainInitTime.Add(time.Duration(r.nextTurn) * r.period).Sub(t)

	if d > r.tick {
		d = r.tick
	}

	return
}

func (r *rt) isMyTurn() bool {
	r.stateMutex.Lock()
	defer r.stateMutex.Unlock()
	return r.nextTurn%r.bpNum == r.index
}

// setNextTurn prepares the runtime state for the next turn.
func (r *rt) setNextTurn() {
	r.stateMutex.Lock()
	defer r.stateMutex.Unlock()
	r.nextTurn++
}

func (r *rt) getIndexTotalServer() (uint32, uint32, proto.NodeID) {
	return r.index, r.bpNum, r.nodeID
}

func (r *rt) getPeerInfoString() string {
	index, bpNum, nodeID := r.getIndexTotalServer()
	return fmt.Sprintf("[%d/%d] %s", index, bpNum, nodeID)
}

// getHeightFromTime calculates the height with this sql-chain config of a given time reading.
func (r *rt) getHeightFromTime(t time.Time) uint32 {
	return uint32(t.Sub(r.chainInitTime) / r.period)
}

func (r *rt) getNextTurn() uint32 {
	r.stateMutex.Lock()
	defer r.stateMutex.Unlock()

	return r.nextTurn
}

func (r *rt) getPeers() *proto.Peers {
	r.peersMutex.Lock()
	defer r.peersMutex.Unlock()
	peers := r.peers.Clone()
	return &peers
}

func (r *rt) currentBranch() *branch {
	r.cacheMu.RLock()
	defer r.cacheMu.RUnlock()
	return r.current
}

func (r *rt) stop() {
	r.cancel()
	r.wg.Wait()
}

func (r *rt) goFunc(f func(ctx context.Context)) {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		f(r.ctx)
	}()
}
