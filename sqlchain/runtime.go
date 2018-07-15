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
	"gitlab.com/thunderdb/ThunderDB/proto"
	ct "gitlab.com/thunderdb/ThunderDB/sqlchain/types"
	wt "gitlab.com/thunderdb/ThunderDB/worker/types"
)

// runtime represents a chain runtime state.
type runtime struct {
	wg     sync.WaitGroup
	stopCh chan struct{}

	// chainInitTime is the initial cycle time, when the Genesis blcok is produced.
	chainInitTime time.Time
	// genesisHash is the hash of genesis block.
	genesisHash hash.Hash

	// The following fields are copied from config, and should be constant during whole runtime.

	// databaseID is the current runtime database ID.
	databaseID proto.DatabaseID
	// period is the block producing cycle.
	period time.Duration
	// tick defines the maximum duration between each cycle.
	tick time.Duration
	// queryTTL sets the unacknowledged query TTL in block periods.
	queryTTL int32
	// muxServer is the multiplexing service of sql-chain PRC.
	muxService *MuxService
	// peers is the peer list of the sql-chain.
	peers *kayak.Peers
	// server is the local peer service instance.
	server *kayak.Server
	// price sets query price in gases.
	price map[wt.QueryType]uint32

	sync.RWMutex // Protects following fields.
	// nextTurn is the height of the next block.
	nextTurn int32
	// offset is the time difference calculated by: coodinatedChainTime - time.Now().
	//
	// TODO(leventeliu): update offset in ping cycle.
	offset time.Duration
}

// newRunTime returns a new sql-chain runtime instance with the specified config.
func newRunTime(c *Config) (r *runtime) {
	r = &runtime{
		stopCh:     make(chan struct{}),
		databaseID: c.DatabaseID,
		period:     c.Period,
		tick:       c.Tick,
		queryTTL:   c.QueryTTL,
		muxService: c.MuxService,
		peers:      c.Peers,
		server:     c.Server,
		price:      c.Price,
		nextTurn:   1,
		offset:     time.Duration(0),
	}

	if c.Genesis != nil {
		r.setGenesis(c.Genesis)
	}

	return
}

func (r *runtime) setGenesis(b *ct.Block) {
	r.chainInitTime = b.SignedHeader.Timestamp
	r.genesisHash = b.SignedHeader.BlockHash
}

func (r *runtime) queryTimeIsExpired(t time.Time) bool {
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
	return r.getHeightFromTime(t) < r.nextTurn-r.queryTTL
}

// updateTime updates the current coodinated chain time.
func (r *runtime) updateTime(now time.Time) {
	r.Lock()
	defer r.Unlock()
	r.offset = now.Sub(time.Now())
}

// now returns the current coodinated chain time.
func (r *runtime) now() time.Time {
	r.RLock()
	defer r.RUnlock()
	return time.Now().Add(r.offset)
}

func (r *runtime) getNextTurn() int32 {
	r.Lock()
	defer r.Unlock()
	return r.nextTurn
}

// setNextTurn prepares the runtime state for the next turn.
func (r *runtime) setNextTurn() {
	r.Lock()
	defer r.Unlock()
	r.nextTurn++
}

// getQueryGas gets the consumption of gas for a specified query type.
func (r *runtime) getQueryGas(t wt.QueryType) uint32 {
	return r.price[t]
}

// stop sends a signal to the Runtime stop channel by closing it.
func (r *runtime) stop() {
	r.stopService()
	select {
	case <-r.stopCh:
	default:
		close(r.stopCh)
	}
	r.wg.Wait()
}

// getHeightFromTime calculates the height with this sql-chain config of a given time reading.
func (r *runtime) getHeightFromTime(t time.Time) int32 {
	return int32(t.Sub(r.chainInitTime) / r.period)
}

// nextTick returns the current clock reading and the duration till the next turn. If duration
// is less or equal to 0, use the clock reading to run the next cycle - this avoids some problem
// caused by concurrently time synchronization.
func (r *runtime) nextTick() (t time.Time, d time.Duration) {
	t = r.now()
	d = r.chainInitTime.Add(time.Duration(r.nextTurn) * r.period).Sub(t)

	if d > r.tick {
		d = r.tick
	}

	return
}

func (r *runtime) startService(chain *Chain) {
	r.muxService.register(r.databaseID, &ChainRPCService{chain: chain})
}

func (r *runtime) stopService() {
	r.muxService.unregister(r.databaseID)
}
