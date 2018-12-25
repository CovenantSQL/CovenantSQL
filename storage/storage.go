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

package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/CovenantSQL/CovenantSQL/twopc"
	"github.com/CovenantSQL/CovenantSQL/utils/log"

	// Register CovenantSQL/go-sqlite3-encrypt engine.
	_ "github.com/CovenantSQL/go-sqlite3-encrypt"
)

var (
	index = struct {
		sync.Mutex
		db map[string]*sql.DB
	}{
		db: make(map[string]*sql.DB),
	}
)

// Query represents the single query of sqlite.
type Query struct {
	Pattern string
	Args    []sql.NamedArg
}

// ExecLog represents the execution log of sqlite.
type ExecLog struct {
	ConnectionID uint64
	SeqNo        uint64
	Timestamp    int64
	Queries      []Query
}

// ExecResult represents the execution result of sqlite.
type ExecResult struct {
	LastInsertID int64
	RowsAffected int64
}

func openDB(dsn string) (db *sql.DB, err error) {
	// Rebuild DSN.
	d, err := NewDSN(dsn)

	if err != nil {
		return nil, err
	}

	d.AddParam("_journal_mode", "WAL")
	d.AddParam("_synchronous", "NORMAL")
	fdsn := d.Format()

	fn := d.GetFileName()
	mode, _ := d.GetParam("mode")
	cache, _ := d.GetParam("cache")

	if (fn == ":memory:" || mode == "memory") && cache != "shared" {
		// Return a new DB instance if it's in memory and private.
		db, err = sql.Open("sqlite3", fdsn)
		return
	}

	index.Lock()
	db, ok := index.db[d.filename]
	index.Unlock()

	if !ok {
		db, err = sql.Open("sqlite3", fdsn)

		if err != nil {
			return nil, err
		}

		index.Lock()
		index.db[d.filename] = db
		index.Unlock()
	}

	return
}

// TxID represents a transaction ID.
type TxID struct {
	ConnectionID uint64
	SeqNo        uint64
	Timestamp    int64
}

func equalTxID(x, y *TxID) bool {
	return x.ConnectionID == y.ConnectionID && x.SeqNo == y.SeqNo && x.Timestamp == y.Timestamp
}

// Storage represents a underlying storage implementation based on sqlite3.
type Storage struct {
	sync.Mutex
	dsn     string
	db      *sql.DB
	tx      *sql.Tx // Current tx
	id      TxID
	queries []Query
}

// New returns a new storage connected by dsn.
func New(dsn string) (st *Storage, err error) {
	db, err := openDB(dsn)

	if err != nil {
		return
	}

	return &Storage{
		dsn: dsn,
		db:  db,
	}, nil
}

// Prepare implements prepare method of two-phase commit worker.
func (s *Storage) Prepare(ctx context.Context, wb twopc.WriteBatch) (err error) {
	el, ok := wb.(*ExecLog)

	if !ok {
		return errors.New("unexpected WriteBatch type")
	}

	s.Lock()
	defer s.Unlock()

	if s.tx != nil {
		if equalTxID(&s.id, &TxID{el.ConnectionID, el.SeqNo, el.Timestamp}) {
			s.queries = el.Queries
			return nil
		}

		return fmt.Errorf("twopc: inconsistent state, currently in tx: "+
			"conn = %d, seq = %d, time = %d", s.id.ConnectionID, s.id.SeqNo, s.id.Timestamp)
	}

	s.tx, err = s.db.BeginTx(ctx, nil)

	if err != nil {
		return
	}

	s.id = TxID{el.ConnectionID, el.SeqNo, el.Timestamp}
	s.queries = el.Queries

	return nil
}

// Commit implements commit method of two-phase commit worker.
func (s *Storage) Commit(ctx context.Context, wb twopc.WriteBatch) (result interface{}, err error) {
	el, ok := wb.(*ExecLog)

	if !ok {
		err = errors.New("unexpected WriteBatch type")
		return
	}

	s.Lock()
	defer s.Unlock()

	if s.tx != nil {
		if equalTxID(&s.id, &TxID{el.ConnectionID, el.SeqNo, el.Timestamp}) {
			// get last insert id and affected rows result
			execResult := ExecResult{}

			for _, q := range s.queries {
				// convert arguments types
				args := make([]interface{}, len(q.Args))

				for i, v := range q.Args {
					args[i] = v
				}

				var res sql.Result
				res, err = s.tx.ExecContext(ctx, q.Pattern, args...)

				if err != nil {
					log.WithError(err).Debug("commit query failed")
					s.tx.Rollback()
					s.tx = nil
					s.queries = nil
					return
				}

				lastInsertID, _ := res.LastInsertId()
				rowsAffected, _ := res.RowsAffected()

				execResult.LastInsertID = lastInsertID
				execResult.RowsAffected += rowsAffected
			}

			s.tx.Commit()
			s.tx = nil
			s.queries = nil
			result = execResult

			return
		}

		err = fmt.Errorf("twopc: inconsistent state, currently in tx: "+
			"conn = %d, seq = %d, time = %d", s.id.ConnectionID, s.id.SeqNo, s.id.Timestamp)
		return
	}

	err = errors.New("twopc: tx not prepared")
	return
}

