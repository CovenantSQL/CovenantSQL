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
	kt "github.com/CovenantSQL/CovenantSQL/kayak/types"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

func (r *Runtime) markMissingLog(index uint64) {
	log.WithFields(log.Fields{
		"index":    index,
		"instance": r.instanceID,
	}).Debug("mark log missing, start fetch")
	rawItem, _ := r.waitLogMap.LoadOrStore(index, newWaitItem(index))
	item := rawItem.(*waitItem)

	select {
	case <-r.stopCh:
	case r.missingLogCh <- item:
	}
}

func (r *Runtime) missingLogCycle() {
	for {
		var waitItem *waitItem

		select {
		case <-r.stopCh:
			return
		case waitItem = <-r.missingLogCh:
		}

		// execute
		func() {
			r.peersLock.RLock()
			defer r.peersLock.RUnlock()

			if waitItem == nil {
				return
			}

			waitItem.waitLock.Lock()
			defer waitItem.waitLock.Unlock()

			var (
				req = &kt.FetchRequest{
					Instance: r.instanceID,
					Index:    waitItem.index,
				}
				resp = &kt.FetchResponse{}
				err  error
			)

			// check existence
			if _, err = r.wal.Get(waitItem.index); err == nil {
				// already exists
				log.WithFields(log.Fields{
					"index":    waitItem.index,
					"instance": r.instanceID,
				}).Debug("log already exists")
				r.triggerLogAwaits(waitItem.index)
				return
			}

			if err = r.getCaller(r.peers.Leader).Call(r.fetchRPCMethod, req, resp); err != nil {
				log.WithFields(log.Fields{
					"index":    waitItem.index,
					"instance": r.instanceID,
				}).WithError(err).Debug("fetch log failed")
				return
			}

			// call follower apply
			if resp.Log != nil {
				if err = r.FollowerApply(resp.Log); err != nil {
					log.WithFields(log.Fields{
						"index":    waitItem.index,
						"instance": r.instanceID,
					}).WithError(err).Debug("apply log failed")
				}
			}
		}()
	}
}
