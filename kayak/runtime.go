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
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"sync"
	"sync/atomic"
	"time"

	kt "github.com/CovenantSQL/CovenantSQL/kayak/types"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/pkg/errors"
)

const (
	// commit channel window size
	commitWindow = 1
	// prepare window
	trackerWindow = 10
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
	// rpc method for coordination requests.
	rpcMethod string
	// tracks the outgoing rpc requests.
	rpcTrackCh chan *rpcTracker

	//// Parameters
	// prepare threshold defines the minimum node count requirement for prepare operation.
	prepareThreshold float64
	// commit threshold defines the minimum node count requirement for commit operation.
	commitThreshold float64
	// prepare timeout defines the max allowed time for prepare operation.
	prepareTimeout time.Duration
	// commit timeout defines the max allowed time for commit operation.
	commitTimeout time.Duration
	// channel for awaiting commits.
	commitCh chan *commitReq

	/// Sub-routines management.
	started uint32
	stopCh  chan struct{}
	wg      sync.WaitGroup
}

// commitReq defines the commit operation input.
type commitReq struct {
	ctx    context.Context
	data   interface{}
	index  uint64
	log    *kt.Log
	result chan *commitResult
}

// commitResult defines the commit operation result.
type commitResult struct {
	result interface{}
	err    error
	rpc    *rpcTracker
}

// NewRuntime creates new kayak Runtime.
func NewRuntime(cfg *kt.RuntimeConfig) (rt *Runtime, err error) {
	peers := cfg.Peers

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
		serviceName: cfg.ServiceName,
		rpcMethod:   fmt.Sprintf("%v.%v", cfg.ServiceName, cfg.MethodName),
		rpcTrackCh:  make(chan *rpcTracker, trackerWindow),

		// commits related
		prepareThreshold: cfg.PrepareThreshold,
		prepareTimeout:   cfg.PrepareTimeout,
		commitThreshold:  cfg.CommitThreshold,
		commitTimeout:    cfg.CommitTimeout,
		commitCh:         make(chan *commitReq, commitWindow),

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
	// start rpc tracker collector
	// TODO():

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
	r.peersLock.RLock()
	defer r.peersLock.RUnlock()

	var tmStart, tmLeaderPrepare, tmFollowerPrepare, tmLeaderRollback, tmRollback, tmCommit time.Time

	defer func() {
		fields := log.Fields{
			"r": logIndex,
		}
		if !tmLeaderPrepare.Before(tmStart) {
			fields["lp"] = tmLeaderPrepare.Sub(tmStart)
		}
		if !tmFollowerPrepare.Before(tmLeaderPrepare) {
			fields["fp"] = tmFollowerPrepare.Sub(tmLeaderPrepare)
		}
		if !tmLeaderRollback.Before(tmFollowerPrepare) {
			fields["lr"] = tmLeaderRollback.Sub(tmFollowerPrepare)
		}
		if !tmRollback.Before(tmLeaderRollback) {
			fields["fr"] = tmRollback.Sub(tmLeaderRollback)
		}
		if !tmCommit.Before(tmFollowerPrepare) {
			fields["c"] = tmCommit.Sub(tmFollowerPrepare)
		}
		log.WithFields(fields).Debug("kayak leader apply")
	}()

	if r.role != proto.Leader {
		// not leader
		err = kt.ErrNotLeader
		return
	}

	tmStart = time.Now()

	// check prepare in leader
	if err = r.doCheck(req); err != nil {
		err = errors.Wrap(err, "leader verify log")
		return
	}

	// encode request
	var encBuf []byte
	if encBuf, err = r.sh.EncodePayload(req); err != nil {
		err = errors.Wrap(err, "encode kayak payload failed")
		return
	}

	// create prepare request
	var prepareLog *kt.Log
	if prepareLog, err = r.leaderLogPrepare(encBuf); err != nil {
		// serve error, leader could not write logs, change leader in block producer
		// TODO(): CHANGE LEADER
		return
	}

	tmLeaderPrepare = time.Now()

	// send prepare to all nodes
	prepareTracker := r.rpc(prepareLog, r.minPreparedFollowers)
	prepareCtx, prepareCtxCancelFunc := context.WithTimeout(ctx, r.prepareTimeout)
	defer prepareCtxCancelFunc()
	prepareErrors, prepareDone, _ := prepareTracker.get(prepareCtx)
	if !prepareDone {
		// timeout, rollback
		err = kt.ErrPrepareTimeout
		goto ROLLBACK
	}

	// collect errors
	if err = r.errorSummary(prepareErrors); err != nil {
		goto ROLLBACK
	}

	tmFollowerPrepare = time.Now()

	select {
	case cResult := <-r.commitResult(ctx, nil, prepareLog):
		if cResult != nil {
			logIndex = prepareLog.Index
			result = cResult.result
			err = cResult.err

			// wait until context deadline or commit done
			if cResult.rpc != nil {
				cResult.rpc.get(ctx)
			}
		} else {
			log.Fatal("IMPOSSIBLE BRANCH")
			select {
			case <-ctx.Done():
				err = errors.Wrap(ctx.Err(), "process commit timeout")
				goto ROLLBACK
			default:
			}
		}
	case <-ctx.Done():
		// pipeline commit timeout
		logIndex = prepareLog.Index
		err = errors.Wrap(ctx.Err(), "enqueue commit timeout")
		goto ROLLBACK
	}

	tmCommit = time.Now()

	return

ROLLBACK:
	// rollback local
	var rollbackLog *kt.Log
	if rollbackLog, err = r.leaderLogRollback(prepareLog.Index); err != nil {
		// serve error, construct rollback log failed, internal error
		// TODO(): CHANGE LEADER
		return
	}

	tmLeaderRollback = time.Now()

	// async send rollback to all nodes
	r.rpc(rollbackLog, 0)

	tmRollback = time.Now()

	return
}

