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
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	xi "github.com/CovenantSQL/CovenantSQL/xenomint/interfaces"
	"github.com/CovenantSQL/sqlparser"
	"github.com/pkg/errors"
)

// State defines a xenomint state which is bound to a underlying storage.
type State struct {
	sync.RWMutex
	strg   xi.Storage
	pool   *pool
	closed bool
	nodeID proto.NodeID

	// TODO(leventeliu): Reload savepoint from last block on chain initialization, and rollback
	// any ongoing transaction on exit.
	//
	// unc is the uncommitted transaction.
	unc             *sql.Tx
	origin          uint64 // origin is the original savepoint of the current transaction
	cmpoint         uint64 // cmpoint is the last commit point of the current transaction
	current         uint64 // current is the current savepoint of the current transaction
	hasSchemaChange uint32 // indicates schema change happens in this uncommitted transaction
}

// NewState returns a new State bound to strg.
func NewState(nodeID proto.NodeID, strg xi.Storage) (s *State, err error) {
	var t = &State{
		nodeID: nodeID,
		strg:   strg,
		pool:   newPool(),
	}
	if t.unc, err = t.strg.Writer().Begin(); err != nil {
		return
	}
	t.setSavepoint()
	s = t
	return
}

func (s *State) incSeq() {
	s.current++
}

func (s *State) setNextTxID() {
	s.origin = s.current
	s.cmpoint = s.current
}

func (s *State) setCommitPoint() {
	s.cmpoint = s.current
}

func (s *State) rollbackID(id uint64) {
	s.current = id
}

// InitTx sets the initial id of the current transaction. This method is not safe for concurrency
// and should only be called at initialization.
func (s *State) InitTx(id uint64) {
	s.origin = id
	s.cmpoint = id
	s.current = id
	s.setSavepoint()
}

func (s *State) getID() uint64 {
	return atomic.LoadUint64(&s.current)
}

// Close commits any ongoing transaction if needed and closes the underlying storage.
func (s *State) Close(commit bool) (err error) {
	if s.closed {
		return
	}
	if s.unc != nil {
		if commit {
			if err = s.uncCommit(); err != nil {
				return
			}
		} else {
			// Only rollback to last commmit point
			if err = s.rollback(); err != nil {
				return
			}
			if err = s.uncCommit(); err != nil {
				return
			}
		}
	}
	if err = s.strg.Close(); err != nil {
		return
	}
	s.closed = true
	return
}

