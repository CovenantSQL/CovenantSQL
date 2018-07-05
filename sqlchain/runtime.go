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

package sqlchain

import (
	"sync"
	"time"

	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
	"gitlab.com/thunderdb/ThunderDB/kayak"
	ct "gitlab.com/thunderdb/ThunderDB/sqlchain/types"
	wt "gitlab.com/thunderdb/ThunderDB/worker/types"
)

// Runtime represents a chain runtime state.
type Runtime struct {
	wg     sync.WaitGroup
	stopCh chan struct{}

	// ChainInitTime is the initial cycle time, when the Genesis blcok is produced.
	ChainInitTime time.Time
	// GenesisHash is the hash of genesis block.
	GenesisHash hash.Hash

	// The following fields are copied from config, and should be constant during whole runtime.

	// Period is the block producing cycle.
	Period time.Duration
	// Tick defines the maximum duration between each cycle.
	Tick time.Duration
	// QueryTTL sets the unacknowledged query TTL in block periods.
	QueryTTL int32
	// Peers is the peer list of the sql-chain.
	Peers *kayak.Peers
	// Server is the local peer service instance.
	Server *kayak.Server
	// Price sets query price in gases.
	Price map[wt.QueryType]uint32

	sync.RWMutex // Protects following fields.
	// NextTurn is the height of the next block.
	NextTurn int32
	// Offset is the time difference calculated by: coodinatedChainTime - time.Now().
	//
	// TODO(leventeliu): update Offset in ping cycle.
	Offset time.Duration
}

// NewRunTime returns a new sql-chain runtime instance with the specified config.
func NewRunTime(c *Config) (r *Runtime) {
	r = &Runtime{
		stopCh:   make(chan struct{}),
		Period:   c.Period,
		Tick:     c.Tick,
		QueryTTL: c.QueryTTL,
		Peers:    c.Peers,
		Server:   c.Server,
		Price:    c.Price,
		NextTurn: 1,
		Offset:   time.Duration(0),
	}

	if c.Genesis != nil {
		r.setGenesis(c.Genesis)
	}

	return
}

func (r *Runtime) setGenesis(b *ct.Block) {
	r.ChainInitTime = b.SignedHeader.Timestamp
	r.GenesisHash = b.SignedHeader.BlockHash
}

func (r *Runtime) queryTimeIsExpired(t time.Time) bool {
	// Checking query expiration for the pending block, whose height is c.rt.NextHeight:
	//
	//     TimeLived = r.NextTurn - r.GetHeightFromTime(t)
	//
	// Return true if:  QueryTTL < TimeLived.
	//
	// NOTE(leventeliu): as a result, a TTL=1 requires any query to be acknowledged and received
	// immediately.
	// Consider the case that a query has happended right before period h, which has height h.
	// If its ACK+Roundtrip time>0, it will be seemed as acknowledged in period h+1, or even later.
	// So, period h+1 has NextHeight h+2, and TimeLived of this query will be 2 at that time - it
	// has expired.
	//
	return r.GetHeightFromTime(t) < r.NextTurn-r.QueryTTL
}

// UpdateTime updates the current coodinated chain time.
func (r *Runtime) UpdateTime(now time.Time) {
	r.Lock()
	defer r.Unlock()
	r.Offset = now.Sub(time.Now())
}

// Now returns the current coodinated chain time.
func (r *Runtime) Now() time.Time {
	r.RLock()
	defer r.RUnlock()
	return time.Now().Add(r.Offset)
}

func (r *Runtime) getNextTurn() int32 {
	r.Lock()
	defer r.Unlock()
	return r.NextTurn
}

// setNextTurn prepares the runtime state for the next turn.
func (r *Runtime) setNextTurn() {
	r.Lock()
	defer r.Unlock()
	r.NextTurn++
}

// getQueryGas gets the consumption of gas for a specified query type.
func (r *Runtime) getQueryGas(t wt.QueryType) uint32 {
	return r.Price[t]
}

// Stop sends a signal to the Runtime stop channel by closing it.
func (r *Runtime) Stop() {
	close(r.stopCh)
	r.wg.Wait()
}

// GetHeightFromTime calculates the height with this sql-chain config of a given time reading.
func (r *Runtime) GetHeightFromTime(t time.Time) int32 {
	return int32(t.Sub(r.ChainInitTime) / r.Period)
}

// NextTick returns the current clock reading and the duration till the next turn. If duration
// is less or equal to 0, use the clock reading to run the next cycle - this avoids some problem
// caused by concurrently time synchronization.
func (r *Runtime) NextTick() (t time.Time, d time.Duration) {
	t = r.Now()
	d = r.ChainInitTime.Add(time.Duration(r.NextTurn) * r.Period).Sub(t)

	if d > r.Tick {
		d = r.Tick
	}

	return
}