// FollowerApply defines entry for follower node.
func (r *Runtime) FollowerApply(l *kt.Log) (err error) {
	if l == nil {
		err = errors.Wrap(kt.ErrInvalidLog, "log is nil")
		return
	}

	r.peersLock.RLock()
	defer r.peersLock.RUnlock()

	if r.role == proto.Leader {
		// not follower
		err = kt.ErrNotFollower
		return
	}

	// verify log structure
	switch l.Type {
	case kt.LogPrepare:
		err = r.followerPrepare(l)
	case kt.LogRollback:
		err = r.followerRollback(l)
	case kt.LogCommit:
		err = r.followerCommit(l)
	case kt.LogBarrier:
		// support barrier for log truncation and peer update
		fallthrough
	case kt.LogNoop:
		// do nothing
		err = r.followerNoop(l)
	}

	if err == nil {
		r.updateNextIndex(l)
	}

	return
}

// UpdatePeers defines entry for peers update logic.
func (r *Runtime) UpdatePeers(peers *proto.Peers) (err error) {
	r.peersLock.Lock()
	defer r.peersLock.Unlock()

	return
}

func (r *Runtime) leaderLogPrepare(data []byte) (*kt.Log, error) {
	// just write new log
	return r.newLog(kt.LogPrepare, data)
}

func (r *Runtime) leaderLogRollback(i uint64) (*kt.Log, error) {
	// just write new log
	return r.newLog(kt.LogRollback, r.uint64ToBytes(i))
}

func (r *Runtime) doCheck(req interface{}) (err error) {
	if err = r.sh.Check(req); err != nil {
		err = errors.Wrap(err, "verify log")
		return
	}

	return
}

func (r *Runtime) followerPrepare(l *kt.Log) (err error) {
	// decode
	var req interface{}
	if req, err = r.sh.DecodePayload(l.Data); err != nil {
		err = errors.Wrap(err, "decode kayak payload failed")
		return
	}

	if err = r.doCheck(req); err != nil {
		return
	}

	// write log
	if err = r.wal.Write(l); err != nil {
		err = errors.Wrap(err, "write follower prepare log failed")
		return
	}

	r.markPendingPrepare(l.Index)

	return
}

