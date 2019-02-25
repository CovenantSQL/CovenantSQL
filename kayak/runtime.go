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

package kayak

import (
	"context"
	"math"
	"sync"
	"sync/atomic"
	"time"

	kt "github.com/CovenantSQL/CovenantSQL/kayak/types"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/CovenantSQL/CovenantSQL/utils/timer"
	"github.com/CovenantSQL/CovenantSQL/utils/trace"
	"github.com/pkg/errors"
)

const (
	// commit channel window size
	commitWindow = 0
	// missing log window
	missingLogWindow = 10
	// missing log concurrency
	missingLogConcurrency = 10
)

// Runtime defines the main kayak Runtime.
type Runtime struct {
	/// Indexes
	// index for next log.
	nextIndexLock sync.Mutex
	nextIndex     uint64
	// lastCommit, last commit log index
	lastCommit uint64
	// pendingPrepares, prepares needs to be committed/rollback
	pendingPrepares     map[uint64]bool
	pendingPreparesLock sync.RWMutex

	/// Runtime entities
	// current node id.
	nodeID proto.NodeID
	// instance identifies kayak in multi-instance environment
	// e.g. use database id for SQLChain scenario.
	instanceID string
	// wal defines the wal for kayak.
	wal kt.Wal
	// underlying handler
	sh kt.Handler

	/// Peers info
	// peers defines the server peers.
	peers *proto.Peers
	// cached role of current node in peers, calculated from peers info.
	role proto.ServerRole
	// cached followers in peers, calculated from peers info.
	followers []proto.NodeID
	// peers lock for peers update logic.
	peersLock sync.RWMutex
	// calculated min follower nodes for prepare.
	minPreparedFollowers int
	// calculated min follower nodes for commit.
	minCommitFollowers int

	/// RPC related
	// callerMap caches the caller for peering nodes.
	callerMap sync.Map // map[proto.NodeID]Caller
	// service name for mux service.
	serviceName string
	// rpc method for apply requests.
	applyRPCMethod string
	// rpc method for fetch requests.
	fetchRPCMethod string

	//// Parameters
	// prepare threshold defines the minimum node count requirement for prepare operation.
	prepareThreshold float64
	// commit threshold defines the minimum node count requirement for commit operation.
	commitThreshold float64
	// prepare timeout defines the max allowed time for prepare operation.
	prepareTimeout time.Duration
	// commit timeout defines the max allowed time for commit operation.
	commitTimeout time.Duration
	// log wait timeout to fetch missing logs.
	logWaitTimeout time.Duration
	// channel for awaiting commits.
	commitCh chan *commitReq
	// channel for missing log indexes.
	missingLogCh chan *waitItem
	waitLogMap   sync.Map // map[uint64]*waitItem

	/// Sub-routines management.
	started uint32
	stopCh  chan struct{}
	wg      sync.WaitGroup
}

// commitReq defines the commit operation input.
type commitReq struct {
	ctx        context.Context
	data       interface{}
	index      uint64
	lastCommit uint64
	log        *kt.Log
	result     *commitFuture
	tm         *timer.Timer
}

// commitResult defines the commit operation result.
type commitResult struct {
	index  uint64
	result interface{}
	err    error
	rpc    *rpcTracker
}

type commitFuture struct {
	ch chan *commitResult
}

func newCommitFuture() *commitFuture {
	return &commitFuture{
		ch: make(chan *commitResult, 1),
	}
}

func (f *commitFuture) Get(ctx context.Context) (cr *commitResult, err error) {
	if f == nil || f.ch == nil {
		err = errors.Wrap(ctx.Err(), "enqueue commit timeout")
		return
	}

	select {
	case <-ctx.Done():
		err = errors.Wrap(ctx.Err(), "get commit result timeout")
		return
	case cr = <-f.ch:
		return
	}
}

func (f *commitFuture) Set(cr *commitResult) {
	select {
	case f.ch <- cr:
	default:
	}
}

