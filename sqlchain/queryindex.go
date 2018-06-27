/*
 * Copyright 2018 The ThunderDB Authors.
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

// TODO(leventeliu): use pooled objects to speed up this index.

import (
	"sync"

	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
	"gitlab.com/thunderdb/ThunderDB/worker"
)

type QuerySummary struct {
	// TODO(leventeliu): maybe we don't need them to be "signed" here. Given that the Response or
	// Ack is already verified, simply use Header.
	Response *worker.SignedResponseHeader
	Ack      *worker.SignedAckHeader

	// Packer is the hash of the block which contains this query.
	Packer *hash.Hash
}

func (s *QuerySummary) UpdateQuerySummaryWithResp(resp *worker.SignedResponseHeader) (err error) {
	if s.Response == nil {
		s.Response = resp
		s.Ack = nil
	} // else it's same as s.Response, cause they have same header hash

	return
}

func (s *QuerySummary) UpdateQuerySummaryWithAck(ack *worker.SignedAckHeader) (err error) {
	if s.Response == nil {
		s.Response = ack.SignedResponseHeader()
		s.Ack = ack
	} else if s.Ack == nil {
		// A later Ack can overwrite the original Response setting
		s.Response = ack.SignedResponseHeader()
		s.Ack = ack
	} else if !s.Ack.HeaderHash.IsEqual(&ack.HeaderHash) {
		// This may happen when a client sends multiple acknowledgements for a same query (same
		// response header hash)
		err = ErrMultipleAck
	} // else it's same as s.Ack, let's try not to overwrite it

	return
}

type HashIndex map[hash.Hash]*QuerySummary

type SeqAcks struct {
	FirstAck *QuerySummary
	Queries  []*QuerySummary
}

func NewSeqAcks() *SeqAcks {
	return &SeqAcks{
		// TODO(leventeliu): set appropriate capacity.
		FirstAck: nil,
		Queries:  make([]*QuerySummary, 0, 10),
	}
}

type SeqIndex map[uint64]*SeqAcks

func (i SeqIndex) Ensure(seq uint64) (v *SeqAcks) {
	v, ok := i[seq]

	if !ok {
		v = NewSeqAcks()
		i[seq] = v
	}

	return
}

type MultiIndex struct {
	sync.Mutex
	HashIndex
	SeqIndex
}

func NewMultiIndex() *MultiIndex {
	return &MultiIndex{
		HashIndex: make(map[hash.Hash]*QuerySummary),
		SeqIndex:  make(map[uint64]*SeqAcks),
	}
}

func (i *MultiIndex) AddResponse(resp *worker.SignedResponseHeader) error {
	i.Lock()
	defer i.Unlock()

	if v, ok := i.HashIndex[resp.HeaderHash]; ok && v != nil {
		return v.UpdateQuerySummaryWithResp(resp)
	}

	// Build new QuerySummary and update both indexes
	s := &QuerySummary{
		Response: resp,
	}

	i.HashIndex[resp.HeaderHash] = s
	q := i.SeqIndex.Ensure(resp.Request.SeqNo)
	q.Queries = append(q.Queries, s)

	return nil
}

func (i *MultiIndex) AddAck(ack *worker.SignedAckHeader) error {
	i.Lock()
	defer i.Unlock()

	if v, ok := i.HashIndex[ack.ResponseHeaderHash()]; ok && v != nil {
		// This should also update the *SeqAcks indexed by seqNo
		return v.UpdateQuerySummaryWithAck(ack)
	}

	// Build new QuerySummary and update both indexes
	s := &QuerySummary{
		Response: ack.SignedResponseHeader(),
		Ack:      ack,
	}

	i.HashIndex[ack.ResponseHeaderHash()] = s
	q := i.SeqIndex.Ensure(ack.SignedRequestHeader().SeqNo)
	q.Queries = append(q.Queries, s)

	// TODO(leventeliu):
	// This query has multiple signed acknowledgements. It may be caused by a network problem.
	// We will keep the first ack counted anyway. But, should we report it to someone?
	if q.FirstAck == nil || q.FirstAck.Ack.Timestamp.After(s.Ack.Timestamp) {
		q.FirstAck = s
	}

	return nil
}

func (i *MultiIndex) expire() {
	i.Lock()
	defer i.Unlock()

	for _, q := range i.SeqIndex {
		if q.FirstAck == nil {
			// TODO(leventeliu):
			// This query is not acknowledged and expires now.
		} else if q.FirstAck.Packer == nil {
			// TODO(leventeliu):
			// This query was acknowledged normally but collectors didn't pack it in any block.
			// There is definitely something wrong with them.
		}

		for _, s := range q.Queries {
			if s != q.FirstAck {
				// TODO(leventeliu): so these guys lost the competition in this query. Should we
				// do something about it?
			}
		}
	}
}

type HeightIndex map[int32]*MultiIndex

func (i HeightIndex) EnsureRange(l, h int32) {
	for x := l; x < h; x++ {
		if _, ok := i[x]; !ok {
			i[x] = NewMultiIndex()
		}
	}
}

func (i HeightIndex) EnsureHeight(h int32) (v *MultiIndex) {
	v, ok := i[h]

	if !ok {
		v = NewMultiIndex()
		i[h] = v
	}

	return
}

func (i HeightIndex) HasHeight(h int32) (ok bool) {
	_, ok = i[h]
	return
}

type QueryIndex struct {
	barrier     int32
	heightIndex HeightIndex
}

func NewQueryIndex() *QueryIndex {
	return &QueryIndex{
		heightIndex: make(map[int32]*MultiIndex),
	}
}

func (i *QueryIndex) AddResponse(h int32, resp *worker.SignedResponseHeader) error {
	// TODO(leventeliu): we should ensure that the Request uses coordinated timestamp, instead of
	// any client local time.
	return i.heightIndex.EnsureHeight(h).AddResponse(resp)
}

func (i *QueryIndex) AddAck(h int32, ack *worker.SignedAckHeader) error {
	return i.heightIndex.EnsureHeight(h).AddAck(ack)
}

func (i *QueryIndex) expireHeight(h int32) {
	if i.heightIndex.HasHeight(h) {
		i.heightIndex[h].expire()
		delete(i.heightIndex, h)
	}
}

// AdvanceBarrier moves barrier to given height. All buckets lower than this height will be set as
// expired, and all the queries which are not packed in these buckets will be reported.
func (i *QueryIndex) AdvanceBarrier(height int32) {
	for x := i.barrier; x < height; x++ {
		i.expireHeight(x)
	}

	i.barrier = height
}
