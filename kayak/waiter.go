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

package kayak

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"

	kt "github.com/CovenantSQL/CovenantSQL/kayak/types"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/CovenantSQL/CovenantSQL/utils/trace"
)

type waitItem struct {
	r          *Runtime
	index      uint64
	l          sync.RWMutex
	log        *kt.Log
	fetchTimer *time.Timer
	started    uint32
	stopCh     chan struct{}
	waitCh     chan struct{}
}

func newWaitItem(r *Runtime, index uint64) *waitItem {
	return &waitItem{
		r:      r,
		index:  index,
		waitCh: make(chan struct{}, 1),
		stopCh: make(chan struct{}),
	}
}

func (i *waitItem) startFetch() {
	i.l.Lock()
	defer i.l.Unlock()

	if atomic.CompareAndSwapUint32(&i.started, 0, 1) {
		go i.run()
	}
}

func (i *waitItem) run() {
	// startFetch and apply and trigger pending
	i.r.peersLock.RLock()
	defer i.r.peersLock.RUnlock()

	// check log existence
	if l, err := i.r.wal.Get(i.index); err == nil {
		i.set(l)
		return
	}

	var (
		req = &kt.FetchRequest{
			Instance: i.r.instanceID,
			Index:    i.index,
		}
		resp *kt.FetchResponse
		err  error
	)

	// fetch log
	for {
		select {
		case <-i.stopCh:
			return
		case <-time.After(i.r.logWaitTimeout):
		}

		resp = new(kt.FetchResponse)

		if err = i.r.getCaller(i.r.peers.Leader).Call(i.r.fetchRPCMethod, req, resp); err != nil {
			log.WithFields(log.Fields{
				"index":    i.index,
				"instance": i.r.instanceID,
			}).WithError(err).Debug("send fetch rpc failed")
			continue
		} else if resp.Log == nil {
			log.WithFields(log.Fields{
				"index":    i.index,
				"instance": i.r.instanceID,
			}).Debug("could not fetch log")
			continue
		}

		if err = i.r.followerApply(resp.Log, false); err != nil {
			// apply log
			log.WithFields(log.Fields{
				"index":    i.index,
				"instance": i.r.instanceID,
			}).WithError(err).Debug("apply log failed")
			continue
		}

		return
	}
}

func (i *waitItem) get() *kt.Log {
	i.l.RLock()
	defer i.l.RUnlock()

	return i.log
}

func (i *waitItem) set(l *kt.Log) {
	i.l.Lock()
	defer i.l.Unlock()

	i.log = l

	if i.waitCh != nil {
		select {
		case <-i.waitCh:
		default:
			close(i.waitCh)
		}
	}

	if i.stopCh != nil {
		select {
		case <-i.stopCh:
		default:
			close(i.stopCh)
		}
	}
}

func (r *Runtime) waitForLog(ctx context.Context, index uint64) (l *kt.Log, err error) {
	defer trace.StartRegion(ctx, "waitForLog").End()

	if l, err = r.wal.Get(index); err == nil {
		// exists
		return
	}

	rawItem, _ := r.waitLogMap.LoadOrStore(index, newWaitItem(r, index))
	item := rawItem.(*waitItem)

	if item == nil {
		err = kt.ErrInvalidLog
		return
	}

	item.startFetch()

	select {
	case <-item.waitCh:
		l = item.get()
		if l != nil {
			err = nil
		} else {
			err = errors.Wrapf(kt.ErrInvalidLog, "could not fetch log %d", index)
		}
		r.waitLogMap.Delete(index)
	case <-ctx.Done():
		err = ctx.Err()
	}

	return
}

func (r *Runtime) triggerLogAwaits(l *kt.Log) {
	rawItem, ok := r.waitLogMap.Load(l.Index)
	if !ok || rawItem == nil {
		return
	}

	item := rawItem.(*waitItem)

	if item == nil {
		return
	}

	item.set(l)
}
