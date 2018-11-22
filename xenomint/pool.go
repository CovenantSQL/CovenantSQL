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

package xenomint

import (
	"sync"
	"sync/atomic"

	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/types"
)

// QueryTracker defines an object to track query as a request - response pair.
type QueryTracker struct {
	sync.RWMutex
	Req  *types.Request
	Resp *types.Response
}

// UpdateResp updates response of the QueryTracker within locking scope.
func (q *QueryTracker) UpdateResp(resp *types.Response) {
	q.Lock()
	defer q.Unlock()
	q.Resp = resp
}

// Ready reports whether the query is ready for block producing. It is assumed that all objects
// should be ready shortly.
func (q *QueryTracker) Ready() bool {
	q.RLock()
	defer q.RUnlock()
	return q.Resp != nil
}

type pool struct {
	// Failed queries: hash => Request
	failed map[hash.Hash]*types.Request
	// Succeeded queries and their index
	queries []*QueryTracker
	index   map[uint64]int
	// Atomic counters for stats
	failedRequestCount int32
	trackerCount       int32
}

func newPool() *pool {
	return &pool{
		failed:  make(map[hash.Hash]*types.Request),
		queries: make([]*QueryTracker, 0),
		index:   make(map[uint64]int),
	}
}

func (p *pool) enqueue(sp uint64, q *QueryTracker) {
	var pos = len(p.queries)
	p.queries = append(p.queries, q)
	p.index[sp] = pos
	atomic.StoreInt32(&p.trackerCount, int32(len(p.queries)))
	return
}

func (p *pool) setFailed(req *types.Request) {
	p.failed[req.Header.Hash()] = req
	atomic.StoreInt32(&p.failedRequestCount, int32(len(p.failed)))
}

func (p *pool) failedList() (reqs []*types.Request) {
	reqs = make([]*types.Request, 0, len(p.failed))
	for _, v := range p.failed {
		reqs = append(reqs, v)
	}
	return
}

func (p *pool) removeFailed(req *types.Request) {
	delete(p.failed, req.Header.Hash())
	atomic.StoreInt32(&p.failedRequestCount, int32(len(p.failed)))
}

func (p *pool) match(sp uint64, req *types.Request) bool {
	var (
		pos int
		ok  bool
	)
	if pos, ok = p.index[sp]; !ok {
		return false
	}
	if p.queries[pos].Req.Header.Hash() != req.Header.Hash() {
		return false
	}
	return true
}

func (p *pool) matchLast(sp uint64) bool {
	var (
		pos int
		ok  bool
	)
	if pos, ok = p.index[sp]; !ok {
		return false
	}
	if pos != len(p.queries)-1 {
		return false
	}
	return true
}

func (p *pool) truncate(sp uint64) {
	var (
		pos int
		ok  bool
		ni  map[uint64]int
	)
	if pos, ok = p.index[sp]; !ok {
		return
	}
	// Rebuild index
	ni = make(map[uint64]int)
	for k, v := range p.index {
		if k > sp {
			ni[k] = v - (pos + 1)
		}
	}
	p.index = ni
	p.queries = p.queries[pos+1:]
	atomic.StoreInt32(&p.trackerCount, int32(len(p.queries)))
}
