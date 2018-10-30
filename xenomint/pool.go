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

	wt "github.com/CovenantSQL/CovenantSQL/worker/types"
)

type query struct {
	req  *wt.Request
	resp *wt.Response
}

type pool struct {
	sync.RWMutex
	queries []*query
	index   map[uint64]int
}

func newPool() *pool {
	return &pool{
		queries: make([]*query, 0),
		index:   make(map[uint64]int),
	}
}

func (p *pool) enqueue(req *wt.Request, resp *wt.Response) {
	p.Lock()
	defer p.Unlock()
	var pos = len(p.queries)
	p.queries = append(p.queries, &query{req: req, resp: resp})
	p.index[resp.Header.LogOffset] = pos
	return
}
