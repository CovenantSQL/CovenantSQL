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

package xenomint

import (
	"database/sql"
	"sync/atomic"
	"time"

	wt "github.com/CovenantSQL/CovenantSQL/worker/types"
	xi "github.com/CovenantSQL/CovenantSQL/xenomint/interfaces"
	"github.com/pkg/errors"
)

type state struct {
	strg xi.Storage
	pool *pool
	// unc is the uncommitted transaction.
	unc *sql.Tx
	id  uint64
}

func newState(strg xi.Storage) (s *state, err error) {
	var t = &state{
		strg: strg,
	}
	if t.unc, err = t.strg.Writer().Begin(); err != nil {
		return
	}
	s = t
	return
}

func (s *state) incSeq() {
	atomic.AddUint64(&s.id, 1)
}

func (s *state) setNextTxID() {
	var old = atomic.LoadUint64(&s.id)
	atomic.StoreUint64(&s.id, (old&uint64(0xffffffff00000000))+uint64(1)<<32)
}

func (s *state) resetTxID() {
	var old = atomic.LoadUint64(&s.id)
	atomic.StoreUint64(&s.id, old&uint64(0xffffffff00000000))
}

func (s *state) rollbackID(id uint64) {
	atomic.StoreUint64(&s.id, id)
}

func (s *state) getID() uint64 {
	return atomic.LoadUint64(&s.id)
}

func (s *state) close(commit bool) (err error) {
	if s.unc != nil {
		if commit {
			if err = s.unc.Commit(); err != nil {
				return
			}
		} else {
			if err = s.unc.Rollback(); err != nil {
				return
			}
		}
	}
	return s.strg.Close()
}

func buildArgsFromSQLNamedArgs(args []sql.NamedArg) (ifs []interface{}) {
	ifs = make([]interface{}, len(args))
	for i, v := range args {
		ifs[i] = v
	}
	return
}

func buildTypeNamesFromSQLColumnTypes(types []*sql.ColumnType) (names []string) {
	names = make([]string, len(types))
	for i, v := range types {
		names[i] = v.DatabaseTypeName()
	}
	return
}

type sqlQuerier interface {
	Query(query string, args ...interface{}) (*sql.Rows, error)
}

func readSingle(
	qer sqlQuerier, q *wt.Query) (names []string, types []string, data [][]interface{}, err error,
) {
	var (
		rows *sql.Rows
		cols []*sql.ColumnType
	)
	if rows, err = qer.Query(
		q.Pattern, buildArgsFromSQLNamedArgs(q.Args)...,
	); err != nil {
		return
	}
	defer rows.Close()
	// Fetch column names and types
	if names, err = rows.Columns(); err != nil {
		return
	}
	if cols, err = rows.ColumnTypes(); err != nil {
		return
	}
	types = buildTypeNamesFromSQLColumnTypes(cols)
	// Scan data row by row
	data = make([][]interface{}, 0)
	for rows.Next() {
		var (
			row  = make([]interface{}, len(cols))
			dest = make([]interface{}, len(cols))
		)
		for i := range row {
			dest[i] = &row[i]
		}
		if err = rows.Scan(dest...); err != nil {
			return
		}
		data = append(data, row)
	}
	return
}

func buildRowsFromNativeData(data [][]interface{}) (rows []wt.ResponseRow) {
	rows = make([]wt.ResponseRow, len(data))
	for i, v := range data {
		rows[i].Values = v
	}
	return
}

func (s *state) read(req *wt.Request) (resp *wt.Response, err error) {
	var (
		ierr         error
		names, types []string
		data         [][]interface{}
	)
	// TODO(leventeliu): no need to run every read query here.
	for i, v := range req.Payload.Queries {
		if names, types, data, ierr = readSingle(s.strg.DirtyReader(), &v); ierr != nil {
			err = errors.Wrapf(ierr, "query at #%d failed", i)
			return
		}
	}
	// Build query response
	resp = &wt.Response{
		Header: wt.SignedResponseHeader{
			ResponseHeader: wt.ResponseHeader{
				Request:   req.Header,
				NodeID:    "",
				Timestamp: time.Now(),
				RowCount:  uint64(len(data)),
				LogOffset: s.getID(),
			},
		},
		Payload: wt.ResponsePayload{
			Columns:   names,
			DeclTypes: types,
			Rows:      buildRowsFromNativeData(data),
		},
	}
	return
}