// Rollback implements rollback method of two-phase commit worker.
func (s *Storage) Rollback(ctx context.Context, wb twopc.WriteBatch) (err error) {
	el, ok := wb.(*ExecLog)

	if !ok {
		return errors.New("unexpected WriteBatch type")
	}

	s.Lock()
	defer s.Unlock()

	if !equalTxID(&s.id, &TxID{el.ConnectionID, el.SeqNo, el.Timestamp}) {
		return fmt.Errorf("twopc: inconsistent state, currently in tx: "+
			"conn = %d, seq = %d, time = %d", s.id.ConnectionID, s.id.SeqNo, s.id.Timestamp)
	}

	if s.tx != nil {
		s.tx.Rollback()
		s.tx = nil
		s.queries = nil
	}

	return nil
}

// Query implements read-only query feature.
func (s *Storage) Query(ctx context.Context, queries []Query) (columns []string, types []string,
	data [][]interface{}, err error) {
	data = make([][]interface{}, 0)

	if len(queries) == 0 {
		return
	}

	var tx *sql.Tx
	var txOptions = &sql.TxOptions{
		ReadOnly: true,
	}

	if tx, err = s.db.BeginTx(ctx, txOptions); err != nil {
		return
	}

	// always rollback on complete
	defer tx.Rollback()

	q := queries[len(queries)-1]

	// convert arguments types
	args := make([]interface{}, len(q.Args))

	for i, v := range q.Args {
		args[i] = v
	}

	var rows *sql.Rows
	if rows, err = tx.Query(q.Pattern, args...); err != nil {
		return
	}

	// free result set
	defer rows.Close()

	// get rows meta
	if columns, err = rows.Columns(); err != nil {
		return
	}

	// if there is empty columns, treat result as empty
	if len(columns) == 0 {
		return
	}

	// get types meta
	if types, err = s.transformColumnTypes(rows.ColumnTypes()); err != nil {
		return
	}

	rs := newRowScanner(len(columns))

	for rows.Next() {
		err = rows.Scan(rs.ScanArgs()...)
		if err != nil {
			return
		}

		data = append(data, rs.GetRow())
	}

	err = rows.Err()
	return
}

// Exec implements write query feature.
func (s *Storage) Exec(ctx context.Context, queries []Query) (result ExecResult, err error) {
	if len(queries) == 0 {
		return
	}

	var tx *sql.Tx
	var txOptions = &sql.TxOptions{
		ReadOnly: false,
	}

	if tx, err = s.db.BeginTx(ctx, txOptions); err != nil {
		return
	}

	defer tx.Rollback()

	for _, q := range queries {
		// convert arguments types
		args := make([]interface{}, len(q.Args))

		for i, v := range q.Args {
			args[i] = v
		}

		var r sql.Result
		if r, err = tx.Exec(q.Pattern, args...); err != nil {
			log.WithError(err).Debug("execute query failed")
			return
		}

		var affected int64
		affected, _ = r.RowsAffected()
		result.RowsAffected += affected
		result.LastInsertID, _ = r.LastInsertId()
	}

	tx.Commit()

	return
}

// Close implements database safe close feature.
func (s *Storage) Close() (err error) {
	d, err := NewDSN(s.dsn)
	if err != nil {
		return
	}

	index.Lock()
	defer index.Unlock()
	delete(index.db, d.filename)
	return s.db.Close()
}

func (s *Storage) transformColumnTypes(columnTypes []*sql.ColumnType, e error) (types []string, err error) {
	if e != nil {
		err = e
		return
	}

	types = make([]string, len(columnTypes))

	for i, c := range columnTypes {
		types[i] = c.DatabaseTypeName()
	}

	return
}

// golang does trick convert, use rowScanner to return the original result type in sqlite3 driver
type rowScanner struct {
	fieldCnt int
	column   int           // current column
	fields   []interface{} // temp fields
	scanArgs []interface{}
}

func newRowScanner(fieldCnt int) (s *rowScanner) {
	s = &rowScanner{
		fieldCnt: fieldCnt,
		column:   0,
		fields:   make([]interface{}, fieldCnt),
		scanArgs: make([]interface{}, fieldCnt),
	}

	for i := 0; i != fieldCnt; i++ {
		s.scanArgs[i] = s
	}

	return
}

func (s *rowScanner) Scan(src interface{}) error {
	if s.fieldCnt <= s.column {
		// read complete
		return io.EOF
	}

	s.fields[s.column] = src
	s.column++

	return nil
}

func (s *rowScanner) GetRow() []interface{} {
	return s.fields
}

func (s *rowScanner) ScanArgs() []interface{} {
	// reset
	s.column = 0
	s.fields = make([]interface{}, s.fieldCnt)
	return s.scanArgs
}