// NewRuntime creates new kayak Runtime.
func NewRuntime(cfg *kt.RuntimeConfig) (rt *Runtime, err error) {
	if cfg == nil {
		err = errors.Wrap(kt.ErrInvalidConfig, "nil config")
		return
	}

	peers := cfg.Peers

	if peers == nil {
		err = errors.Wrap(kt.ErrInvalidConfig, "nil peers")
		return
	}

	// verify peers
	if err = peers.Verify(); err != nil {
		err = errors.Wrap(err, "verify peers during kayak init failed")
		return
	}

	followers := make([]proto.NodeID, 0, len(peers.Servers))
	exists := false
	var role proto.ServerRole

	for _, v := range peers.Servers {
		if !v.IsEqual(&peers.Leader) {
			followers = append(followers, v)
		}

		if v.IsEqual(&cfg.NodeID) {
			exists = true
			if v.IsEqual(&peers.Leader) {
				role = proto.Leader
			} else {
				role = proto.Follower
			}
		}
	}

	if !exists {
		err = errors.Wrapf(kt.ErrNotInPeer, "node %v not in peers %v", cfg.NodeID, peers)
		return
	}

	// calculate fan-out count according to threshold and peers info
	minPreparedFollowers := int(math.Max(math.Ceil(cfg.PrepareThreshold*float64(len(peers.Servers))), 1) - 1)
	minCommitFollowers := int(math.Max(math.Ceil(cfg.CommitThreshold*float64(len(peers.Servers))), 1) - 1)

	rt = &Runtime{
		// indexes
		pendingPrepares: make(map[uint64]bool, commitWindow*2),

		// handler and logs
		sh:         cfg.Handler,
		wal:        cfg.Wal,
		instanceID: cfg.InstanceID,

		// peers
		peers:                cfg.Peers,
		nodeID:               cfg.NodeID,
		followers:            followers,
		role:                 role,
		minPreparedFollowers: minPreparedFollowers,
		minCommitFollowers:   minCommitFollowers,

		// rpc related
		serviceName:    cfg.ServiceName,
		applyRPCMethod: cfg.ServiceName + "." + cfg.ApplyMethodName,
		fetchRPCMethod: cfg.ServiceName + "." + cfg.FetchMethodName,

		// commits related
		prepareThreshold: cfg.PrepareThreshold,
		prepareTimeout:   cfg.PrepareTimeout,
		commitThreshold:  cfg.CommitThreshold,
		commitTimeout:    cfg.CommitTimeout,
		logWaitTimeout:   cfg.LogWaitTimeout,
		commitCh:         make(chan *commitReq, commitWindow),
		missingLogCh:     make(chan *waitItem, missingLogWindow),

		// stop coordinator
		stopCh: make(chan struct{}),
	}

	// read from pool to rebuild uncommitted log map
	if err = rt.readLogs(); err != nil {
		return
	}

	return
}

// Start starts the Runtime.
func (r *Runtime) Start() (err error) {
	if !atomic.CompareAndSwapUint32(&r.started, 0, 1) {
		return
	}

	// start commit cycle
	r.goFunc(r.commitCycle)
	// start missing log worker
	for i := 0; i != missingLogConcurrency; i++ {
		r.goFunc(r.missingLogCycle)
	}

	return
}

// Shutdown waits for the Runtime to stop.
func (r *Runtime) Shutdown() (err error) {
	if !atomic.CompareAndSwapUint32(&r.started, 1, 2) {
		return
	}

	select {
	case <-r.stopCh:
	default:
		close(r.stopCh)
	}
	r.wg.Wait()

	return
}

// Apply defines entry for Leader node.
func (r *Runtime) Apply(ctx context.Context, req interface{}) (result interface{}, logIndex uint64, err error) {
	if atomic.LoadUint32(&r.started) != 1 {
		err = kt.ErrStopped
		return
	}

	ctx, task := trace.NewTask(ctx, "Kayak.Apply")
	defer task.End()

	tm := timer.NewTimer()

	defer func() {
		log.WithField("r", logIndex).
			WithFields(tm.ToLogFields()).
			WithError(err).
			Debug("kayak leader apply")
	}()

	r.peersLock.RLock()
	defer r.peersLock.RUnlock()

	tm.Add("peers_lock")

	if r.role != proto.Leader {
		// not leader
		err = kt.ErrNotLeader
		return
	}

	// prepare
	prepareLog, err := r.doLeaderPrepare(ctx, tm, req)

	if prepareLog != nil {
		defer r.markPrepareFinished(ctx, prepareLog.Index)
	}

	if err == nil {
		// commit
		return r.doLeaderCommit(ctx, tm, prepareLog, req)
	}

	// rollback
	if prepareLog != nil {
		r.doLeaderRollback(ctx, tm, prepareLog)
	}

	return
}

