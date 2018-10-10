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
	"fmt"
	"sync"
	"time"

	"github.com/CovenantSQL/CovenantSQL/kayak"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/rpc"
)

// copy from /sqlchain/runtime.go
// rt define the runtime of main chain.
type rt struct {
	wg     sync.WaitGroup
	stopCh chan struct{}

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
	peers      *kayak.Peers
	nodeID     proto.NodeID

	stateMutex sync.Mutex // Protects following fields.
	// nextTurn is the height of the next block.
	nextTurn uint32
	// head is the current head of the best chain.
	head *State

	// timeMutex protects following time-relative fields.
	timeMutex sync.Mutex
	// offset is the time difference calculated by: coodinatedChainTime - time.Now().
	//
	// TODO(leventeliu): update offset in ping cycle.
	offset time.Duration
}

// now returns the current coodinated chain time.
func (r *rt) now() time.Time {
	r.timeMutex.Lock()
	defer r.timeMutex.Unlock()
	// TODO(lambda): why does sqlchain not need UTC
	return time.Now().UTC().Add(r.offset)
}

func newRuntime(cfg *Config, accountAddress proto.AccountAddress) *rt {
	var index uint32
	for i, s := range cfg.Peers.Servers {
		if cfg.NodeID.IsEqual(&s.ID) {
			index = uint32(i)
		}
	}
	return &rt{
		stopCh:         make(chan struct{}),
		chainInitTime:  cfg.Genesis.SignedHeader.Timestamp,
		accountAddress: accountAddress,
		server:         cfg.Server,
		bpNum:          uint32(len(cfg.Peers.Servers)),
		index:          index,
		period:         cfg.Period,
		tick:           cfg.Tick,
		peers:          cfg.Peers,
		nodeID:         cfg.NodeID,
		nextTurn:       1,
		head:           &State{},
		offset:         time.Duration(0),
	}
}

func (r *rt) startService(chain *Chain) {
	r.server.RegisterService(MainChainRPCName, &ChainRPCService{chain: chain})
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

func (r *rt) getPeers() *kayak.Peers {
	r.peersMutex.Lock()
	defer r.peersMutex.Unlock()
	peers := r.peers.Clone()
	return &peers
}

func (r *rt) getHead() *State {
	r.stateMutex.Lock()
	defer r.stateMutex.Unlock()
	return r.head
}

func (r *rt) setHead(head *State) {
	r.stateMutex.Lock()
	defer r.stateMutex.Unlock()
	r.head = head
}

func (r *rt) stop() {
	r.stopService()
	select {
	case <-r.stopCh:
	default:
		close(r.stopCh)
	}
	r.wg.Wait()
}

func (r *rt) stopService() {
}
