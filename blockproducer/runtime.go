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

package blockproducer

import (
	"sync"
	"time"

	"gitlab.com/thunderdb/ThunderDB/kayak"

	"gitlab.com/thunderdb/ThunderDB/proto"
	"gitlab.com/thunderdb/ThunderDB/rpc"
)

// copy from /sqlchain/runtime.go
// rt define the runtime of main chain
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

	// TODO(lambda): why need this lock, and why not include index?
	sync.RWMutex // Protects following fields.
	peers        *kayak.Peers
	nodeID       proto.NodeID
	// nextTurn is the height of the next block.
	nextTurn uint32
	// offset is the time difference calculated by: coodinatedChainTime - time.Now().
	//
	// TODO(leventeliu): update offset in ping cycle.
	offset time.Duration
}

// now returns the current coodinated chain time.
func (r *rt) now() time.Time {
	r.RLock()
	defer r.RUnlock()
	// TODO(lambda): why does sqlchain not need UTC
	return time.Now().UTC().Add(r.offset)
}

func newRuntime(cfg *config, accountAddress proto.AccountAddress) *rt {
	var index uint32
	for i, s := range cfg.peers.Servers {
		if cfg.nodeID == s.ID {
			index = uint32(i)
		}
	}
	return &rt{
		chainInitTime:  cfg.genesis.SignedHeader.Timestamp,
		accountAddress: accountAddress,
		server:         cfg.server,
		bpNum:          uint32(len(cfg.peers.Servers)),
		index:          index,
		period:         cfg.period,
		tick:           cfg.tick,
		peers:          cfg.peers,
		nodeID:         cfg.nodeID,
		nextTurn:       1,
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
	r.Lock()
	defer r.Unlock()
	return r.nextTurn%r.bpNum == r.index
}

// setNextTurn prepares the runtime state for the next turn.
func (r *rt) setNextTurn() {
	r.Lock()
	defer r.Unlock()
	r.nextTurn++
}
