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

package kayak

import (
	"context"
	"sync"
	"sync/atomic"

	kt "github.com/CovenantSQL/CovenantSQL/kayak/types"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils/trace"
)

// rpcTracker defines the rpc call tracker
// support tracking the rpc result.
type rpcTracker struct {
	// related runtime
	r *Runtime
	// target nodes, a copy of current followers
	nodes []proto.NodeID
	// rpc method
	method string
	// rpc request
	req interface{}
	// minimum response count
	minCount int
	// responses
	errLock sync.RWMutex
	errors  map[proto.NodeID]error
	// scoreboard
	complete int
	sent     uint32
	doneOnce sync.Once
	doneCh   chan struct{}
	wg       sync.WaitGroup
	closed   uint32
}

func newTracker(r *Runtime, req interface{}, minCount int) (t *rpcTracker) {
	// copy nodes
	nodes := append([]proto.NodeID(nil), r.followers...)

	if minCount > len(nodes) {
		minCount = len(nodes)
	}
	if minCount < 0 {
		minCount = 0
	}

	t = &rpcTracker{
		r:        r,
		nodes:    nodes,
		method:   r.applyRPCMethod,
		req:      req,
		minCount: minCount,
		errors:   make(map[proto.NodeID]error, len(nodes)),
		doneCh:   make(chan struct{}),
	}

	return
}

func (t *rpcTracker) send() {
	if !atomic.CompareAndSwapUint32(&t.sent, 0, 1) {
		return
	}

	for i := range t.nodes {
		t.wg.Add(1)
		go t.callSingle(i)
	}

	if t.minCount == 0 {
		t.done()
	}
}

func (t *rpcTracker) callSingle(idx int) {
	err := t.r.getCaller(t.nodes[idx]).Call(t.method, t.req, nil)
	defer t.wg.Done()
	t.errLock.Lock()
	defer t.errLock.Unlock()
	t.errors[t.nodes[idx]] = err
	t.complete++

	if t.complete >= t.minCount {
		t.done()
	}
}

func (t *rpcTracker) done() {
	t.doneOnce.Do(func() {
		if t.doneCh != nil {
			select {
			case <-t.doneCh:
			default:
				close(t.doneCh)
			}
		}
	})
}

func (t *rpcTracker) get(ctx context.Context) (errors map[proto.NodeID]error, meets bool, finished bool) {
	if trace.IsEnabled() {
		// get request log type
		traceType := "rpcCall"

		if rawReq, ok := t.req.(*kt.ApplyRequest); ok {
			traceType += rawReq.Log.Type.String()
		}

		defer trace.StartRegion(ctx, traceType).End()
	}

	for {
		select {
		case <-t.doneCh:
			meets = true
		default:
		}

		select {
		case <-ctx.Done():
		case <-t.doneCh:
			meets = true
		}

		break
	}

	t.errLock.RLock()
	defer t.errLock.RUnlock()

	errors = make(map[proto.NodeID]error)

	for s, e := range t.errors {
		errors[s] = e
	}

	if !meets && len(errors) >= t.minCount {
		meets = true
	}

	if len(errors) == len(t.nodes) {
		finished = true
	}

	return
}

func (t *rpcTracker) close() {
	if !atomic.CompareAndSwapUint32(&t.closed, 0, 1) {
		return
	}

	t.wg.Wait()
	t.done()
}
