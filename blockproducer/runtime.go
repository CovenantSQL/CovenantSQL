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
	"github.com/pkg/errors"
)

// rt defines the runtime of main chain.
type rt struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     *sync.WaitGroup

	// The following fields are read-only in runtime.
	server      *rpc.Server
	address     proto.AccountAddress
	genesisTime time.Time
	period      time.Duration
	tick        time.Duration

	sync.RWMutex // protects following fields
	peers        *proto.Peers
	nodeID       proto.NodeID
	comfirms     uint32
	serversNum   uint32
	locSvIndex   uint32
	nextTurn     uint32
	offset       time.Duration
	lastIrre     *blockNode
	immutable    *metaState
	headIndex    int
	headBranch   *branch
	branches     []*branch
	txPool       map[hash.Hash]pi.Transaction
}

func newRuntime(
	ctx context.Context, cfg *Config, accountAddress proto.AccountAddress,
	irre *blockNode, heads []*blockNode, immutable *metaState, txPool map[hash.Hash]pi.Transaction,
) *rt {
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
		err      error
	)
	if t = cfg.ComfirmThreshold; t <= 0.0 {
		t = float64(2) / 3.0
	}
	if m = math.Ceil(l*t + 1); m > l {
		m = l
	}

	// Rebuild branches
	var (
		branches  []*branch
		br, head  *branch
		headIndex int
	)
	if len(heads) == 0 {
		log.Fatal("At least one branch head is needed")
	}
	for _, v := range heads {
		log.WithFields(log.Fields{
			"irre_hash":  irre.hash.Short(4),
			"irre_count": irre.count,
			"head_hash":  v.hash.Short(4),
			"head_count": v.count,
		}).Debug("Checking head")
		if v.hasAncestor(irre) {
			if br, err = fork(irre, v, immutable, txPool); err != nil {
				log.WithError(err).Fatal("Failed to rebuild branch")
			}
			branches = append(branches, br)
		}
	}
	for i, v := range branches {
		if head == nil || v.head.count > head.head.count {
			headIndex = i
			head = v
		}
	}

	return &rt{
		ctx:    cld,
		cancel: ccl,
		wg:     &sync.WaitGroup{},

		server:      cfg.Server,
		address:     accountAddress,
		genesisTime: cfg.Genesis.SignedHeader.Timestamp,
		period:      cfg.Period,
		tick:        cfg.Tick,

		peers:      cfg.Peers,
		nodeID:     cfg.NodeID,
		comfirms:   uint32(m),
		serversNum: uint32(len(cfg.Peers.Servers)),
		locSvIndex: index,
		nextTurn:   1,
		offset:     time.Duration(0),

		lastIrre:   irre,
		immutable:  immutable,
		headIndex:  headIndex,
		headBranch: head,
		branches:   branches,
		txPool:     txPool,
	}
}

func (r *rt) addTx(st xi.Storage, tx pi.Transaction) (err error) {
	var k = tx.Hash()
	r.Lock()
	defer r.Unlock()
	if _, ok := r.txPool[k]; ok {
		err = ErrExistedTx
		return
	}

	return store(st, []storageProcedure{addTx(tx)}, func() {
		r.txPool[k] = tx
		for _, v := range r.branches {
			v.addTx(tx)
		}
	})
}

func (r *rt) nextNonce(addr proto.AccountAddress) (n pi.AccountNonce, err error) {
	r.RLock()
	defer r.RUnlock()
	return r.headBranch.preview.nextNonce(addr)
}

func (r *rt) loadAccountCovenantBalance(addr proto.AccountAddress) (balance uint64, ok bool) {
	r.RLock()
	defer r.RUnlock()
	return r.immutable.loadAccountCovenantBalance(addr)
}

func (r *rt) loadAccountStableBalance(addr proto.AccountAddress) (balance uint64, ok bool) {
	r.RLock()
	defer r.RUnlock()
	return r.immutable.loadAccountStableBalance(addr)
}