func (r *Runtime) followerRollback(l *kt.Log) (err error) {
	var prepareLog *kt.Log
	if _, prepareLog, err = r.getPrepareLog(l); err != nil || prepareLog == nil {
		err = errors.Wrap(err, "get original request in rollback failed")
		return
	}

	// check if prepare already processed
	if r.checkIfPrepareFinished(prepareLog.Index) {
		err = errors.Wrap(kt.ErrInvalidLog, "prepare request already processed")
		return
	}

	// write wal
	if err = r.wal.Write(l); err != nil {
		err = errors.Wrap(err, "write follower rollback log failed")
	}

	return
}

func (r *Runtime) followerCommit(l *kt.Log) (err error) {
	var prepareLog *kt.Log
	var lastCommit uint64
	if lastCommit, prepareLog, err = r.getPrepareLog(l); err != nil {
		err = errors.Wrap(err, "get original request in commit failed")
		return
	}

	// check if prepare already processed
	if r.checkIfPrepareFinished(prepareLog.Index) {
		err = errors.Wrap(kt.ErrInvalidLog, "prepare request already processed")
		return
	}

	myLastCommit := atomic.LoadUint64(&r.lastCommit)

	// check committed index
	if lastCommit < myLastCommit {
		// leader pushed a early index before commit
		log.WithFields(log.Fields{
			"head":     myLastCommit,
			"supplied": lastCommit,
		}).Warning("invalid last commit log")
		err = errors.Wrap(kt.ErrInvalidLog, "invalid last commit log index")
		return
	} else if lastCommit > myLastCommit {
		// last log does not committed yet
		// DO RECOVERY
		log.WithFields(log.Fields{
			"expected": lastCommit,
			"actual":   myLastCommit,
		}).Warning("DO RECOVERY, REQUIRED LAST COMMITTED DOES NOT COMMIT YET")
		err = errors.Wrap(kt.ErrNeedRecovery, "last commit does not received, need recovery")
		return
	}

	cResult := <-r.commitResult(context.Background(), l, prepareLog)
	if cResult != nil {
		err = cResult.err
	}

	return
}

func (r *Runtime) commitResult(ctx context.Context, commitLog *kt.Log, prepareLog *kt.Log) (res chan *commitResult) {
	// decode log and send to commit channel to process
	res = make(chan *commitResult, 1)

	if prepareLog == nil {
		res <- &commitResult{
			err: errors.Wrap(kt.ErrInvalidLog, "nil prepare log in commit"),
		}
		return
	}

	// decode prepare log
	var logReq interface{}
	var err error
	if logReq, err = r.sh.DecodePayload(prepareLog.Data); err != nil {
		res <- &commitResult{
			err: errors.Wrap(err, "decode log payload failed"),
		}
		return
	}

	req := &commitReq{
		ctx:    ctx,
		data:   logReq,
		index:  prepareLog.Index,
		result: res,
		log:    commitLog,
	}

	select {
	case <-ctx.Done():
	case r.commitCh <- req:
	}

	return
}

func (r *Runtime) commitCycle() {
	// TODO(): panic recovery
	for {
		var cReq *commitReq

		select {
		case <-r.stopCh:
			return
		case cReq = <-r.commitCh:
		}

		if cReq != nil {
			r.doCommit(cReq)
		}
	}
}

func (r *Runtime) doCommit(req *commitReq) {
	r.peersLock.RLock()
	defer r.peersLock.RUnlock()

	resp := &commitResult{}

	if r.role == proto.Leader {
		resp.rpc, resp.result, resp.err = r.leaderDoCommit(req)
	} else {
		resp.err = r.followerDoCommit(req)
	}

	req.result <- resp
}

