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

package worker

import (
	"container/list"
	"context"

	"github.com/CovenantSQL/CovenantSQL/sqlchain/storage"
	"github.com/CovenantSQL/CovenantSQL/twopc"
	"github.com/CovenantSQL/CovenantSQL/utils"
	wt "github.com/CovenantSQL/CovenantSQL/worker/types"
)

// Following contains storage related logic extracted from main database instance definition.

// Prepare implements twopc.Worker.Prepare.
func (db *Database) Prepare(ctx context.Context, wb twopc.WriteBatch) (err error) {
	// wrap storage with signature check
	var log *storage.ExecLog
	if log, err = db.convertRequest(wb); err != nil {
		return
	}
	return db.storage.Prepare(ctx, log)
}

// Commit implements twopc.Worker.Commmit.
func (db *Database) Commit(ctx context.Context, wb twopc.WriteBatch) (err error) {
	// wrap storage with signature check
	var log *storage.ExecLog
	if log, err = db.convertRequest(wb); err != nil {
		return
	}
	db.recordSequence(log)
	return db.storage.Commit(ctx, log)
}

// Rollback implements twopc.Worker.Rollback.
func (db *Database) Rollback(ctx context.Context, wb twopc.WriteBatch) (err error) {
	// wrap storage with signature check
	var log *storage.ExecLog
	if log, err = db.convertRequest(wb); err != nil {
		return
	}
	db.recordSequence(log)
	return db.storage.Rollback(ctx, log)
}

func (db *Database) recordSequence(log *storage.ExecLog) {
	db.connSeqs.Store(log.ConnectionID, log.SeqNo)
}

func (db *Database) verifySequence(log *storage.ExecLog) (err error) {
	var data interface{}
	var ok bool
	var lastSeq uint64

	if data, ok = db.connSeqs.Load(log.ConnectionID); ok {
		lastSeq, _ = data.(uint64)

		if log.SeqNo <= lastSeq {
			return ErrInvalidRequestSeq
		}
	}

	return
}

func (db *Database) convertRequest(wb twopc.WriteBatch) (log *storage.ExecLog, err error) {
	var ok bool

	// type convert
	var payloadBytes []byte
	if payloadBytes, ok = wb.([]byte); !ok {
		err = ErrInvalidRequest
		return
	}

	// decode
	var req wt.Request
	if err = utils.DecodeMsgPack(payloadBytes, &req); err != nil {
		return
	}

	// verify
	if err = req.Verify(); err != nil {
		return
	}

	// verify timestamp
	nowTime := getLocalTime()
	minTime := nowTime.Add(-db.cfg.MaxWriteTimeGap)
	maxTime := nowTime.Add(db.cfg.MaxWriteTimeGap)

	if req.Header.Timestamp.Before(minTime) || req.Header.Timestamp.After(maxTime) {
		err = ErrInvalidRequest
		return
	}

	// convert
	log = new(storage.ExecLog)
	log.ConnectionID = req.Header.ConnectionID
	log.SeqNo = req.Header.SeqNo
	log.Timestamp = req.Header.Timestamp.UnixNano()
	log.Queries = convertQuery(req.Payload.Queries)

	// verify connection sequence
	if err = db.verifySequence(log); err != nil {
		return
	}

	return
}

func (db *Database) evictSequences() {
	m := make(map[uint64]*list.Element)
	l := list.New()

	for connSeq := range db.connSeqEvictCh {
		if e, ok := m[connSeq]; ok {
			l.MoveToFront(e)
			return
		}

		e := l.PushFront(connSeq)
		m[connSeq] = e

		if l.Len() > MaxRecordedConnectionSequences {
			e = l.Back()
			if e != nil {
				l.Remove(e)
				evictSeq := e.Value.(uint64)
				delete(m, evictSeq)
				db.connSeqs.Delete(evictSeq)
			}
		}
	}
}
