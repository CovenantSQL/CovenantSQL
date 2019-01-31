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

package main

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/CovenantSQL/CovenantSQL/worker"
)

type subscribeWorker struct {
	l      sync.Mutex
	s      *Service
	dbID   proto.DatabaseID
	head   int32
	wg     *sync.WaitGroup
	stopCh chan struct{}
}

func newSubscribeWorker(dbID proto.DatabaseID, head int32, s *Service) *subscribeWorker {
	return &subscribeWorker{
		dbID: dbID,
		head: head,
		s:    s,
	}
}

func (w *subscribeWorker) run() {
	defer w.wg.Done()

	// calc next tick
	var nextTick time.Duration

	for {

		select {
		case <-w.stopCh:
			return
		case <-time.After(nextTick):
			if err := w.pull(atomic.LoadInt32(&w.head)); err != nil {
				// calc next tick
				nextTick = conf.GConf.SQLChainPeriod
			} else {
				nextTick /= 10
			}
		}
	}
}

func (w *subscribeWorker) pull(count int32) (err error) {
	var (
		req  = new(worker.ObserverFetchBlockReq)
		resp = new(worker.ObserverFetchBlockResp)
		next int32
	)

	defer func() {
		lf := log.WithFields(log.Fields{
			"req_count": count,
			"count":     resp.Count,
		})

		if err != nil {
			lf.WithError(err).Debug("sync block failed")
		} else {
			if resp.Block != nil {
				lf = lf.WithField("block", resp.Block.BlockHash())
			} else {
				lf = lf.WithField("block", nil)
			}
			lf.WithField("next", next).Debug("sync block success")
		}
	}()

	req.DatabaseID = w.dbID
	req.Count = count

	if err = w.s.minerRequest(w.dbID, route.DBSObserverFetchBlock.String(), req, resp); err != nil {
		return
	}

	if resp.Block == nil {
		err = errors.New("nil block, try later")
		return
	}

	if err = w.s.addBlock(w.dbID, count, resp.Block); err != nil {
		return
	}

	if count < 0 {
		next = resp.Count + 1
	} else {
		next = count + 1
	}

	atomic.CompareAndSwapInt32(&w.head, count, next)

	return
}

func (w *subscribeWorker) reset(head int32) {
	atomic.StoreInt32(&w.head, head)
	w.start()
}

func (w *subscribeWorker) start() {
	w.l.Lock()
	defer w.l.Unlock()

	if w.isStopped() {
		w.stopCh = make(chan struct{})
		w.wg = new(sync.WaitGroup)
		w.wg.Add(1)
		go w.run()
	}
}

func (w *subscribeWorker) stop() {
	w.l.Lock()
	defer w.l.Unlock()

	if !w.isStopped() {
		// stop
		close(w.stopCh)
		w.wg.Wait()
	}
}

func (w *subscribeWorker) isStopped() bool {
	if w.stopCh == nil {
		return true
	}

	select {
	case <-w.stopCh:
		return true
	default:
		return false
	}
}
