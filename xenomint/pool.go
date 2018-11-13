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

	"github.com/CovenantSQL/CovenantSQL/types"
)

type QueryTracker struct {
	sync.RWMutex
	req  *types.Request
	resp *types.Response
}

func (q *QueryTracker) UpdateResp(resp *types.Response) {
	q.Lock()
	defer q.Unlock()
	q.resp = resp
}

type pool struct {
	queries []*QueryTracker
	index   map[uint64]int
}

func newPool() *pool {
	return &pool{
		queries: make([]*QueryTracker, 0),
		index:   make(map[uint64]int),
	}
}

func (p *pool) enqueue(sp uint64, q *QueryTracker) {
	var pos = len(p.queries)
	p.queries = append(p.queries, q)
	p.index[sp] = pos
	return
}

func (p *pool) match(sp uint64, req *types.Request) bool {
	var (
		pos int
		ok  bool
	)
	if pos, ok = p.index[sp]; !ok {
		return false
	}
	if p.queries[pos].req.Header.Hash() != req.Header.Hash() {
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
}
