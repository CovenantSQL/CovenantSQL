/*
 * Copyright 2019 The CovenantSQL Authors.
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

package rpc

import (
	"context"
	"expvar"
	"net/rpc"
	"sync"
	"time"

	"github.com/pkg/errors"
	mw "github.com/zserge/metric"

	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
)

var (
	callRPCExpvarLock sync.Mutex
)

func recordRPCCost(startTime time.Time, method string, err error) {
	var (
		name, nameC string
		val, valC   expvar.Var
	)
	costTime := time.Since(startTime)
	if err == nil {
		name = "t_succ:" + method
		nameC = "c_succ:" + method
	} else {
		name = "t_fail:" + method
		nameC = "c_fail:" + method
	}
	// Optimistically, val will not be nil except the first Call of method
	// expvar uses sync.Map
	// So, we try it first without lock
	val = expvar.Get(name)
	valC = expvar.Get(nameC)
	if val == nil || valC == nil {
		callRPCExpvarLock.Lock()
		val = expvar.Get(name)
		if val == nil {
			expvar.Publish(name, mw.NewHistogram("10s1s", "1m5s", "1h1m"))
			expvar.Publish(nameC, mw.NewCounter("10s1s", "1h1m"))
		}
		callRPCExpvarLock.Unlock()
		val = expvar.Get(name)
		valC = expvar.Get(nameC)
	}
	val.(mw.Metric).Add(costTime.Seconds())
	valC.(mw.Metric).Add(1)
	return
}

// Caller is a wrapper for session pooling and RPC calling.
type Caller struct {
	pool NOClientPool
}

// NewCallerWithPool returns a new Caller with the pool.
func NewCallerWithPool(pool NOClientPool) *Caller {
	return &Caller{
		pool: pool,
	}
}

// CallNodeWithContext calls node method with context.
func (c *Caller) CallNodeWithContext(
	ctx context.Context, node proto.NodeID, method string, args, reply interface{}) (err error,
) {
	startTime := time.Now()
	defer func() {
		recordRPCCost(startTime, method, err)
	}()

	client, err := DialToNodeWithPool(c.pool, node, method == route.DHTPing.String())
	if err != nil {
		err = errors.Wrapf(err, "dial to node %s failed", node)
		return
	}
	defer func() { _ = client.Close() }()

	// TODO(xq262144): golang net/rpc does not support cancel in progress calls
	ch := client.Go(method, args, reply, make(chan *rpc.Call, 1))

	select {
	case <-ctx.Done():
		err = ctx.Err()
	case call := <-ch.Done:
		err = call.Error
		// Set error state so that the associated will not reuse this client
		if err != nil { // TODO(leventeliu): check recoverable errors
			if setter, ok := client.(LastErrSetter); ok {
				setter.SetLastErr(err)
			}
		}
	}

	return
}

// CallNode calls node method.
func (c *Caller) CallNode(node proto.NodeID, method string, args, reply interface{}) (err error) {
	return c.CallNodeWithContext(context.Background(), node, method, args, reply)
}
