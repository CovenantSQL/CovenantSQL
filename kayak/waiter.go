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
	"time"

	kt "github.com/CovenantSQL/CovenantSQL/kayak/types"
	"github.com/CovenantSQL/CovenantSQL/utils/trace"
)

type waitItem struct {
	index    uint64
	doneOnce sync.Once
	ch       chan struct{}
	waitLock sync.Mutex
}

func newWaitItem(index uint64) *waitItem {
	return &waitItem{
		index: index,
		ch:    make(chan struct{}),
	}
}

func (r *Runtime) waitForLog(ctx context.Context, index uint64) (l *kt.Log, err error) {
	defer trace.StartRegion(ctx, "waitForLog").End()

	for {
		if l, err = r.wal.Get(index); err == nil {
			// exists
			return
		}

		rawItem, _ := r.waitLogMap.LoadOrStore(index, newWaitItem(index))
		item := rawItem.(*waitItem)

		if item == nil {
			err = kt.ErrInvalidLog
			return
		}

		select {
		case <-item.ch:
			r.waitLogMap.Delete(index)
		case <-time.After(r.logWaitTimeout):
			r.markMissingLog(index)
		case <-ctx.Done():
			err = ctx.Err()
			return
		}
	}
}

func (r *Runtime) triggerLogAwaits(index uint64) {
	rawItem, ok := r.waitLogMap.Load(index)
	if !ok || rawItem == nil {
		return
	}

	item := rawItem.(*waitItem)

	if item == nil {
		return
	}

	item.doneOnce.Do(func() {
		if item.ch != nil {
			close(item.ch)
		}
	})
}
