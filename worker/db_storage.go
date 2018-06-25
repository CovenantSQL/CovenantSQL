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

package worker

import (
	"context"

	"gitlab.com/thunderdb/ThunderDB/sqlchain/storage"
	"gitlab.com/thunderdb/ThunderDB/twopc"
	"gitlab.com/thunderdb/ThunderDB/utils"
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
	var req Request
	if err = utils.DecodeMsgPack(payloadBytes, &req); err != nil {
		return
	}

	// verify
	if err = req.Verify(); err != nil {
		return
	}

	// verify timestamp
	// TODO(xq262144), maybe using central time service
	nowTime := db.getLocalTime()
	minTime := nowTime.Add(-db.cfg.MaxWriteTimeGap)
	maxTime := nowTime.Add(db.cfg.MaxWriteTimeGap)

	// TODO(xq262144), need refactor, response to connection attacks
	if req.Header.Timestamp.Before(minTime) || req.Header.Timestamp.After(maxTime) {
		err = ErrInvalidRequest
		return
	}

	// verify connection sequence
	if err = db.verifySequence(log); err != nil {
		return
	}

	// convert
	log = new(storage.ExecLog)
	log.ConnectionID = req.Header.ConnectionID
	log.SeqNo = req.Header.SeqNo
	log.Timestamp = req.Header.Timestamp.UnixNano()
	log.Queries = make([]string, len(req.Payload.Queries))
	copy(log.Queries, req.Payload.Queries)

	return
}
