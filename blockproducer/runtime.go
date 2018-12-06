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
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	"github.com/CovenantSQL/CovenantSQL/types"
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
	// head is the current head of the best chain.
	head          *Obsolete
	optionalHeads []*Obsolete

	// timeMutex protects following time-relative fields.
	timeMutex sync.Mutex
	// offset is the time difference calculated by: coodinatedChainTime - time.Now().
	//
	// TODO(leventeliu): update offset in ping cycle.
	offset time.Duration

	// Cached state
	cacheMu   sync.RWMutex
	irre      *blockNode
	immutable *view
	current   *branch
	optional  []*branch
	txPool    map[hash.Hash]pi.Transaction
}

func (r *rt) addTx(tx pi.Transaction) {
	var k = tx.Hash()
	r.cacheMu.Lock()
	defer r.cacheMu.Unlock()
	if _, ok := r.txPool[k]; !ok {
		r.txPool[k] = tx
	}
	r.current.addTx(tx)
	for _, v := range r.optional {
		v.addTx(tx)
	}
}

func (r *rt) produceBlock(st xi.Storage) (err error) {
	var (
		br       *branch
		bl       *types.BPBlock
		irre     *blockNode
		newIrres []*blockNode
	)
	r.cacheMu.Lock()
	r.cacheMu.Unlock()

	// Try to produce new block
	if br, bl, err = r.current.produceBlock(); err != nil {
		return
	}

	// Find new irreversible blocks
	//
	// NOTE(leventeliu):
	// May have multiple new irreversible blocks here if peer list shrinks.
	// May also have no new irreversible block at all if peer list expands.
	irre = br.head.lastIrreversible(r.minComfirm)
	for n := irre; n.count > r.irre.count; n = n.parent {
		newIrres = append(newIrres, n)
	}

	var (
		sps []storageProcedure
		up  storageCallback
	)

	// Prepare storage procedures to update immutable
	sps = append(sps, addBlock(bl))
	// Note that block nodes are pushed in reverse order and should be applied in ascending order
	for i := len(newIrres) - 1; i >= 0; i-- {
		var v = newIrres[i]
		sps = append(
			sps,
			updateImmutable(v.block.Transactions),
			deleteTxs(v.block.Transactions),
		)
	}
	sps = append(sps, updateIrreversible(irre.hash))

	// Prepare callback to update cache
	up = func() {
		// Replace current branch
		r.current = br
		// Prune branches
		var brs = make([]*branch, 0, len(r.optional))
		for _, v := range r.optional {
			if v.head.hasAncestor(irre) {
				brs = append(brs, v)
			}
		}
		r.optional = brs
		// Update last irreversible block
		r.irre = irre
	}

	// Write to immutable database and update cache
	return store(st, sps, up)
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
		head:           &Obsolete{},
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

func (r *rt) getHead() *Obsolete {
	r.stateMutex.Lock()
	defer r.stateMutex.Unlock()
	return r.head
}

func (r *rt) setHead(head *Obsolete) {
	r.stateMutex.Lock()
	defer r.stateMutex.Unlock()
	r.head = head
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
