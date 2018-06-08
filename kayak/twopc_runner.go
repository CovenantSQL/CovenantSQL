/*
 * Copyright 2018 The ThunderDB Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the “License”);
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an “AS IS” BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package kayak

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/thunderdb/ThunderDB/crypto/hash"
	"github.com/thunderdb/ThunderDB/twopc"
)

var (
	// current term stored in local meta
	keyCurrentTerm = []byte("CurrentTerm")

	// committed index store in local meta
	keyCommittedIndex = []byte("CommittedIndex")

	// ErrInvalidRequest indicate inconsistent state
	ErrInvalidRequest = errors.New("invalid request")
)

// TwoPCConfig is a RuntimeConfig implementation organizing two phase commit mutation
type TwoPCConfig struct {
	RuntimeConfig

	// LogCodec is the underlying storage log codec
	LogCodec TwoPCLogCodec

	// Storage is the underlying twopc Storage
	Storage twopc.Worker

	// PrepareTimeout
	PrepareTimeout time.Duration

	// CommitTimeout
	CommitTimeout time.Duration

	// RollbackTimeout
	RollbackTimeout time.Duration
}

// TwoPCRunner is a Runner implementation organizing two phase commit mutation
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
	role   ServerRole

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

	currentState ServerState
	stateLock    sync.Mutex

	// Tracks running goroutines
	routinesGroup sync.WaitGroup
}

// TwoPCWorkerWrapper wraps remote runner as worker
type TwoPCWorkerWrapper struct {
	runner   *TwoPCRunner
	serverID ServerID
}

// TwoPCLogCodec is the log data encode/decoder
type TwoPCLogCodec interface {
	// Encode log to bytes
	Encode(interface{}) ([]byte, error)

	// Decode logs to bytes
	Decode([]byte, interface{}) error
}

// NewTwoPCRunner create a two pc runner
func NewTwoPCRunner() *TwoPCRunner {
	return &TwoPCRunner{
		shutdownCh:     make(chan struct{}),
		processReq:     make(chan []byte),
		processRes:     make(chan error),
		updatePeersReq: make(chan *Peers),
		updatePeersRes: make(chan error),
	}
}

// GetRuntimeConfig implements Config.GetRuntimeConfig
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
	r.currentState = Idle

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
		r.config.Logger.Warningf("truncating local uncommitted log, uncommitted: %d, committed: %d",
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
	// TODO(xq262144), maybe re-validating local log hashes
	return nil
}

func (r *TwoPCRunner) restoreUnderlying() error {
	// TODO(xq262144), restore underlying from snapshot and replaying local logs
	return nil
}

// UpdatePeers implements Runner.UpdatePeers.
func (r *TwoPCRunner) UpdatePeers(peers *Peers) error {
	r.updatePeersLock.Lock()
	defer r.updatePeersLock.Unlock()

	// wait for transaction completion
	// TODO(xq262144), support transaction timeout

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

// Process implements Runner.Process.
func (r *TwoPCRunner) Process(data []byte) error {
	r.processLock.Lock()
	defer r.processLock.Unlock()

	// check leader privilege
	if r.role != Leader {
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
			// TODO(xq262144), cleanup logic
			return
		case data := <-r.processReq:
			r.processRes <- r.processNewLog(data)
		case request := <-r.transport.Process():
			r.processRequest(request)
			// TODO(xq262144), support timeout logic for auto rollback prepared transaction on leader change
		case peersUpdate := <-r.safeForPeersUpdate():
			r.processPeersUpdate(peersUpdate)
		}
	}
}

func (r *TwoPCRunner) safeForPeersUpdate() chan *Peers {
	if r.currentState == Idle {
		return r.updatePeersReq
	}

	return nil
}

func (r *TwoPCRunner) processNewLog(data []byte) error {
	// build Log
	l := &Log{
		Index:    r.lastLogIndex + 1,
		Term:     r.currentTerm,
		Data:     data,
		LastHash: r.lastLogHash,
	}

	// compute hash
	l.ComputeHash()

	// decode log payload
	var err error
	var decodedLog interface{}
	if decodedLog, err = r.decodeLogData(l.Data); err != nil {
		return err
	}

	hasRollback := false

	localPrepare := func(ctx context.Context) error {
		return nestedTimeoutCtx(ctx, r.config.PrepareTimeout, func(prepareCtx context.Context) error {
			// prepare local prepare node
			if err := r.config.Storage.Prepare(prepareCtx, decodedLog); err != nil {
				return err
			}

			// write log to storage
			return r.logStore.StoreLog(l)
		})
	}

	localRollback := func(ctx context.Context) error {
		hasRollback = true

		return nestedTimeoutCtx(ctx, r.config.RollbackTimeout, func(rollbackCtx context.Context) error {
			// prepare local rollback node
			// TODO(xq262144), check log position
			r.logStore.DeleteRange(r.lastLogIndex+1, l.Index)
			return r.config.Storage.Rollback(rollbackCtx, decodedLog)
		})
	}

	localCommit := func(ctx context.Context) error {
		return nestedTimeoutCtx(ctx, r.config.CommitTimeout, func(commitCtx context.Context) error {
			return r.config.Storage.Commit(commitCtx, decodedLog)
		})
	}

	// init context
	ctx, cancel := context.WithTimeout(context.Background(), r.config.ProcessTimeout)
	defer cancel()

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
			localPrepare,
			localRollback,
		))

		// TODO(xq262144), commit error management considering failure node count
		// only return error on transaction has been rollback
		if err := c.Put(nodes, l); err != nil && hasRollback {
			return err
		}
	} else {
		if err := localPrepare(ctx); err != nil {
			localRollback(ctx)
			return err
		}
	}

	// Commit myself
	if err := localCommit(ctx); err != nil {
		return err
	}

	r.stableStore.SetUint64(keyCommittedIndex, l.Index)
	r.lastLogHash = &l.Hash
	r.lastLogIndex = l.Index
	r.lastLogTerm = l.Term

	return nil
}

func (r *TwoPCRunner) setState(state ServerState) {
	r.stateLock.Lock()
	defer r.stateLock.Unlock()
	r.currentState = state
}

func (r *TwoPCRunner) processRequest(req Request) {
	// verify call from leader
	if err := r.verifyLeader(req); err != nil {
		req.SendResponse(err)
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
		req.SendResponse(ErrInvalidRequest)
	}
}

func (r *TwoPCRunner) processPeersUpdate(peersUpdate *Peers) {
	// update peers
	// TODO(xq262144), handle step down, promote up
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
	// TODO(xq262144), verify call from current leader or from new leader containing new peers info
	if req.GetServerID() != r.peers.Leader.ID {
		// not our leader
		return ErrInvalidRequest
	}

	return nil
}

func (r *TwoPCRunner) decodeLogIndex(data interface{}) (uint64, error) {
	if index, ok := data.(uint64); ok {
		return index, nil
	}

	return 0, ErrInvalidLog
}

func (r *TwoPCRunner) decodeLog(data interface{}) (*Log, error) {
	if l, ok := data.(*Log); ok {
		return l, nil
	}

	return nil, ErrInvalidLog
}

func (r *TwoPCRunner) decodeLogData(data []byte) (interface{}, error) {
	var decoded interface{}
	if err := r.config.LogCodec.Decode(data, &decoded); err != nil {
		return nil, err
	}

	return decoded, nil
}

func (r *TwoPCRunner) processPrepare(req Request) {
	req.SendResponse(nestedTimeoutCtx(context.Background(), r.config.PrepareTimeout, func(ctx context.Context) (err error) {
		// already in transaction, try abort previous
		if r.currentState != Idle {
			// TODO(xq262144), has running transaction
			// TODO(xq262144), abort previous or failed current
		}

		// get log
		var l *Log
		if l, err = r.decodeLog(req.GetRequest()); err != nil {
			return
		}

		// validate log
		if !l.VerifyHash() {
			return ErrInvalidLog
		}

		// check log index existence
		var lastIndex uint64
		if lastIndex, err = r.logStore.LastIndex(); err != nil || lastIndex >= l.Index {
			// already prepared or failed
			return err
		}

		// check prepare hash with last log hash
		if l.LastHash != nil && lastIndex == 0 {
			// invalid
			return ErrInvalidLog
		}

		if lastIndex > 0 {
			var lastLog Log
			if err = r.logStore.GetLog(lastIndex, &lastLog); err != nil {
				return err
			}

			if !l.LastHash.IsEqual(&lastLog.Hash) {
				return ErrInvalidLog
			}
		}

		// decode log payload
		var decodedLog interface{}
		if decodedLog, err = r.decodeLogData(l.Data); err != nil {
			return err
		}

		// prepare on storage
		if err = r.config.Storage.Prepare(ctx, decodedLog); err != nil {
			return err
		}

		// write log to storage
		if err = r.logStore.StoreLog(l); err != nil {
			return err
		}

		// set state to prepared
		r.setState(Prepared)

		return nil
	}))
}

func (r *TwoPCRunner) processCommit(req Request) {
	// commit log
	req.SendResponse(nestedTimeoutCtx(context.Background(), r.config.CommitTimeout, func(ctx context.Context) (err error) {
		// TODO(xq262144), check current running transaction index
		if r.currentState != Prepared {
			// not prepared, failed directly
			return ErrInvalidRequest
		}

		// get index
		var index uint64
		if index, err = r.decodeLogIndex(req.GetRequest()); err != nil {
			return
		}

		var lastIndex uint64
		if lastIndex, err = r.logStore.LastIndex(); err != nil {
			return err
		} else if lastIndex < index {
			// not logged, need re-prepare
			return ErrInvalidLog
		}

		if r.lastLogIndex+1 != index {
			// not at the head of the commit position
			return ErrInvalidLog
		}

		// get log
		var lastLog Log
		if err = r.logStore.GetLog(index, &lastLog); err != nil {
			return err
		}

		// decode log
		var decodedLog interface{}
		if decodedLog, err = r.decodeLogData(lastLog.Data); err != nil {
			return err
		}

		// commit on storage
		if err = r.config.Storage.Commit(ctx, decodedLog); err != nil {
			return err
		}

		// commit log
		r.stableStore.SetUint64(keyCommittedIndex, index)
		r.lastLogHash = &lastLog.Hash
		r.lastLogIndex = lastLog.Index
		r.lastLogTerm = lastLog.Term

		// set state to idle
		r.setState(Idle)

		return nil
	}))
}

func (r *TwoPCRunner) processRollback(req Request) {
	// rollback log
	req.SendResponse(nestedTimeoutCtx(context.Background(), r.config.RollbackTimeout, func(ctx context.Context) (err error) {
		// TODO(xq262144), check current running transaction index
		if r.currentState != Prepared {
			// not prepared, failed directly
			return ErrInvalidRequest
		}

		// get index
		var index uint64
		if index, err = r.decodeLogIndex(req.GetRequest()); err != nil {
			return
		}

		var lastIndex uint64
		if lastIndex, err = r.logStore.LastIndex(); err != nil {
			return err
		} else if lastIndex < index {
			// not logged, no rollback required, maybe previous initiated rollback
			return nil
		}

		if r.lastLogIndex+1 != index {
			// not at the head of the commit position
			return ErrInvalidLog
		}

		// get log
		var lastLog Log
		if err = r.logStore.GetLog(index, &lastLog); err != nil {
			return err
		}

		// decode log
		var decodedLog interface{}
		if decodedLog, err = r.decodeLogData(lastLog.Data); err != nil {
			return err
		}

		// rollback on storage
		if err = r.config.Storage.Rollback(ctx, decodedLog); err != nil {
			return err
		}

		// rewind log, can be failed, since committedIndex is not updated
		r.logStore.DeleteRange(r.lastLogIndex+1, index)

		// set state to idle
		r.setState(Idle)

		return nil
	}))
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

// NewTwoPCWorkerWrapper returns a wrapper for remote worker
func NewTwoPCWorkerWrapper(runner *TwoPCRunner, serverID ServerID) *TwoPCWorkerWrapper {
	return &TwoPCWorkerWrapper{
		serverID: serverID,
		runner:   runner,
	}
}

// Prepare implements twopc.Worker.Prepare
func (tpww *TwoPCWorkerWrapper) Prepare(ctx context.Context, wb twopc.WriteBatch) error {
	return tpww.callRemote("Prepare", ctx, wb)
}

// Commit implements twopc.Worker.Commit
func (tpww *TwoPCWorkerWrapper) Commit(ctx context.Context, wb twopc.WriteBatch) error {
	// extract log index only
	l, ok := wb.(*Log)
	if !ok {
		return ErrInvalidLog
	}

	return tpww.callRemote("Commit", ctx, l.Index)
}

// Rollback implements twopc.Worker.Rollback
func (tpww *TwoPCWorkerWrapper) Rollback(ctx context.Context, wb twopc.WriteBatch) error {
	// extract log index only
	l, ok := wb.(*Log)
	if !ok {
		return ErrInvalidLog
	}

	return tpww.callRemote("Rollback", ctx, l.Index)
}

func (tpww *TwoPCWorkerWrapper) callRemote(method string, ctx context.Context, args interface{}) error {
	var remoteErr error

	// TODO(xq262144), handle retry

	if err := tpww.runner.transport.Request(ctx, tpww.serverID, method, args, &remoteErr); err != nil {
		return err
	}

	return remoteErr
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