func (r *rt) switchBranch(st xi.Storage, bl *types.BPBlock, origin int, head *branch) (err error) {
	var (
		irre     *blockNode
		newIrres []*blockNode
		sps      []storageProcedure
		up       storageCallback
		height   = r.height(bl.Timestamp())
	)

	// Find new irreversible blocks
	//
	// NOTE(leventeliu):
	// May have multiple new irreversible blocks here if peer list shrinks. May also have
	// no new irreversible block at all if peer list expands.
	irre = head.head.lastIrreversible(r.comfirms)
	newIrres = irre.fetchNodeList(r.lastIrre.count)

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
	sps = append(sps, addBlock(height, bl))
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
		r.lastIrre = irre
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
		r.headBranch = head
		r.headIndex = idx
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

func (r *rt) log() {
	r.RLock()
	defer r.RUnlock()
	for i, v := range r.branches {
		var buff string
		if i == r.headIndex {
			buff += "[head] "
		} else {
			buff += fmt.Sprintf("[%04d] ", i)
		}
		buff += v.sprint(r.lastIrre.count)
		log.WithFields(log.Fields{
			"branch": buff,
		}).Debug("Runtime state")
	}
	return
}

func (r *rt) applyBlock(st xi.Storage, bl *types.BPBlock) (err error) {
	var (
		ok     bool
		br     *branch
		parent *blockNode
		head   *blockNode
		height = r.height(bl.Timestamp())
	)

	defer r.log()
	r.Lock()
	defer r.Unlock()

	for i, v := range r.branches {
		// Grow a branch
		if v.head.hash.IsEqual(&bl.SignedHeader.ParentHash) {
			head = newBlockNode(height, bl, v.head)
			if br, err = v.applyBlock(head); err != nil {
				return
			}
			// Grow a branch while the current branch is not changed
			if br.head.count <= r.headBranch.head.count {
				return store(st,
					[]storageProcedure{addBlock(height, bl)},
					func() { r.branches[i] = br },
				)
			}
			// Switch branch or grow current branch
			return r.switchBranch(st, bl, i, br)
		}
		// Fork and create new branch
		if parent, ok = v.head.canForkFrom(bl.SignedHeader.ParentHash, r.lastIrre.count); ok {
			head = newBlockNode(height, bl, parent)
			if br, err = fork(r.lastIrre, head, r.immutable, r.txPool); err != nil {
				return
			}
			return store(st,
				[]storageProcedure{addBlock(height, bl)},
				func() { r.branches = append(r.branches, br) },
			)
		}
	}

	return
}

func (r *rt) produceBlock(
	st xi.Storage, now time.Time, priv *asymmetric.PrivateKey) (out *types.BPBlock, err error,
) {
	var (
		bl   *types.BPBlock
		br   *branch
		ierr error
	)

	defer r.log()
	r.Lock()
	defer r.Unlock()

	// Try to produce new block
	if br, bl, ierr = r.headBranch.produceBlock(
		r.height(now), now, r.address, priv,
	); ierr != nil {
		err = errors.Wrapf(ierr, "failed to produce block at head %s",
			r.headBranch.head.hash.Short(4))
		return
	}
	if ierr = r.switchBranch(st, bl, r.headIndex, br); ierr != nil {
		err = errors.Wrapf(ierr, "failed to switch branch #%d:%s",
			r.headIndex, r.headBranch.head.hash.Short(4))
		return
	}
	out = bl
	return
}

// now returns the current coordinated chain time.
func (r *rt) now() time.Time {
	r.RLock()
	defer r.RUnlock()
	return time.Now().UTC().Add(r.offset)
}

func (r *rt) startService(chain *Chain) {
	r.server.RegisterService(route.BlockProducerRPCName, &ChainRPCService{chain: chain})
}

// nextTick returns the current clock reading and the duration till the next turn. If duration
// is less or equal to 0, use the clock reading to run the next cycle - this avoids some problem
// caused by concurrent time synchronization.
func (r *rt) nextTick() (t time.Time, d time.Duration) {
	var nt uint32
	nt, t = func() (nt uint32, t time.Time) {
		r.RLock()
		defer r.RUnlock()
		nt = r.nextTurn
		t = time.Now().UTC().Add(r.offset)
		return
	}()
	d = r.genesisTime.Add(time.Duration(nt) * r.period).Sub(t)
	if d > r.tick {
		d = r.tick
	}
	return
}

func (r *rt) isMyTurn() bool {
	r.RLock()
	defer r.RUnlock()
	return r.nextTurn%r.serversNum == r.locSvIndex
}

// setNextTurn prepares the runtime state for the next turn.
func (r *rt) setNextTurn() {
	r.Lock()
	defer r.Unlock()
	r.nextTurn++
}

func (r *rt) peerInfo() string {
	var index, bpNum, nodeID = func() (uint32, uint32, proto.NodeID) {
		r.RLock()
		defer r.RUnlock()
		return r.locSvIndex, r.serversNum, r.nodeID
	}()
	return fmt.Sprintf("[%d/%d] %s", index, bpNum, nodeID)
}

// height calculates the height with this sql-chain config of a given time reading.
func (r *rt) height(t time.Time) uint32 {
	return uint32(t.Sub(r.genesisTime) / r.period)
}

func (r *rt) getNextTurn() uint32 {
	r.RLock()
	defer r.RUnlock()
	return r.nextTurn
}

func (r *rt) getPeers() *proto.Peers {
	r.RLock()
	defer r.RUnlock()
	var peers = r.peers.Clone()
	return &peers
}

func (r *rt) head() *blockNode {
	r.RLock()
	defer r.RUnlock()
	return r.headBranch.head
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
