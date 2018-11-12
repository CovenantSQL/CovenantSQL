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