func (r *Runtime) leaderDoCommit(req *commitReq) (tracker *rpcTracker, result interface{}, err error) {
	if req.log != nil {
		// mis-use follower commit for leader
		log.Fatal("INVALID EXISTING LOG FOR LEADER COMMIT")
		return
	}

	// create leader log
	var l *kt.Log
	var logData []byte

	logData = append(logData, r.uint64ToBytes(req.index)...)
	logData = append(logData, r.uint64ToBytes(atomic.LoadUint64(&r.lastCommit))...)

	if l, err = r.newLog(kt.LogCommit, logData); err != nil {
		// serve error, leader could not write log
		return
	}

	// not wrapping underlying handler commit error
	result, err = r.sh.Commit(req.data)

	if err == nil {
		// mark last commit
		atomic.StoreUint64(&r.lastCommit, l.Index)
	}

	// send commit
	commitCtx, commitCtxCancelFunc := context.WithTimeout(context.Background(), r.commitTimeout)
	defer commitCtxCancelFunc()
	tracker = r.rpc(l, r.minCommitFollowers)
	_, _, _ = tracker.get(commitCtx)

	// TODO(): text log for rpc errors

	// TODO(): mark uncommitted nodes and remove from peers

	return
}

func (r *Runtime) followerDoCommit(req *commitReq) (err error) {
	if req.log == nil {
		log.Fatal("NO LOG FOR FOLLOWER COMMIT")
		return
	}

	// write log first
	if err = r.wal.Write(req.log); err != nil {
		err = errors.Wrap(err, "write follower commit log failed")
		return
	}

	// do commit, not wrapping underlying handler commit error
	_, err = r.sh.Commit(req.data)

	if err == nil {
		atomic.StoreUint64(&r.lastCommit, req.log.Index)
	}

	return
}

func (r *Runtime) getPrepareLog(l *kt.Log) (lastCommitIndex uint64, pl *kt.Log, err error) {
	var prepareIndex uint64

	// decode prepare index
	if prepareIndex, err = r.bytesToUint64(l.Data); err != nil {
		err = errors.Wrap(err, "log does not contain valid prepare index")
		return
	}

	// decode commit index
	if len(l.Data) >= 16 {
		lastCommitIndex, _ = r.bytesToUint64(l.Data[8:])
	}

	pl, err = r.wal.Get(prepareIndex)

	return
}

func (r *Runtime) newLog(logType kt.LogType, data []byte) (l *kt.Log, err error) {
	// allocate index
	r.nextIndexLock.Lock()
	i := r.nextIndex
	r.nextIndex++
	r.nextIndexLock.Unlock()
	l = &kt.Log{
		LogHeader: kt.LogHeader{
			Index:    i,
			Type:     logType,
			Producer: r.nodeID,
		},
		Data: data,
	}

	// error write will be a fatal error, cause to node to fail fast
	if err = r.wal.Write(l); err != nil {
		log.Fatalf("WRITE LOG FAILED: %v", err)
	}

	return
}

