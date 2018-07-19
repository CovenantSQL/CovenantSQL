/*
 * Copyright 2018 The ThunderDB Authors.
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
	"fmt"
	"sync"
	"time"

	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
	"gitlab.com/thunderdb/ThunderDB/proto"
	"gitlab.com/thunderdb/ThunderDB/twopc"
	"gitlab.com/thunderdb/ThunderDB/utils/log"
)

var (
	// current term stored in local meta
	keyCurrentTerm = []byte("CurrentTerm")

	// committed index store in local meta
	keyCommittedIndex = []byte("CommittedIndex")
)

// TwoPCConfig is a RuntimeConfig implementation organizing two phase commit mutation.
type TwoPCConfig struct {
	RuntimeConfig

	// Storage is the underlying twopc Storage
	Storage twopc.Worker
}

// TwoPCRunner is a Runner implementation organizing two phase commit mutation.
type TwoPCRunner struct {
	config      *TwoPCConfig
	peers       *Peers
	logStore    LogStore
	stableStore StableStore
	transport   Transport

	// Current term/log state
	currentTerm  uint64
	lastLogIndex uint64
	lastLogTerm  uint64
	lastLogHash  *hash.Hash

	// Server role
	leader *Server
	role   proto.ServerRole

	// Shutdown channel to exit, protected to prevent concurrent exits
	shutdown     bool
	shutdownCh   chan struct{}
	shutdownLock sync.Mutex

	// Lock/events
	processLock     sync.Mutex
	processReq      chan []byte
	processRes      chan error
	updatePeersLock sync.Mutex
	updatePeersReq  chan *Peers
	updatePeersRes  chan error

	currentState   ServerState
	stateLock      sync.Mutex
	currentContext context.Context

	// Tracks running goroutines
	routinesGroup sync.WaitGroup
}

// TwoPCWorkerWrapper wraps remote runner as worker.
type TwoPCWorkerWrapper struct {
	runner *TwoPCRunner
	nodeID proto.NodeID
}

// NewTwoPCRunner create a two pc runner.
func NewTwoPCRunner() *TwoPCRunner {
	return &TwoPCRunner{
		shutdownCh:     make(chan struct{}),
		processReq:     make(chan []byte),
		processRes:     make(chan error),
		updatePeersReq: make(chan *Peers),
		updatePeersRes: make(chan error),
	}
}

// GetRuntimeConfig implements Config.GetRuntimeConfig.
func (tpc *TwoPCConfig) GetRuntimeConfig() *RuntimeConfig {
	return &tpc.RuntimeConfig
}

// Init implements Runner.Init.
func (r *TwoPCRunner) Init(config Config, peers *Peers, logs LogStore, stable StableStore, transport Transport) error {
	if _, ok := config.(*TwoPCConfig); !ok {
		return ErrInvalidConfig
	}

	if peers == nil || logs == nil || stable == nil || transport == nil {
		return ErrInvalidConfig
	}

	r.config = config.(*TwoPCConfig)
	r.peers = peers
	r.logStore = logs
	r.stableStore = stable
	r.transport = transport
	r.setState(Idle)

	// restore from log/stable store
	if err := r.tryRestore(); err != nil {
		return err
	}

	// set init peers and update term
	if err := r.initState(); err != nil {
		return err
	}

	r.goFunc(r.run)

	return nil
}

func (r *TwoPCRunner) tryRestore() error {
	// Init term, committedIndex, storage
	var err error
	var lastTerm uint64

	lastTerm, err = r.stableStore.GetUint64(keyCurrentTerm)
	if err != nil && err != ErrKeyNotFound {
		return fmt.Errorf("get last term failed: %s", err.Error())
	}

	if r.peers.Term < lastTerm {
		// invalid config, term older than current context
		// suggest rebuild local config
		return ErrInvalidConfig
	}

	var lastCommitted uint64
	lastCommitted, err = r.stableStore.GetUint64(keyCommittedIndex)
	if err != nil && err != ErrKeyNotFound {
		return fmt.Errorf("last committed index not found: %s", err.Error())
	}

	var lastCommittedLog Log
	if lastCommitted > 0 {
		if err = r.logStore.GetLog(lastCommitted, &lastCommittedLog); err != nil {
			return fmt.Errorf("failed to get last log at index %d: %s", lastCommitted, err.Error())
		}
	}

	// committed index term check
	if r.peers.Term < lastCommittedLog.Term {
		return fmt.Errorf("invalid last committed log term, peers: %d, local committed: %d",
			r.peers.Term, lastCommittedLog.Term)
	}

	// assert index related log validation
	if lastCommitted != lastCommittedLog.Index {
		// invalid log
		return fmt.Errorf("invalid last committed log index, index: %d, log: %d",
			lastCommitted, lastCommittedLog.Index)
	}

	// get last index
	var lastIndex uint64
	lastIndex, err = r.logStore.LastIndex()
	if err != nil {
		return fmt.Errorf("failed to get last index: %s", err.Error())
	}

	if lastIndex > lastCommitted {
		// uncommitted log found, print warning
		log.Warningf("truncating local uncommitted log, uncommitted: %d, committed: %d",
			lastIndex, lastCommitted)

		// truncate local uncommitted logs
		r.logStore.DeleteRange(lastCommitted+1, lastIndex)
	}

	if err = r.reValidateLocalLogs(); err != nil {
		return err
	}

	if err = r.restoreUnderlying(); err != nil {
		return err
	}

	r.currentTerm = r.peers.Term
	r.lastLogTerm = lastCommittedLog.Term
	r.lastLogIndex = lastCommitted
	if lastCommittedLog.Index != 0 {
		r.lastLogHash = &lastCommittedLog.Hash
	} else {
		r.lastLogHash = nil
	}

	return nil
}

func (r *TwoPCRunner) initState() error {
	if !r.peers.Verify() {
		return ErrInvalidConfig
	}

	// set leader and node role
	r.leader = r.peers.Leader

	for _, s := range r.peers.Servers {
		if s.ID == r.config.LocalID {
			r.role = s.Role
			break
		}
	}

	// update peers term
	return r.stableStore.SetUint64(keyCurrentTerm, r.peers.Term)
}

func (r *TwoPCRunner) reValidateLocalLogs() error {
	// TODO(xq262144): maybe re-validating local log hashes
	return nil
}

func (r *TwoPCRunner) restoreUnderlying() error {
	// TODO(xq262144): restore underlying from snapshot and replaying local logs
	return nil
}

// UpdatePeers implements Runner.UpdatePeers.
func (r *TwoPCRunner) UpdatePeers(peers *Peers) error {
	r.updatePeersLock.Lock()
	defer r.updatePeersLock.Unlock()

	// wait for transaction completion
	// TODO(xq262144): support transaction timeout

	if peers.Term == r.peers.Term {
		// same term, ignore
		return nil
	}

	if peers.Term < r.peers.Term {
		// lower term, maybe spoofing request
		return ErrInvalidConfig
	}

	// validate peers structure
	if !peers.Verify() {
		return ErrInvalidConfig
	}

	r.updatePeersReq <- peers
	return <-r.updatePeersRes
}

// Apply implements Runner.Apply.
func (r *TwoPCRunner) Apply(data []byte) error {
	r.processLock.Lock()
	defer r.processLock.Unlock()

	// check leader privilege
	if r.role != proto.Leader {
		return ErrNotLeader
	}

	r.processReq <- data
	return <-r.processRes
}

// Shutdown implements Runner.Shutdown.
func (r *TwoPCRunner) Shutdown(wait bool) error {
	r.shutdownLock.Lock()
	defer r.shutdownLock.Unlock()

	if !r.shutdown {
		close(r.shutdownCh)
		r.shutdown = true
		r.setState(Shutdown)
		if wait {
			r.routinesGroup.Wait()
		}
	}

	return nil
}

func (r *TwoPCRunner) run() {
	for {
		select {
		case <-r.shutdownCh:
			// TODO(xq262144): cleanup logic
			return
		case data := <-r.processReq:
			r.processRes <- r.processNewLog(data)
		case request := <-r.transport.Process():
			r.processRequest(request)
			// TODO(xq262144): support timeout logic for auto rollback prepared transaction on leader change
		case peersUpdate := <-r.safeForPeersUpdate():
			r.processPeersUpdate(peersUpdate)
		}
	}
}

func (r *TwoPCRunner) safeForPeersUpdate() chan *Peers {
	if r.getState() == Idle {
		return r.updatePeersReq
	}

	return nil
}

func (r *TwoPCRunner) processNewLog(data []byte) (err error) {
	// build Log
	l := &Log{
		Index:    r.lastLogIndex + 1,
		Term:     r.currentTerm,
		Data:     data,
		LastHash: r.lastLogHash,
	}

	// compute hash
	l.ComputeHash()

	localPrepare := func(ctx context.Context) error {
		// prepare local prepare node
		if err := r.config.Storage.Prepare(ctx, l.Data); err != nil {
			return err
		}

		// write log to storage
		return r.logStore.StoreLog(l)
	}

	localRollback := func(ctx context.Context) error {
		// prepare local rollback node
		r.logStore.DeleteRange(r.lastLogIndex+1, l.Index)
		return r.config.Storage.Rollback(ctx, l.Data)
	}

	localCommit := func(ctx context.Context) (err error) {
		err = r.config.Storage.Commit(ctx, l.Data)

		r.stableStore.SetUint64(keyCommittedIndex, l.Index)
		r.lastLogHash = &l.Hash
		r.lastLogIndex = l.Index
		r.lastLogTerm = l.Term

		return
	}

	// build 2PC workers
	if len(r.peers.Servers) > 1 {
		nodes := make([]twopc.Worker, 0, len(r.peers.Servers)-1)

		for _, s := range r.peers.Servers {
			if s.ID != r.config.LocalID {
				nodes = append(nodes, NewTwoPCWorkerWrapper(r, s.ID))
			}
		}

		// start coordination
		c := twopc.NewCoordinator(twopc.NewOptionsWithCallback(
			r.config.ProcessTimeout,
			nil,
			localPrepare,  // after all remote nodes prepared
			localRollback, // before all remote nodes rollback
			localCommit,   // after all remote nodes commit
		))

		err = c.Put(nodes, l)
	} else {
		// single node short cut
		// init context
		ctx, cancel := context.WithTimeout(context.Background(), r.config.ProcessTimeout)
		defer cancel()

		if err := localPrepare(ctx); err != nil {
			localRollback(ctx)
			return err
		}

		// Commit myself
		// return commit err but still commit
		err = localCommit(ctx)
	}

	return
}

func (r *TwoPCRunner) setState(state ServerState) {
	r.stateLock.Lock()
	defer r.stateLock.Unlock()
	r.currentState = state
}

func (r *TwoPCRunner) getState() ServerState {
	r.stateLock.Lock()
	defer r.stateLock.Unlock()
	return r.currentState
}

func (r *TwoPCRunner) processRequest(req Request) {
	// verify call from leader
	if err := r.verifyLeader(req); err != nil {
		req.SendResponse(nil, err)
		return
	}

	switch req.GetMethod() {
	case "Prepare":
		r.processPrepare(req)
	case "Commit":
		r.processCommit(req)
	case "Rollback":
		r.processRollback(req)
	default:
		req.SendResponse(nil, ErrInvalidRequest)
	}
}

func (r *TwoPCRunner) processPeersUpdate(peersUpdate *Peers) {
	// update peers
	var err error
	if err = r.stableStore.SetUint64(keyCurrentTerm, peersUpdate.Term); err == nil {
		r.peers = peersUpdate
		r.currentTerm = peersUpdate.Term

		// change role
		r.leader = r.peers.Leader

		notFound := true

		for _, s := range r.peers.Servers {
			if s.ID == r.config.LocalID {
				r.role = s.Role
				notFound = false
				break
			}
		}

		if notFound {
			// shutdown
			r.Shutdown(false)
		}
	}

	r.updatePeersRes <- err
}

func (r *TwoPCRunner) verifyLeader(req Request) error {
	// TODO(xq262144): verify call from current leader or from new leader containing new peers info
	if req.GetPeerNodeID() != r.peers.Leader.ID {
		// not our leader
		return ErrInvalidRequest
	}

	return nil
}

func (r *TwoPCRunner) verifyLog(req Request) (log *Log, err error) {
	log = req.GetLog()

	if log == nil {
		err = ErrInvalidLog
		return
	}

	if !log.VerifyHash() {
		err = ErrInvalidLog
		return
	}

	return
}

func (r *TwoPCRunner) processPrepare(req Request) {
	req.SendResponse(nil, func() (err error) {
		// already in transaction, try abort previous
		if r.getState() != Idle {
			// TODO(xq262144): has running transaction
			// TODO(xq262144): abort previous or failed current
		}

		// init context
		r.currentContext, _ = context.WithTimeout(context.Background(), r.config.ProcessTimeout)

		// get log
		var l *Log
		if l, err = r.verifyLog(req); err != nil {
			return
		}

		// check log index existence
		var lastIndex uint64
		if lastIndex, err = r.logStore.LastIndex(); err != nil || lastIndex >= l.Index {
			// already prepared or failed
			return
		}

		// check prepare hash with last log hash
		if l.LastHash != nil && lastIndex == 0 {
			// invalid
			return ErrInvalidLog
		}

		if lastIndex > 0 {
			var lastLog Log
			if err = r.logStore.GetLog(lastIndex, &lastLog); err != nil {
				return
			}

			if !l.LastHash.IsEqual(&lastLog.Hash) {
				return ErrInvalidLog
			}
		}

		// prepare on storage
		if err = r.config.Storage.Prepare(r.currentContext, l.Data); err != nil {
			return
		}

		// write log to storage
		if err = r.logStore.StoreLog(l); err != nil {
			return
		}

		// set state to prepared
		r.setState(Prepared)

		return nil
	}())
}

func (r *TwoPCRunner) processCommit(req Request) {
	// commit log
	req.SendResponse(nil, func() (err error) {
		// TODO(xq262144): check current running transaction index
		if r.getState() != Prepared {
			// not prepared, failed directly
			return ErrInvalidRequest
		}

		// get log
		var l *Log
		if l, err = r.verifyLog(req); err != nil {
			return
		}

		var lastIndex uint64
		if lastIndex, err = r.logStore.LastIndex(); err != nil {
			return
		} else if lastIndex < l.Index {
			// not logged, need re-prepare
			return ErrInvalidLog
		}

		if r.lastLogIndex+1 != l.Index {
			// not at the head of the commit position
			return ErrInvalidLog
		}

		// get log
		var lastLog Log
		if err = r.logStore.GetLog(l.Index, &lastLog); err != nil {
			return
		}

		// commit on storage
		// return err but still commit local index
		err = r.config.Storage.Commit(r.currentContext, l.Data)

		// commit log
		r.stableStore.SetUint64(keyCommittedIndex, l.Index)
		r.lastLogHash = &lastLog.Hash
		r.lastLogIndex = lastLog.Index
		r.lastLogTerm = lastLog.Term

		// set state to idle
		r.setState(Idle)

		return
	}())
}

func (r *TwoPCRunner) processRollback(req Request) {
	// rollback log
	req.SendResponse(nil, func() (err error) {
		// TODO(xq262144): check current running transaction index
		if r.getState() != Prepared {
			// not prepared, failed directly
			return ErrInvalidRequest
		}

		// get log
		var l *Log
		if l, err = r.verifyLog(req); err != nil {
			return
		}

		var lastIndex uint64
		if lastIndex, err = r.logStore.LastIndex(); err != nil {
			return
		} else if lastIndex < l.Index {
			// not logged, no rollback required, maybe previous initiated rollback
			return
		}

		if r.lastLogIndex+1 != l.Index {
			// not at the head of the commit position
			return ErrInvalidLog
		}

		// get log
		var lastLog Log
		if err = r.logStore.GetLog(l.Index, &lastLog); err != nil {
			return
		}

		// rollback on storage
		if err = r.config.Storage.Rollback(r.currentContext, l.Data); err != nil {
			return
		}

		// rewind log, can be failed, since committedIndex is not updated
		r.logStore.DeleteRange(r.lastLogIndex+1, l.Index)

		// set state to idle
		r.setState(Idle)

		return
	}())
}

// Start a goroutine and properly handle the race between a routine
// starting and incrementing, and exiting and decrementing.
func (r *TwoPCRunner) goFunc(f func()) {
	r.routinesGroup.Add(1)
	go func() {
		defer r.routinesGroup.Done()
		f()
	}()
}

// NewTwoPCWorkerWrapper returns a wrapper for remote worker.
func NewTwoPCWorkerWrapper(runner *TwoPCRunner, nodeID proto.NodeID) *TwoPCWorkerWrapper {
	return &TwoPCWorkerWrapper{
		nodeID: nodeID,
		runner: runner,
	}
}

// Prepare implements twopc.Worker.Prepare.
func (tpww *TwoPCWorkerWrapper) Prepare(ctx context.Context, wb twopc.WriteBatch) error {
	// extract log
	l, ok := wb.(*Log)
	if !ok {
		return ErrInvalidLog
	}

	return tpww.callRemote(ctx, "Prepare", l)
}

// Commit implements twopc.Worker.Commit.
func (tpww *TwoPCWorkerWrapper) Commit(ctx context.Context, wb twopc.WriteBatch) error {
	// extract log
	l, ok := wb.(*Log)
	if !ok {
		return ErrInvalidLog
	}

	return tpww.callRemote(ctx, "Commit", l)
}

// Rollback implements twopc.Worker.Rollback.
func (tpww *TwoPCWorkerWrapper) Rollback(ctx context.Context, wb twopc.WriteBatch) error {
	// extract log
	l, ok := wb.(*Log)
	if !ok {
		return ErrInvalidLog
	}

	return tpww.callRemote(ctx, "Rollback", l)
}

func (tpww *TwoPCWorkerWrapper) callRemote(ctx context.Context, method string, log *Log) (err error) {
	// TODO(xq262144): handle retry
	_, err = tpww.runner.transport.Request(ctx, tpww.nodeID, method, log)
	return
}

func nestedTimeoutCtx(ctx context.Context, timeout time.Duration, process func(context.Context) error) error {
	nestedCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return process(nestedCtx)
}

var (
	_ Config       = &TwoPCConfig{}
	_ Runner       = &TwoPCRunner{}
	_ twopc.Worker = &TwoPCWorkerWrapper{}
)
