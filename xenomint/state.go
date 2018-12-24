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
	"context"
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

	// unc is the uncommitted transaction.
	unc             *sql.Tx
	maxTx           uint64
	lastCommitPoint uint64
	current         uint64 // current is the current lastSeq of the current transaction
	hasSchemaChange uint32 // indicates schema change happens in this uncommitted transaction
}

// NewState returns a new State bound to strg.
func NewState(nodeID proto.NodeID, strg xi.Storage) (s *State, err error) {
	var t = &State{
		nodeID: nodeID,
		strg:   strg,
		pool:   newPool(),
		maxTx:  100,
	}
	if t.unc, err = t.strg.Writer().Begin(); err != nil {
		return
	}
	s = t
	return
}

func (s *State) incSeq() {
	atomic.AddUint64(&s.current, 1)
}

func (s *State) setSeq(id uint64) {
	atomic.StoreUint64(&s.current, id)
}

// SetSeq sets the initial id of the current transaction.
func (s *State) SetSeq(id uint64) {
	s.setSeq(id)
}

func (s *State) getSeq() uint64 {
	return atomic.LoadUint64(&s.current)
}

func (s *State) getLastCommitPoint() uint64 {
	return atomic.LoadUint64(&s.lastCommitPoint)
}

// Close commits any ongoing transaction if needed and closes the underlying storage.
func (s *State) Close(commit bool) (err error) {
	s.Lock()
	defer s.Unlock()
	if s.closed {
		return
	}
	if s.unc != nil {
		if commit {
			if err = s.uncCommit(); err != nil {
				log.WithError(err).Fatal("failed to commit")
			}
		} else {
			if err = s.uncRollback(); err != nil {
				log.WithError(err).Fatal("failed to rollback")
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
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
}

func readSingle(
	ctx context.Context, qer sqlQuerier, q *types.Query,
) (
	names []string, types []string, data [][]interface{}, err error,
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
	if rows, err = qer.QueryContext(ctx, pattern, args...); err != nil {
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
	return s.readWithContext(context.Background(), req)
}

func (s *State) readWithContext(
	ctx context.Context, req *types.Request) (ref *QueryTracker, resp *types.Response, err error,
) {
	var (
		ierr           error
		cnames, ctypes []string
		data           [][]interface{}
	)
	// TODO(leventeliu): no need to run every read query here.
	for i, v := range req.Payload.Queries {
		if cnames, ctypes, data, ierr = readSingle(ctx, s.strg.DirtyReader(), &v); ierr != nil {
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
				LogOffset: s.getSeq(),
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

func (s *State) readTx(
	ctx context.Context, req *types.Request) (ref *QueryTracker, resp *types.Response, err error,
) {
	var (
		id             = s.getSeq()
		ierr           error
		cnames, ctypes []string
		data           [][]interface{}
		querier        sqlQuerier
	)
	if atomic.LoadUint32(&s.hasSchemaChange) == 1 {
		// lock transaction
		s.Lock()
		defer s.Unlock()
		querier = s.unc
	} else {
		var tx *sql.Tx
		if tx, ierr = s.strg.DirtyReader().Begin(); ierr != nil {
			err = errors.Wrap(ierr, "open tx failed")
			return
		}
		querier = tx
		defer tx.Rollback()
	}

	defer func() {
		if ctx.Err() != nil {
			log.WithError(ctx.Err()).WithFields(log.Fields{
				"req":       req,
				"id":        id,
				"dirtyRead": atomic.LoadUint32(&s.hasSchemaChange) != 1,
			}).Warning("read query canceled")
		}
	}()

	for i, v := range req.Payload.Queries {
		if cnames, ctypes, data, ierr = readSingle(ctx, querier, &v); ierr != nil {
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

func (s *State) writeSingle(
	ctx context.Context, q *types.Query) (res sql.Result, err error,
) {
	var (
		containsDDL bool
		pattern     string
		args        []interface{}
		//start       = time.Now()

		//parsed, executed time.Duration
	)

	//defer func() {
	//	var fields = log.Fields{}
	//	fields["lastSeq"] = s.current
	//	if parsed > 0 {
	//		fields["1#parsed"] = float64(parsed.Nanoseconds()) / 1000
	//	}
	//	if executed > 0 {
	//		fields["2#executed"] = float64((executed - parsed).Nanoseconds()) / 1000
	//	}
	//	log.WithFields(fields).Debug("writeSingle duration stat (us)")
	//}()
	if containsDDL, pattern, args, err = convertQueryAndBuildArgs(q.Pattern, q.Args); err != nil {
		return
	}
	//parsed = time.Since(start)
	if res, err = s.unc.Exec(pattern, args...); err == nil {
		if containsDDL {
			atomic.StoreUint32(&s.hasSchemaChange, 1)
		}
		s.incSeq()
	}
	//executed = time.Since(start)
	return
}

func (s *State) write(
	ctx context.Context, req *types.Request) (ref *QueryTracker, resp *types.Response, err error,
) {
	var (
		lastSeq           uint64
		query             = &QueryTracker{Req: req}
		totalAffectedRows int64
		curAffectedRows   int64
		lastInsertID      int64
		start             = time.Now()

		lockAcquired, writeDone, enqueued, lockReleased, respBuilt time.Duration
	)

	defer func() {
		var fields = log.Fields{}
		fields["lastSeq"] = lastSeq
		fields["1#lockAcquired"] = float64(lockAcquired.Nanoseconds()) / 1000
		if writeDone > 0 {
			fields["2#writeDone"] = float64((writeDone - lockAcquired).Nanoseconds()) / 1000
		}
		if enqueued > 0 {
			fields["3#enqueued"] = float64((enqueued - writeDone).Nanoseconds()) / 1000
		}
		if lockReleased > 0 {
			fields["4#lockReleased"] = float64((lockReleased - enqueued).Nanoseconds()) / 1000
		}
		if respBuilt > 0 {
			fields["5#respBuilt"] = float64((respBuilt - lockReleased).Nanoseconds()) / 1000
		}
		log.WithFields(fields).Debug("Write duration stat (us)")
		if ctx.Err() != nil {
			log.WithError(err).WithField("req", req).Warning("write query canceled")
		}
	}()

	if err = func() (err error) {
		var (
			ierr error
			qcnt = len(req.Payload.Queries)
		)
		s.Lock()
		lockAcquired = time.Since(start)
		defer func() {
			s.Unlock()
			lockReleased = time.Since(start)
		}()
		lastSeq = s.getSeq()
		if qcnt > 1 {
			// Set savepoint
			if _, ierr = s.unc.Exec(`SAVEPOINT "?"`, lastSeq); ierr != nil {
				err = errors.Wrapf(ierr, "failed to create savepoint %d", lastSeq)
				return
			}
			defer s.unc.Exec(`ROLLBACK TO "?"`, lastSeq)
		}
		for i, v := range req.Payload.Queries {
			var res sql.Result
			if res, ierr = s.writeSingle(ctx, &v); ierr != nil {
				err = errors.Wrapf(ierr, "execute at #%d failed", i)
				// TODO(leventeliu): request may actually be partial successed without
				// rolling back.
				s.pool.setFailed(req)
				return
			}

			curAffectedRows, _ = res.RowsAffected()
			lastInsertID, _ = res.LastInsertId()
			totalAffectedRows += curAffectedRows
		}
		if qcnt > 1 {
			// Release savepoint
			if _, ierr = s.unc.Exec(`RELEASE SAVEPOINT "?"`, lastSeq); ierr != nil {
				err = errors.Wrapf(ierr, "failed to release savepoint %d", lastSeq)
				return
			}
		}
		// Try to commit if the ongoing tx is too large or schema is changed
		if s.getSeq()-s.getLastCommitPoint() > s.maxTx ||
			atomic.LoadUint32(&s.hasSchemaChange) != 0 {
			s.tryCommit()
		}
		writeDone = time.Since(start)
		s.pool.enqueue(lastSeq, query)
		enqueued = time.Since(start)
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
				LogOffset:    lastSeq,
				AffectedRows: totalAffectedRows,
				LastInsertID: lastInsertID,
			},
		},
	}
	respBuilt = time.Since(start)
	return
}

func (s *State) replay(ctx context.Context, req *types.Request, resp *types.Response) (err error) {
	var (
		ierr    error
		lastSeq uint64
		query   = &QueryTracker{Req: req, Resp: resp}
	)
	s.Lock()
	defer s.Unlock()
	lastSeq = s.getSeq()
	if resp.Header.ResponseHeader.LogOffset != lastSeq {
		err = errors.Wrapf(
			ErrQueryConflict,
			"local id %d vs replaying id %d", lastSeq, resp.Header.ResponseHeader.LogOffset,
		)
		return
	}
	for i, v := range req.Payload.Queries {
		if _, ierr = s.writeSingle(ctx, &v); ierr != nil {
			err = errors.Wrapf(ierr, "execute at #%d failed", i)
			return
		}
	}
	// Try to commit if the ongoing tx is too large or schema is changed
	if s.getSeq()-s.getLastCommitPoint() > s.maxTx ||
		atomic.LoadUint32(&s.hasSchemaChange) != 0 {
		s.tryCommit()
	}
	s.pool.enqueue(lastSeq, query)
	return
}

// ReplayBlock replays the queries from block. It also checks and skips some preceding pooled
// queries.
func (s *State) ReplayBlock(block *types.Block) (err error) {
	return s.ReplayBlockWithContext(context.Background(), block)
}

// ReplayBlockWithContext replays the queries from block with context. It also checks and
// skips some preceding pooled queries.
func (s *State) ReplayBlockWithContext(ctx context.Context, block *types.Block) (err error) {
	var (
		ierr   error
		lastsp uint64 // Last lastSeq
	)
	s.Lock()
	defer s.Unlock()
	for i, q := range block.QueryTxs {
		var query = &QueryTracker{Req: q.Request, Resp: &types.Response{Header: *q.Response}}
		lastsp = s.getSeq()
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
				return
			}
			if _, ierr = s.writeSingle(ctx, &v); ierr != nil {
				err = errors.Wrapf(ierr, "execute at %d:%d failed", i, j)
				return
			}
		}
		s.pool.enqueue(lastsp, query)
	}
	// Always try to commit after a block is successfully replayed
	s.tryCommit()
	// Remove duplicate failed queries from local pool
	for _, r := range block.FailedReqs {
		s.pool.removeFailed(r)
	}
	// Truncate pooled queries
	s.pool.truncate(lastsp)
	return
}

func (s *State) commit() (err error) {
	var (
		start = time.Now()

		lockAcquired, committed, poolCleaned, lockReleased time.Duration
	)

	defer func() {
		var fields = log.Fields{}
		fields["1#lockAcquired"] = float64(lockAcquired.Nanoseconds()) / 1000
		if committed > 0 {
			fields["2#committed"] = float64((committed - lockAcquired).Nanoseconds()) / 1000
		}
		if poolCleaned > 0 {
			fields["3#poolCleaned"] = float64((poolCleaned - committed).Nanoseconds()) / 1000
		}
		if lockReleased > 0 {
			fields["4#lockReleased"] = float64((lockReleased - poolCleaned).Nanoseconds()) / 1000
		}
		log.WithFields(fields).Debug("Commit duration stat (us)")
	}()

	s.Lock()
	defer func() {
		s.Unlock()
		lockReleased = time.Since(start)
	}()
	lockAcquired = time.Since(start)
	if err = s.uncCommit(); err != nil {
		log.WithError(err).Fatal("failed to commit")
	}
	if s.unc, err = s.strg.Writer().Begin(); err != nil {
		log.WithError(err).Fatal("failed to begin")
	}
	committed = time.Since(start)
	_ = s.pool.queries
	s.pool = newPool()
	poolCleaned = time.Since(start)
	return
}

// CommitEx commits the current transaction and returns all the pooled queries.
func (s *State) CommitEx() (failed []*types.Request, queries []*QueryTracker, err error) {
	return s.CommitExWithContext(context.Background())
}

// CommitExWithContext commits the current transaction and returns all the pooled queries
// with context.
func (s *State) CommitExWithContext(
	ctx context.Context) (failed []*types.Request, queries []*QueryTracker, err error,
) {
	var (
		start = time.Now()

		lockAcquired, committed, poolCleaned, lockReleased time.Duration
	)

	defer func() {
		var fields = log.Fields{}
		fields["1#lockAcquired"] = float64(lockAcquired.Nanoseconds()) / 1000
		if committed > 0 {
			fields["2#committed"] = float64((committed - lockAcquired).Nanoseconds()) / 1000
		}
		if poolCleaned > 0 {
			fields["3#poolCleaned"] = float64((poolCleaned - committed).Nanoseconds()) / 1000
		}
		if lockReleased > 0 {
			fields["4#lockReleased"] = float64((lockReleased - poolCleaned).Nanoseconds()) / 1000
		}
		log.WithFields(fields).Debug("Commit duration stat (us)")
	}()

	s.Lock()
	lockAcquired = time.Since(start)
	defer func() {
		s.Unlock()
		lockReleased = time.Since(start)
	}()
	// Always try to commit before the block is produced
	s.tryCommit()
	committed = time.Since(start)
	// Return pooled items and reset
	failed = s.pool.failedList()
	queries = s.pool.queries
	s.pool = newPool()
	poolCleaned = time.Since(start)
	return
}

func (s *State) tryCommit() {
	var err error
	if err = s.uncCommit(); err != nil {
		log.WithError(err).Fatal("failed to commit")
	}
	if s.unc, err = s.strg.Writer().Begin(); err != nil {
		log.WithError(err).Fatal("failed to begin")
	}
}

func (s *State) uncCommit() (err error) {
	if err = s.unc.Commit(); err != nil {
		return
	}
	// reset schema change flag
	atomic.StoreUint32(&s.hasSchemaChange, 0)
	atomic.StoreUint64(&s.lastCommitPoint, s.getSeq())
	return
}

func (s *State) uncRollback() (err error) {
	if err = s.unc.Rollback(); err != nil {
		return
	}
	// reset schema change flag
	atomic.StoreUint32(&s.hasSchemaChange, 0)
	return
}

func (s *State) getLocalTime() time.Time {
	return time.Now().UTC()
}

// Query does the query(ies) in req, pools the request and persists any change to
// the underlying storage.
func (s *State) Query(req *types.Request) (ref *QueryTracker, resp *types.Response, err error) {
	return s.QueryWithContext(context.Background(), req)
}

// QueryWithContext does the query(ies) in req, pools the request and persists any change to
// the underlying storage.
func (s *State) QueryWithContext(
	ctx context.Context, req *types.Request) (ref *QueryTracker, resp *types.Response, err error,
) {
	switch req.Header.QueryType {
	case types.ReadQuery:
		return s.readTx(ctx, req)
	case types.WriteQuery:
		return s.write(ctx, req)
	default:
		err = ErrInvalidRequest
	}
	return
}

// Replay replays a write log from other peer to replicate storage state.
func (s *State) Replay(req *types.Request, resp *types.Response) (err error) {
	return s.ReplayWithContext(context.Background(), req, resp)
}

// ReplayWithContext replays a write log from other peer to replicate storage state with context.
func (s *State) ReplayWithContext(
	ctx context.Context, req *types.Request, resp *types.Response) (err error,
) {
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
		return s.replay(ctx, req, resp)
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
	}).Info("xeno pool stats")
}