func (r *Runtime) readLogs() (err error) {
	// load logs, only called during init
	var l *kt.Log

	for {
		if l, err = r.wal.Read(); err != nil && err != io.EOF {
			err = errors.Wrap(err, "load previous logs in wal failed")
			break
		} else if err == io.EOF {
			err = nil
			break
		}

		switch l.Type {
		case kt.LogPrepare:
			// record in pending prepares
			r.pendingPrepares[l.Index] = true
		case kt.LogCommit:
			// record last commit
			var lastCommit uint64
			var prepareLog *kt.Log
			if lastCommit, prepareLog, err = r.getPrepareLog(l); err != nil {
				err = errors.Wrap(err, "previous prepare does not exists, node need full recovery")
				break
			}
			if lastCommit != r.lastCommit {
				err = errors.Wrapf(err,
					"last commit record in wal mismatched (expected: %v, actual: %v)", r.lastCommit, lastCommit)
				break
			}
			if !r.pendingPrepares[prepareLog.Index] {
				err = errors.Wrap(kt.ErrInvalidLog, "previous prepare already committed/rollback")
				break
			}
			r.lastCommit = l.Index
			// resolve previous prepared
			delete(r.pendingPrepares, prepareLog.Index)
		case kt.LogRollback:
			var prepareLog *kt.Log
			if _, prepareLog, err = r.getPrepareLog(l); err != nil {
				err = errors.Wrap(err, "previous prepare doe snot exists, node need full recovery")
				return
			}
			if !r.pendingPrepares[prepareLog.Index] {
				err = errors.Wrap(kt.ErrInvalidLog, "previous prepare already committed/rollback")
				break
			}
			// resolve previous prepared
			delete(r.pendingPrepares, prepareLog.Index)
		default:
			err = errors.Wrapf(kt.ErrInvalidLog, "invalid log type: %v", l.Type)
			break
		}

		// record nextIndex
		r.updateNextIndex(l)
	}

	return
}

func (r *Runtime) updateNextIndex(l *kt.Log) {
	r.nextIndexLock.Lock()
	defer r.nextIndexLock.Unlock()

	if r.nextIndex < l.Index+1 {
		r.nextIndex = l.Index + 1
	}
}

func (r *Runtime) checkIfPrepareFinished(index uint64) (finished bool) {
	r.pendingPreparesLock.RLock()
	defer r.pendingPreparesLock.RUnlock()

	return !r.pendingPrepares[index]
}

func (r *Runtime) markPendingPrepare(index uint64) {
	r.pendingPreparesLock.Lock()
	defer r.pendingPreparesLock.Unlock()

	r.pendingPrepares[index] = true
}

func (r *Runtime) markPrepareFinished(index uint64) {
	r.pendingPreparesLock.Lock()
	defer r.pendingPreparesLock.Unlock()

	delete(r.pendingPrepares, index)
}

func (r *Runtime) errorSummary(errs map[proto.NodeID]error) error {
	failNodes := make([]proto.NodeID, 0, len(errs))

	for s, err := range errs {
		if err != nil {
			failNodes = append(failNodes, s)
		}
	}

	if len(failNodes) == 0 {
		return nil
	}

	return errors.Wrapf(kt.ErrPrepareFailed, "fail on nodes: %v", failNodes)
}

/// rpc related
func (r *Runtime) rpc(l *kt.Log, minCount int) (tracker *rpcTracker) {
	req := &kt.RPCRequest{
		Instance: r.instanceID,
		Log:      l,
	}

	tracker = newTracker(r, req, minCount)
	tracker.send()

	// TODO(): track this rpc

	// TODO(): log remote errors

	return
}

func (r *Runtime) getCaller(id proto.NodeID) Caller {
	var caller Caller = rpc.NewPersistentCaller(id)
	rawCaller, _ := r.callerMap.LoadOrStore(id, caller)
	return rawCaller.(Caller)
}

// SetCaller injects caller for test purpose.
func (r *Runtime) SetCaller(id proto.NodeID, c Caller) {
	r.callerMap.Store(id, c)
}

// RemoveCaller removes cached caller.
func (r *Runtime) RemoveCaller(id proto.NodeID) {
	r.callerMap.Delete(id)
}

func (r *Runtime) goFunc(f func()) {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		f()
	}()
}

/// utils
func (r *Runtime) uint64ToBytes(i uint64) (res []byte) {
	res = make([]byte, 8)
	binary.BigEndian.PutUint64(res, i)
	return
}

func (r *Runtime) bytesToUint64(b []byte) (uint64, error) {
	if len(b) < 8 {
		return 0, kt.ErrInvalidLog
	}
	return binary.BigEndian.Uint64(b), nil
}

//// future extensions, barrier, noop log placeholder etc.
func (r *Runtime) followerNoop(l *kt.Log) (err error) {
	return r.wal.Write(l)
}