func (s *state) readTx(req *wt.Request) (resp *wt.Response, err error) {
	var (
		tx           *sql.Tx
		id           uint64
		ierr         error
		names, types []string
		data         [][]interface{}
	)
	if tx, err = s.strg.DirtyReader().Begin(); err != nil {
		return
	}
	defer tx.Rollback()
	id = s.getID()
	for i, v := range req.Payload.Queries {
		if names, types, data, ierr = readSingle(tx, &v); ierr != nil {
			err = errors.Wrapf(ierr, "query at #%d failed", i)
			return
		}
	}
	// Build query response
	resp = &wt.Response{
		Header: wt.SignedResponseHeader{
			ResponseHeader: wt.ResponseHeader{
				Request:   req.Header,
				NodeID:    "",
				Timestamp: time.Now(),
				RowCount:  uint64(len(data)),
				LogOffset: id,
			},
		},
		Payload: wt.ResponsePayload{
			Columns:   names,
			DeclTypes: types,
			Rows:      buildRowsFromNativeData(data),
		},
	}
	return
}

func (s *state) writeSingle(q *wt.Query) (res sql.Result, err error) {
	if res, err = s.unc.Exec(q.Pattern, buildArgsFromSQLNamedArgs(q.Args)...); err == nil {
		s.incSeq()
	}
	return
}

func (s *state) setSavepoint() (savepoint uint64) {
	savepoint = s.getID()
	s.unc.Exec("SAVEPOINT ?", savepoint)
	return
}

func (s *state) rollbackTo(savepoint uint64) {
	s.rollbackID(savepoint)
	s.unc.Exec("ROLLBACK TO ?", savepoint)
}

func (s *state) write(req *wt.Request) (resp *wt.Response, err error) {
	var (
		ierr      error
		savepoint = s.setSavepoint()
	)
	// FIXME(leventeliu): savepoint is a sqlite-specified solution for nested transaction.
	for i, v := range req.Payload.Queries {
		if _, ierr = s.writeSingle(&v); ierr != nil {
			err = errors.Wrapf(ierr, "execute at #%d failed", i)
			s.rollbackTo(savepoint)
			return
		}
	}
	// Build query response
	resp = &wt.Response{
		Header: wt.SignedResponseHeader{
			ResponseHeader: wt.ResponseHeader{
				Request:   req.Header,
				NodeID:    "",
				Timestamp: time.Now(),
				RowCount:  0,
				LogOffset: savepoint,
			},
		},
	}
	s.pool.enqueue(resp)
	return
}

func (s *state) replay(req *wt.Request, resp *wt.Response) (err error) {
	var (
		loc *wt.Response
		ok  bool
	)
	if loc, ok = s.pool.loadResponse(resp.Header.LogOffset); ok {
		if !req.Header.HeaderHash.IsEqual(&loc.Header.Request.HeaderHash) {
			err = ErrQueryConflict
			return
		}
		return
	}
	if resp.Header.ResponseHeader.LogOffset != s.getID() {
		err = ErrQueryConflict
		return
	}
	if loc, err = s.write(req); err != nil {
		return
	}
	s.pool.enqueue(resp)
	return
}

func (s *state) commit() (err error) {
	if err = s.unc.Commit(); err != nil {
		return
	}
	if s.unc, err = s.strg.Writer().Begin(); err != nil {
		return
	}
	s.setNextTxID()
	return
}

func (s *state) rollback() (err error) {
	if err = s.unc.Rollback(); err != nil {
		return
	}
	s.resetTxID()
	return
}

func (s *state) Query(req *wt.Request) (resp *wt.Response, err error) {
	switch req.Header.QueryType {
	case wt.ReadQuery:
		return s.readTx(req)
	case wt.WriteQuery:
		return s.write(req)
	default:
		err = ErrInvalidRequest
	}
	return
}

func (s *state) Replay(req *wt.Request, resp *wt.Response) (err error) {
	switch req.Header.QueryType {
	case wt.ReadQuery:
		return
	case wt.WriteQuery:
		return s.replay(req, resp)
	default:
		err = ErrInvalidRequest
	}
	return
}