func convertQueryAndBuildArgs(pattern string, args []types.NamedArg) (containsDDL bool, p string, ifs []interface{}, err error) {
	var (
		tokenizer  = sqlparser.NewStringTokenizer(pattern)
		stmt       sqlparser.Statement
		lastPos    int
		query      string
		queryParts []string
	)

	for {
		stmt, err = sqlparser.ParseNext(tokenizer)

		if err != nil && err != io.EOF {
			return
		}

		if err == io.EOF {
			err = nil
			break
		}

		query = pattern[lastPos : tokenizer.Position-1]
		lastPos = tokenizer.Position + 1

		// translate show statement
		if showStmt, ok := stmt.(*sqlparser.Show); ok {
			origQuery := query

			switch showStmt.Type {
			case "table":
				if showStmt.ShowCreate {
					query = "SELECT sql FROM sqlite_master WHERE type = \"table\" AND tbl_name = \"" +
						showStmt.OnTable.Name.String() + "\""
				} else {
					query = "PRAGMA table_info(" + showStmt.OnTable.Name.String() + ")"
				}
			case "index":
				query = "SELECT name FROM sqlite_master WHERE type = \"index\" AND tbl_name = \"" +
					showStmt.OnTable.Name.String() + "\""
			case "tables":
				query = "SELECT name FROM sqlite_master WHERE type = \"table\""
			}

			log.WithFields(log.Fields{
				"from": origQuery,
				"to":   query,
			}).Debug("query translated")
		} else if _, ok := stmt.(*sqlparser.DDL); ok {
			containsDDL = true
		}

		queryParts = append(queryParts, query)
	}

	p = strings.Join(queryParts, "; ")

	ifs = make([]interface{}, len(args))
	for i, v := range args {
		ifs[i] = sql.NamedArg{
			Name:  v.Name,
			Value: v.Value,
		}
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
	qer sqlQuerier, q *types.Query) (names []string, types []string, data [][]interface{}, err error,
) {
	var (
		rows    *sql.Rows
		cols    []*sql.ColumnType
		pattern string
		args    []interface{}
	)

	if _, pattern, args, err = convertQueryAndBuildArgs(q.Pattern, q.Args); err != nil {
		return
	}
	if rows, err = qer.Query(pattern, args...); err != nil {
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

func buildRowsFromNativeData(data [][]interface{}) (rows []types.ResponseRow) {
	rows = make([]types.ResponseRow, len(data))
	for i, v := range data {
		rows[i].Values = v
	}
	return
}

func (s *State) read(req *types.Request) (ref *QueryTracker, resp *types.Response, err error) {
	var (
		ierr           error
		cnames, ctypes []string
		data           [][]interface{}
	)
	// TODO(leventeliu): no need to run every read query here.
	for i, v := range req.Payload.Queries {
		if cnames, ctypes, data, ierr = readSingle(s.strg.DirtyReader(), &v); ierr != nil {
			err = errors.Wrapf(ierr, "query at #%d failed", i)
			// Add to failed pool list
			s.pool.setFailed(req)
			return
		}
	}
	// Build query response
	ref = &QueryTracker{Req: req}
	resp = &types.Response{
		Header: types.SignedResponseHeader{
			ResponseHeader: types.ResponseHeader{
				Request:   req.Header,
				NodeID:    s.nodeID,
				Timestamp: s.getLocalTime(),
				RowCount:  uint64(len(data)),
				LogOffset: s.getID(),
			},
		},
		Payload: types.ResponsePayload{
			Columns:   cnames,
			DeclTypes: ctypes,
			Rows:      buildRowsFromNativeData(data),
		},
	}
	return
}

func (s *State) readTx(req *types.Request) (ref *QueryTracker, resp *types.Response, err error) {
	var (
		tx             *sql.Tx
		id             uint64
		ierr           error
		cnames, ctypes []string
		data           [][]interface{}
		querier        sqlQuerier
	)
	id = s.getID()
	if atomic.LoadUint32(&s.hasSchemaChange) == 1 {
		// lock transaction
		s.Lock()
		defer s.Unlock()
		s.setSavepoint()
		querier = s.unc
		defer s.rollbackTo(id)
	} else {
		if tx, ierr = s.strg.DirtyReader().Begin(); ierr != nil {
			err = errors.Wrap(ierr, "open tx failed")
			return
		}
		querier = tx
		defer tx.Rollback()
	}

	for i, v := range req.Payload.Queries {
		if cnames, ctypes, data, ierr = readSingle(querier, &v); ierr != nil {
			err = errors.Wrapf(ierr, "query at #%d failed", i)
			// Add to failed pool list
			s.pool.setFailed(req)
			return
		}
	}
	// Build query response
	ref = &QueryTracker{Req: req}
	resp = &types.Response{
		Header: types.SignedResponseHeader{
			ResponseHeader: types.ResponseHeader{
				Request:   req.Header,
				NodeID:    s.nodeID,
				Timestamp: s.getLocalTime(),
				RowCount:  uint64(len(data)),
				LogOffset: id,
			},
		},
		Payload: types.ResponsePayload{
			Columns:   cnames,
			DeclTypes: ctypes,
			Rows:      buildRowsFromNativeData(data),
		},
	}
	return
}

func (s *State) writeSingle(q *types.Query) (res sql.Result, err error) {
	var (
		containsDDL bool
		pattern     string
		args        []interface{}
	)

	if containsDDL, pattern, args, err = convertQueryAndBuildArgs(q.Pattern, q.Args); err != nil {
		return
	}
	if res, err = s.unc.Exec(pattern, args...); err == nil {
		if containsDDL {
			atomic.StoreUint32(&s.hasSchemaChange, 1)
		}
		s.incSeq()
	}
	return
}

func (s *State) setSavepoint() (savepoint uint64) {
	savepoint = s.getID()
	s.unc.Exec("SAVEPOINT \"?\"", savepoint)
	return
}

func (s *State) rollbackTo(savepoint uint64) {
	s.rollbackID(savepoint)
	s.unc.Exec("ROLLBACK TO \"?\"", savepoint)
}

func (s *State) write(req *types.Request) (ref *QueryTracker, resp *types.Response, err error) {
	var (
		savepoint         uint64
		query             = &QueryTracker{Req: req}
		totalAffectedRows int64
		curAffectedRows   int64
		lastInsertID      int64
	)

	// TODO(leventeliu): savepoint is a sqlite-specified solution for nested transaction.
	if err = func() (err error) {
		var ierr error
		s.Lock()
		defer s.Unlock()
		savepoint = s.getID()
		for i, v := range req.Payload.Queries {
			var res sql.Result
			if res, ierr = s.writeSingle(&v); ierr != nil {
				err = errors.Wrapf(ierr, "execute at #%d failed", i)
				// Add to failed pool list
				s.pool.setFailed(req)
				s.rollbackTo(savepoint)
				return
			}

			curAffectedRows, _ = res.RowsAffected()
			lastInsertID, _ = res.LastInsertId()
			totalAffectedRows += curAffectedRows
		}
		s.setSavepoint()
		s.pool.enqueue(savepoint, query)
		return
	}(); err != nil {
		return
	}
	// Build query response
	ref = query
	resp = &types.Response{
		Header: types.SignedResponseHeader{
			ResponseHeader: types.ResponseHeader{
				Request:      req.Header,
				NodeID:       s.nodeID,
				Timestamp:    s.getLocalTime(),
				RowCount:     0,
				LogOffset:    savepoint,
				AffectedRows: totalAffectedRows,
				LastInsertID: lastInsertID,
			},
		},
	}
	return
}

func (s *State) replay(req *types.Request, resp *types.Response) (err error) {
	var (
		ierr      error
		savepoint uint64
		query     = &QueryTracker{Req: req, Resp: resp}
	)
	s.Lock()
	defer s.Unlock()
	savepoint = s.getID()
	if resp.Header.ResponseHeader.LogOffset != savepoint {
		err = errors.Wrapf(
			ErrQueryConflict,
			"local id %d vs replaying id %d", savepoint, resp.Header.ResponseHeader.LogOffset,
		)
		return
	}
	for i, v := range req.Payload.Queries {
		if _, ierr = s.writeSingle(&v); ierr != nil {
			err = errors.Wrapf(ierr, "execute at #%d failed", i)
			s.rollbackTo(savepoint)
			return
		}
	}
	s.setSavepoint()
	s.pool.enqueue(savepoint, query)
	return
}

// ReplayBlock replays the queries from block. It also checks and skips some preceding pooled
// queries.
func (s *State) ReplayBlock(block *types.Block) (err error) {
	var (
		ierr   error
		lastsp uint64 // Last savepoint
	)
	s.Lock()
	defer s.Unlock()
	for i, q := range block.QueryTxs {
		var query = &QueryTracker{Req: q.Request, Resp: &types.Response{Header: *q.Response}}
		lastsp = s.getID()
		if q.Response.ResponseHeader.LogOffset > lastsp {
			err = ErrMissingParent
			return
		}
		// Match and skip already pooled query
		if q.Response.ResponseHeader.LogOffset < lastsp {
			if !s.pool.match(q.Response.ResponseHeader.LogOffset, q.Request) {
				err = ErrQueryConflict
				return
			}
			continue
		}
		// Replay query
		for j, v := range q.Request.Payload.Queries {
			if q.Request.Header.QueryType == types.ReadQuery {
				continue
			}
			if q.Request.Header.QueryType != types.WriteQuery {
				err = errors.Wrapf(ErrInvalidRequest, "replay block at %d:%d", i, j)
				s.rollbackTo(lastsp)
				return
			}
			if _, ierr = s.writeSingle(&v); ierr != nil {
				err = errors.Wrapf(ierr, "execute at %d:%d failed", i, j)
				s.rollbackTo(lastsp)
				return
			}
		}
		s.setSavepoint()
		s.pool.enqueue(lastsp, query)
	}
	// Remove duplicate failed queries from local pool
	for _, r := range block.FailedReqs {
		s.pool.removeFailed(r)
	}
	// Check if the current transaction is ok to commit
	if s.pool.matchLast(lastsp) {
		if err = s.uncCommit(); err != nil {
			// FATAL ERROR
			return
		}
		if s.unc, err = s.strg.Writer().Begin(); err != nil {
			// FATAL ERROR
			return
		}
		s.setNextTxID()
	} else {
		// Set commit point only, transaction is not actually committed. This commit point will be
		// used on exiting.
		s.setCommitPoint()
	}
	s.setSavepoint()
	// Truncate pooled queries
	s.pool.truncate(lastsp)
	return
}

func (s *State) commit() (err error) {
	s.Lock()
	defer s.Unlock()
	if err = s.uncCommit(); err != nil {
		return
	}
	if s.unc, err = s.strg.Writer().Begin(); err != nil {
		return
	}
	s.setNextTxID()
	s.setSavepoint()
	_ = s.pool.queries
	s.pool = newPool()
	return
}

// CommitEx commits the current transaction and returns all the pooled queries.
func (s *State) CommitEx() (failed []*types.Request, queries []*QueryTracker, err error) {
	s.Lock()
	defer s.Unlock()
	if err = s.uncCommit(); err != nil {
		// FATAL ERROR
		return
	}
	if s.unc, err = s.strg.Writer().Begin(); err != nil {
		// FATAL ERROR
		return
	}
	s.setNextTxID()
	s.setSavepoint()
	// Return pooled items and reset
	failed = s.pool.failedList()
	queries = s.pool.queries
	s.pool = newPool()
	return
}

func (s *State) uncCommit() (err error) {
	if err = s.unc.Commit(); err != nil {
		return
	}

	// reset schema change flag
	atomic.StoreUint32(&s.hasSchemaChange, 0)

	return
}

func (s *State) rollback() (err error) {
	s.Lock()
	defer s.Unlock()
	s.rollbackTo(s.cmpoint)
	s.current = s.cmpoint
	return
}

func (s *State) getLocalTime() time.Time {
	return time.Now().UTC()
}

// Query does the query(ies) in req, pools the request and persists any change to
// the underlying storage.
func (s *State) Query(req *types.Request) (ref *QueryTracker, resp *types.Response, err error) {
	switch req.Header.QueryType {
	case types.ReadQuery:
		return s.readTx(req)
	case types.WriteQuery:
		return s.write(req)
	default:
		err = ErrInvalidRequest
	}
	return
}

// Replay replays a write log from other peer to replicate storage state.
func (s *State) Replay(req *types.Request, resp *types.Response) (err error) {
	// NOTE(leventeliu): in the current implementation, failed requests are not tracked in remote
	// nodes (while replaying via Replay calls). Because we don't want to actually replay read
	// queries in all synchronized nodes, meanwhile, whether a request will fail or not
	// remains unknown until we actually replay it -- a dead end here.
	// So we just keep failed requests in local pool and report them in the next local block
	// producing.
	switch req.Header.QueryType {
	case types.ReadQuery:
		return
	case types.WriteQuery:
		return s.replay(req, resp)
	default:
		err = ErrInvalidRequest
	}
	return
}

// Stat prints the statistic message of the State object.
func (s *State) Stat(id proto.DatabaseID) {
	var (
		p = func() *pool {
			s.RLock()
			defer s.RUnlock()
			return s.pool
		}()
		fc = atomic.LoadInt32(&p.failedRequestCount)
		tc = atomic.LoadInt32(&p.trackerCount)
	)
	log.WithFields(log.Fields{
		"database_id":               id,
		"pooled_fail_request_count": fc,
		"pooled_query_tracker":      tc,
	}).Info("Xeno pool stats")
}