// Fetch defines entry for missing log fetch.
func (r *Runtime) Fetch(ctx context.Context, index uint64) (l *kt.Log, err error) {
	if atomic.LoadUint32(&r.started) != 1 {
		err = kt.ErrStopped
		return
	}

	tm := timer.NewTimer()

	defer func() {
		log.WithField("l", index).
			WithFields(tm.ToLogFields()).
			WithError(err).
			Debug("kayak log fetch")
	}()

	r.peersLock.RLock()
	defer r.peersLock.RUnlock()

	tm.Add("peers_lock")

	if r.role != proto.Leader {
		// not leader
		err = kt.ErrNotLeader
		return
	}

	// wal get
	return r.wal.Get(index)
}

// FollowerApply defines entry for follower node.
func (r *Runtime) FollowerApply(l *kt.Log) (err error) {
	if l == nil {
		err = errors.Wrap(kt.ErrInvalidLog, "log is nil")
		return
	}
	if atomic.LoadUint32(&r.started) != 1 {
		err = kt.ErrStopped
		return
	}

	ctx, task := trace.NewTask(context.Background(), "Kayak.FollowerApply."+l.Type.String())
	defer task.End()

	tm := timer.NewTimer()

	defer func() {
		log.
			WithFields(log.Fields{
				"t": l.Type.String(),
				"i": l.Index,
			}).
			WithFields(tm.ToLogFields()).
			WithError(err).
			Debug("kayak follower apply")
	}()

	r.peersLock.RLock()
	defer r.peersLock.RUnlock()

	tm.Add("peers_lock")

	if r.role == proto.Leader {
		// not follower
		err = kt.ErrNotFollower
		return
	}

	// verify log structure
	switch l.Type {
	case kt.LogPrepare:
		err = r.followerPrepare(ctx, tm, l)
	case kt.LogRollback:
		err = r.followerRollback(ctx, tm, l)
	case kt.LogCommit:
		err = r.followerCommit(ctx, tm, l)
	}

	if err == nil {
		r.updateNextIndex(ctx, l)
		r.triggerLogAwaits(l.Index)
	}

	return
}

// UpdatePeers defines entry for peers update logic.
func (r *Runtime) UpdatePeers(peers *proto.Peers) (err error) {
	r.peersLock.Lock()
	defer r.peersLock.Unlock()

	return
}

func (r *Runtime) updateNextIndex(ctx context.Context, l *kt.Log) {
	defer trace.StartRegion(ctx, "updateNextIndex").End()

	r.nextIndexLock.Lock()
	defer r.nextIndexLock.Unlock()

	if r.nextIndex < l.Index+1 {
		r.nextIndex = l.Index + 1
	}
}

func (r *Runtime) checkIfPrepareFinished(ctx context.Context, index uint64) (finished bool) {
	defer trace.StartRegion(ctx, "checkIfPrepareFinished").End()

	r.pendingPreparesLock.RLock()
	defer r.pendingPreparesLock.RUnlock()

	return !r.pendingPrepares[index]
}

func (r *Runtime) markPendingPrepare(ctx context.Context, index uint64) {
	defer trace.StartRegion(ctx, "markPendingPrepare").End()

	r.pendingPreparesLock.Lock()
	defer r.pendingPreparesLock.Unlock()

	r.pendingPrepares[index] = true
}

func (r *Runtime) markPrepareFinished(ctx context.Context, index uint64) {
	defer trace.StartRegion(ctx, "markPrepareFinished").End()

	r.pendingPreparesLock.Lock()
	defer r.pendingPreparesLock.Unlock()

	delete(r.pendingPrepares, index)
}
