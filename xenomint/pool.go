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

	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	wt "github.com/CovenantSQL/CovenantSQL/worker/types"
)

type pool struct {
	sync.RWMutex
	queries []*wt.Request
	index   map[hash.Hash]*wt.Request
}

func (p *pool) pushQuery(req *wt.Request) (ok bool) {
	p.Lock()
	defer p.Unlock()
	if _, ok = p.index[req.Header.HeaderHash]; ok {
		return
	}
	p.queries = append(p.queries, req)
	p.index[req.Header.HeaderHash] = req
	return
}

func (p *pool) cmpQueries(queries []*wt.Request) (ok bool) {
	p.RLock()
	defer p.RUnlock()
	if ok = (len(p.queries) == len(queries)); !ok {
		return
	}
	for i, v := range p.queries {
		if ok = (v.Header.HeaderHash == queries[i].Header.HeaderHash); !ok {
			return
		}
	}
	return
}
