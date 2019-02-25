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

package sqlchain

import (
	"sync"
	"sync/atomic"

	"github.com/pkg/errors"

	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

var (
	// Global atomic counters for stats
	multiIndexCount int32
	responseCount   int32
	ackCount        int32
)

type multiAckIndex struct {
	sync.RWMutex
	// respIndex is the index of query responses without acks
	respIndex map[types.QueryKey]*types.SignedResponseHeader
	// ackIndex is the index of acknowledged queries
	ackIndex map[types.QueryKey]*types.SignedAckHeader
}

func (i *multiAckIndex) addResponse(resp *types.SignedResponseHeader) (err error) {
	var key = resp.ResponseHeader.Request.GetQueryKey()
	log.Debugf("adding key %s <-- resp %s", &key, resp.Hash())
	i.Lock()
	defer i.Unlock()
	if oresp, ok := i.respIndex[key]; ok {
		if oresp.Hash() != resp.Hash() {
			err = errors.Wrapf(ErrResponseSeqNotMatch, "add key %s <-- resp %s", &key, resp.Hash())
			return
		}
		return
	}
	i.respIndex[key] = resp
	atomic.AddInt32(&responseCount, 1)
	return
}

func (i *multiAckIndex) register(ack *types.SignedAckHeader) (err error) {
	var (
		resp *types.SignedResponseHeader
		ok   bool
		key  = ack.GetQueryKey()
	)
	log.Debugf("registering key %s <-- ack %s", &key, ack.Hash())

	i.Lock()
	defer i.Unlock()
	if resp, ok = i.respIndex[key]; !ok {
		err = errors.Wrapf(ErrQueryNotFound, "register key %s <-- ack %s", &key, ack.Hash())
		return
	}
	if resp.Hash() != ack.GetResponseHash() {
		err = errors.Wrapf(ErrResponseSeqNotMatch, "register key %s <-- ack %s", &key, ack.Hash())
	}
	delete(i.respIndex, key)
	i.ackIndex[key] = ack
	atomic.AddInt32(&responseCount, -1)
	atomic.AddInt32(&ackCount, 1)
	return
}

func (i *multiAckIndex) remove(ack *types.SignedAckHeader) (err error) {
	var key = ack.GetQueryKey()
	log.Debugf("removing key %s -x- ack %s", &key, ack.Hash())
	i.Lock()
	defer i.Unlock()
	if _, ok := i.respIndex[key]; ok {
		delete(i.respIndex, key)
		atomic.AddInt32(&responseCount, -1)
		return
	}
	if oack, ok := i.ackIndex[key]; ok {
		if oack.Hash() != ack.Hash() {
			err = errors.Wrapf(
				ErrMultipleAckOfSeqNo, "remove key %s -x- ack %s", &key, ack.Hash())
			return
		}
		delete(i.ackIndex, key)
		atomic.AddInt32(&ackCount, -1)
		return
	}
	err = errors.Wrapf(ErrQueryNotFound, "remove key %s -x- ack %s", &key, ack.Hash())
	return
}

func (i *multiAckIndex) acks() (ret []*types.SignedAckHeader) {
	i.RLock()
	defer i.RUnlock()
	for _, v := range i.ackIndex {
		ret = append(ret, v)
	}
	return
}

func (i *multiAckIndex) expire() {
	i.RLock()
	defer i.RUnlock()
	// TODO(leventeliu): need further processing.
	for _, v := range i.respIndex {
		log.WithFields(log.Fields{
			"request_hash":  v.GetRequestHash(),
			"request_time":  v.GetRequestTimestamp(),
			"request_type":  v.Request.QueryType,
			"request_node":  v.Request.NodeID,
			"response_hash": v.Hash(),
			"response_node": v.NodeID,
			"response_time": v.Timestamp,
		}).Warn("query expires without acknowledgement")
	}
	for _, v := range i.ackIndex {
		log.WithFields(log.Fields{
			"request_hash":  v.GetRequestHash(),
			"request_time":  v.GetRequestTimestamp(),
			"request_type":  v.Response.Request.QueryType,
			"request_node":  v.Response.Request.NodeID,
			"response_hash": v.GetResponseHash(),
			"response_node": v.Response.NodeID,
			"response_time": v.GetResponseTimestamp(),
			"ack_hash":      v.Hash(),
			"ack_node":      v.NodeID,
			"ack_time":      v.Timestamp,
		}).Warn("query expires without block producing")
	}
}

type ackIndex struct {
	hi map[int32]*multiAckIndex

	sync.RWMutex
	barrier int32
}

func newAckIndex() *ackIndex {
	return &ackIndex{
		hi: make(map[int32]*multiAckIndex),
	}
}

func (i *ackIndex) load(h int32) (mi *multiAckIndex, err error) {
	var ok bool
	i.Lock()
	defer i.Unlock()
	if h < i.barrier {
		err = errors.Wrapf(ErrQueryExpired, "loading index at height %d barrier %d", h, i.barrier)
		return
	}
	if mi, ok = i.hi[h]; !ok {
		mi = &multiAckIndex{
			respIndex: make(map[types.QueryKey]*types.SignedResponseHeader),
			ackIndex:  make(map[types.QueryKey]*types.SignedAckHeader),
		}
		i.hi[h] = mi
		atomic.AddInt32(&multiIndexCount, 1)
	}
	return
}

func (i *ackIndex) advance(h int32) {
	var dl []*multiAckIndex
	i.Lock()
	for x := i.barrier; x < h; x++ {
		if mi, ok := i.hi[x]; ok {
			dl = append(dl, mi)
		}
		delete(i.hi, x)
	}
	i.barrier = h
	i.Unlock()
	// Record expired and not acknowledged queries
	for _, v := range dl {
		v.expire()
		atomic.AddInt32(&responseCount, int32(-len(v.respIndex)))
		atomic.AddInt32(&ackCount, int32(-len(v.ackIndex)))
	}
	atomic.AddInt32(&multiIndexCount, int32(-len(dl)))
}

func (i *ackIndex) addResponse(h int32, resp *types.SignedResponseHeader) (err error) {
	var mi *multiAckIndex
	if mi, err = i.load(h); err != nil {
		return
	}
	return mi.addResponse(resp)
}

func (i *ackIndex) register(h int32, ack *types.SignedAckHeader) (err error) {
	var mi *multiAckIndex
	if mi, err = i.load(h); err != nil {
		return
	}
	return mi.register(ack)
}

func (i *ackIndex) remove(h int32, ack *types.SignedAckHeader) (err error) {
	var mi *multiAckIndex
	if mi, err = i.load(h); err != nil {
		return
	}
	return mi.remove(ack)
}

func (i *ackIndex) acks(h int32) (ret []*types.SignedAckHeader) {
	var b = func() int32 {
		i.RLock()
		defer i.RUnlock()
		return i.barrier
	}()
	for x := b; x <= h; x++ {
		if mi, err := i.load(x); err == nil {
			ret = append(ret, mi.acks()...)
		}
	}
	return
}
