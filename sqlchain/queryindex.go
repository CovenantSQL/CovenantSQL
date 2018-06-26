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
}

func (s *QuerySummary) UpdateQuerySummary(ack *worker.Ack) (err error) {
	if s.Response == nil {
		s.Response = ack.SignedResponseHeader()
		s.Ack = ack.SignedAckHeader()
	} else if s.Ack == nil {
		s.Response = ack.SignedResponseHeader()
		s.Ack = ack.SignedAckHeader()
	} else {
		err = ErrQueryExists
	}

	return
}

// hash: response header hash
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

func (i *MultiIndex) AddAck(ack *worker.Ack) (err error) {
	i.Lock()
	defer i.Unlock()

	if v, ok := i.HashIndex[ack.ResponseHeaderHash()]; ok && v != nil {
		// This should also update the *SeqAcks indexed by seqNo
		return v.UpdateQuerySummary(ack)
	}

	// Build new QuerySummary and update both indexes
	s := &QuerySummary{
		Response: ack.SignedResponseHeader(),
		Ack:      ack.SignedAckHeader(),
	}

	i.HashIndex[ack.ResponseHeaderHash()] = s
	q := i.SeqIndex.Ensure(ack.SignedRequestHeader().SeqNo)
	q.Queries = append(q.Queries, s)

	if q.FirstAck == nil {
		q.FirstAck = s
	}

	return
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

type QueryIndex struct {
	config      Config
	heightIndex HeightIndex
}

func NewQueryIndex() *QueryIndex {
	return &QueryIndex{
		heightIndex: make(map[int32]*MultiIndex),
	}
}

func (i *QueryIndex) AddAck(ack *worker.Ack) error {
	// TODO(leventeliu): we should ensure that the Request uses coordinated timestamp, instead of
	// any client local time.
	return i.heightIndex.EnsureHeight(i.config.GetHeightFromTime(
		ack.SignedRequestHeader().Timestamp)).AddAck(ack)
}
