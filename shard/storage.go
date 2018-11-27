/*
 * Copyright 2018 The CovenantSQL Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package shard

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/CovenantSQL/go-sqlite3-encrypt"
	"github.com/CovenantSQL/sqlparser"
	"github.com/pkg/errors"
)

var _ = log.Printf

const (
	// DBScheme defines the dsn scheme.
	DBScheme = "covenantsqlts"
	// DBSchemeAlias defines the alias dsn scheme.
	DBSchemeAlias    = "cqlts"
	ShardSchemaToken = `/*SHARD*/`
)

func init() {
	d := new(ShardingDriver)
	sql.Register(DBScheme, d)
	sql.Register(DBSchemeAlias, d)
}

// ShardingDriver is a simple shard database on SQLite3 that implements Go's driver.Driver
type ShardingDriver struct {
	RawDriver *sqlite3.SQLiteDriver
}

type ShardingConn struct {
	sync.Mutex
	rawConn        *sqlite3.SQLiteConn
	rawDB          *sql.DB
	conf           map[string]*ShardingConf //map[tableName]*ShardingConf
	shardingTables sync.Map                 //map[shardTableName]true
}

type ShardingConf struct {
	ShardTableName sqlparser.TableName
	ShardColName   sqlparser.ColIdent
	ShardInterval  int64 // in seconds
	ShardStarttime time.Time
	ShardSchema    string
}

type ShardingTx struct {
	c     *ShardingConn
	rawTx *sqlite3.SQLiteTx
}

type ShardingStmt struct {
	c        *ShardingConn
	rawDB    *sql.DB
	rawQuery string
	rawStmt  *sqlite3.SQLiteStmt
}

type ShardingRows struct {
	stmt    *ShardingStmt
	rawRows *sqlite3.SQLiteRows
}

// ShardingResult implements sql.Result.
type ShardingResult struct {
	LastInsertIdi int64
	RowsAffectedi int64
	Err           error
	//rawResult     *sqlite3.SQLiteResult
}

func (d *ShardingDriver) Open(dsn string) (conn driver.Conn, err error) {
	//TODO(auxten)
	if d.RawDriver == nil {
		d.RawDriver = &sqlite3.SQLiteDriver{}
	}

	shardConn := &ShardingConn{}
	rawConn, err := d.RawDriver.Open(dsn)
	if err == nil {
		shardConn.rawConn = rawConn.(*sqlite3.SQLiteConn)
		shardConn.rawDB, err = sql.Open("sqlite3", dsn)
		if err == nil {
			err = shardConn.loadShardConfig()
			if err == nil {
				conn = shardConn
				return
			}
		}
	}

	log.Errorf("open shard db: %s failed: %v", dsn, err)
	return
}

func (c *ShardingConn) BeginTx(ctx context.Context, opts driver.TxOptions) (tx driver.Tx, err error) {
	//TODO(auxten) BeginTx on every shard table
	panic("transaction is to be implemented")
}

func (c *ShardingConn) loadShardConfig() (err error) {
	var (
		tablename string
		colname   string
		sinterval int64
		starttime time.Time
		sschema   string
	)
	// Create sharding meta table
	_, err = c.rawDB.Exec(`
		CREATE TABLE IF NOT EXISTS __SHARDING_META (
			tablename TEXT UNIQUE not null,
			colname TEXT not null,
			sinterval INTEGER,
			starttime TIMESTAMP,
			sschema TEXT not null
			);
		CREATE INDEX IF NOT EXISTS tableindex__SHARDING_META ON __SHARDING_META ( tablename );
		CREATE INDEX IF NOT EXISTS colindex__SHARDING_META ON __SHARDING_META ( colname );`)
	if err != nil {
		return errors.Wrapf(err, "create sharding meta table failed: %v", err)
	}

	c.conf = make(map[string]*ShardingConf)
	rows, err := c.rawDB.Query("select tablename, colname, sinterval, starttime, sschema from __SHARDING_META")
	if err != nil {
		return errors.Wrapf(err, "query sharding meta table")
	}
	defer rows.Close()
	if rows.Next() {
		err = rows.Scan(&tablename, &colname, &sinterval, &starttime, &sschema)
		if err != nil {
			return errors.Wrapf(err, "load shard conf failed: %v", err)
		}
		if sinterval < 10 {
			return errors.New("shard interval should greater or equal 10 seconds")
		}
		conf := &ShardingConf{
			ShardTableName: sqlparser.TableName{
				Name:      sqlparser.NewTableIdent(tablename),
				Qualifier: sqlparser.TableIdent{},
			},
			ShardColName:   sqlparser.NewColIdent(colname),
			ShardInterval:  sinterval,
			ShardStarttime: starttime,
			ShardSchema:    sschema,
		}
		c.conf[tablename] = conf
	}

	return
}

func (c *ShardingConn) SetShardConfig(
	tableName string, colName string, shardInterval int64, starttime int64, shardSchema string) (err error) {

	c.Lock()
	defer c.Unlock()
	//TODO(auxten) do some check if the column is suitable as sharding key
	_, err = c.rawDB.Exec(shardSchema)
	if err != nil {
		return
	}

	var rowCount int
	row := c.rawDB.QueryRow("select count(1) from " + tableName)
	err = row.Scan(&rowCount)
	if err == nil {
		// if row count is not 0, sharding conf can not be changed
		if rowCount == 0 {
			_, err = c.rawDB.Exec(`
            INSERT INTO __SHARDING_META(tablename,colname,sinterval,starttime,sschema)
			  VALUES(?,?,?,?,?)
			  ON CONFLICT(tablename) DO UPDATE SET
			    colname=excluded.colname,
			    sinterval=excluded.sinterval,
				starttime=excluded.starttime,
				sschema=excluded.sschema;`,
				tableName, colName, shardInterval, starttime, shardSchema)
			if err == nil {
				err = c.loadShardConfig()
				return
			}
		}
	}

	log.Errorf("set sharding conf for table %s failed: %v", tableName, err)
	return
}

func (c *ShardingConn) Close() (err error) {
	//TODO(auxten)
	err = c.rawConn.Close()
	return
}

func (c *ShardingConn) ExecContext(
	ctx context.Context,
	query string,
	args []driver.NamedValue) (result driver.Result, err error) {

	// SHARDCONFIG tableName colName shardInterval shardStartTime schema
	if strings.HasPrefix(query, "SHARDCONFIG") {
		shardConfig := strings.SplitN(query, " ", 6)
		log.Debugf("SHARDCONFIG %#v", shardConfig)
		if len(shardConfig) == 6 {
			var shardInterval, startTime int64
			shardInterval, err = strconv.ParseInt(shardConfig[3], 10, 64)
			if err != nil {
				return nil, errors.New("SHARDCONFIG 3rd args should in int64")
			}
			startTime, err = strconv.ParseInt(shardConfig[4], 10, 64)
			if err != nil {
				return nil, errors.New("SHARDCONFIG 4th args should in int64")
			}
			err = c.SetShardConfig(shardConfig[1], shardConfig[2], shardInterval, startTime, shardConfig[5])
			if err != nil {
				log.Errorf("set shard config failed: %v", err)
				return
			}
			return &ShardingResult{}, nil
		} else {
			return nil, errors.New("SHARDCONFIG should have 5 args")
		}
	}

	var (
		tokenizer   = sqlparser.NewStringTokenizer(query)
		lastPos     int
		lastArgs    int
		singleQuery string
		queries     []string
		stmts       []sqlparser.Statement
	)

	for {
		var stmt sqlparser.Statement
		stmt, err = sqlparser.ParseNext(tokenizer)
		if err != nil && err != io.EOF {
			return
		}

		if err == io.EOF {
			err = nil
			break
		}

		singleQuery = query[lastPos : tokenizer.Position-1]
		lastPos = tokenizer.Position
		queries = append(queries, singleQuery)
		stmts = append(stmts, stmt)
		//q.Q(stmt, args[lastArgs:lastArgs+numInputs], stmt)
	}

	// Build query plans
	plans := make([]*Plan, len(queries))
	argss := make([][]driver.NamedValue, len(queries))
	for i, q := range queries {
		var (
			plan       *Plan
			sqliteStmt driver.Stmt
			numInputs  int
			j          int
			a          driver.NamedValue
		)

		insertStmt, ok := stmts[i].(*sqlparser.Insert)
		if ok {
			sqliteStmt, err = c.rawConn.Prepare(q)
			if err == nil {
				numInputs = sqliteStmt.NumInput()
				sqliteStmt.Close()
			}
			// Copy to fix args ordinal error
			newArgs := make([]driver.NamedValue, numInputs)
			for j, a = range args[lastArgs : lastArgs+numInputs] {
				newArgs[j] = driver.NamedValue{
					Name:    a.Name,
					Ordinal: j + 1,
					Value:   a.Value,
				}
			}
			argss[i] = newArgs
			lastArgs += numInputs

			plan, err = BuildFromStmt(q, argss[i], insertStmt, c)
			if err != nil {
				log.Errorf("build shard statement for %v failed: %v", q, err)
				return
			}
		} else {
			// FIXME(auxten) if contains any statement other than sqlparser.Insert, we just
			// execute it for test
			plan = &Plan{
				Original: q,
				Instructions: &BasePrimitive{
					query:   q,
					args:    args,
					rawConn: c.rawConn,
				},
			}
		}
		plans[i] = plan
	}

	//q.Q(plans)
	// Execute query plans
	shardResult := &ShardingResult{}

	for _, p := range plans {
		var r driver.Result
		r, err = p.Instructions.ExecContext(ctx)
		if err != nil {
			return nil,
				errors.Wrapf(err, "exec plan %v for %s failed", p, p.Original)
		} else {
			var ra int64
			ra, shardResult.Err = r.RowsAffected()
			shardResult.RowsAffectedi += ra
			shardResult.LastInsertIdi, _ = r.LastInsertId()
		}
	}

	return shardResult, err
}

func (c *ShardingConn) QueryContext(
	ctx context.Context,
	query string,
	args []driver.NamedValue) (rows driver.Rows, err error) {
	//TODO(auxten)
	rawRows, err := c.rawConn.QueryContext(ctx, query, args)
	if err == nil {
		rows = &ShardingRows{
			rawRows: rawRows.(*sqlite3.SQLiteRows),
		}
	}
	return
}

func (c *ShardingConn) PrepareContext(ctx context.Context, query string) (stmt driver.Stmt, err error) {
	rawStmt, err := c.rawConn.PrepareContext(ctx, query)
	if err == nil {
		stmt = &ShardingStmt{
			c:        c,
			rawStmt:  rawStmt.(*sqlite3.SQLiteStmt),
			rawQuery: query,
			rawDB:    c.rawDB,
		}
	}
	return
}

func (s *ShardingStmt) Close() (err error) {
	err = s.rawStmt.Close()
	return
}

// NumInput return a number of parameters.
func (s *ShardingStmt) NumInput() int {
	return s.rawStmt.NumInput()
}

func (s *ShardingStmt) ExecContext(ctx context.Context, args []driver.NamedValue) (result driver.Result, err error) {
	return s.c.ExecContext(ctx, s.rawQuery, args)
}

func (s *ShardingStmt) QueryContext(ctx context.Context, args []driver.NamedValue) (rows driver.Rows, err error) {
	//TODO(auxten)
	rawRows, err := s.rawStmt.QueryContext(ctx, args)
	if err == nil {
		rows = &ShardingRows{
			rawRows: rawRows.(*sqlite3.SQLiteRows),
		}
	}
	return
}

func (tx *ShardingTx) Commit() error {
	//TODO(auxten) BeginTx on every shard table
	panic("transaction is to be implemented")
}

func (tx *ShardingTx) Rollback() error {
	//TODO(auxten) BeginTx on every shard table
	panic("transaction is to be implemented")
}

// LastInsertId teturn last inserted ID.
func (r *ShardingResult) LastInsertId() (int64, error) {
	//TODO(auxten)
	return r.LastInsertIdi, nil
}

// RowsAffected return how many rows affected.
func (r *ShardingResult) RowsAffected() (int64, error) {
	//TODO(auxten)
	return r.RowsAffectedi, nil
}

func (r *ShardingRows) Columns() []string {
	//TODO(auxten)
	return r.rawRows.Columns()
}

func (r *ShardingRows) Close() (err error) {
	//TODO(auxten)
	err = r.rawRows.Close()
	return
}

func (r *ShardingRows) Next(dest []driver.Value) error {
	//TODO(auxten)
	return r.rawRows.Next(dest)
}

// ColumnTypeDatabaseTypeName implement RowsColumnTypeDatabaseTypeName.
func (r *ShardingRows) ColumnTypeDatabaseTypeName(i int) string {
	return r.rawRows.ColumnTypeDatabaseTypeName(i)
}

// ColumnTypeNullable implement RowsColumnTypeNullable.
func (r *ShardingRows) ColumnTypeNullable(i int) (nullable, ok bool) {
	return r.rawRows.ColumnTypeNullable(i)
}

// ColumnTypeScanType implement RowsColumnTypeScanType.
func (r *ShardingRows) ColumnTypeScanType(i int) reflect.Type {
	return r.rawRows.ColumnTypeScanType(i)
}

/************************* Deprecated interface func below *************************/
func (c *ShardingConn) Begin() (driver.Tx, error) {
	panic("ConnBeginTx was not called")
}

func (c *ShardingConn) Exec(query string, args []driver.Value) (driver.Result, error) {
	panic("ExecContext was not called")
}

func (c *ShardingConn) Query(query string, args []driver.Value) (driver.Rows, error) {
	panic("QueryContext was not called")
}

func (c *ShardingConn) Prepare(query string) (driver.Stmt, error) {
	panic("PrepareContext was not called")
}

func (s *ShardingStmt) Exec(args []driver.Value) (driver.Result, error) {
	panic("ExecContext was not called")
}

func (s *ShardingStmt) Query(args []driver.Value) (driver.Rows, error) {
	panic("QueryContext was not called")
}

/************************* Deprecated interface func above *************************/
