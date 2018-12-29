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
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/CovenantSQL/CovenantSQL/utils/log"
	sqlite3 "github.com/CovenantSQL/go-sqlite3-encrypt"
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
	ShardStartTime time.Time
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
	plan    *Plan
	stmt    *ShardingStmt
	rawRows *sql.Rows
}

// ShardingResult implements sql.Result.
type ShardingResult struct {
	LastInsertIdi int64
	RowsAffectedi int64
	Err           error
	//rawResult     *sqlite3.SQLiteResult
}

func (d *ShardingDriver) Open(dsn string) (conn driver.Conn, err error) {
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

func (sc *ShardingConn) BeginTx(ctx context.Context, opts driver.TxOptions) (tx driver.Tx, err error) {
	//TODO(auxten) BeginTx on every shard table
	panic("transaction is to be implemented")
}

func (sc *ShardingConn) loadShardConfig() (err error) {
	var (
		tableName string
		colName   string
		sInterval int64
		startTime time.Time
		sSchema   string
	)
	// Create sharding meta table
	_, err = sc.rawDB.Exec(`
		CREATE TABLE IF NOT EXISTS __SHARDING_META (
			tableName TEXT UNIQUE not null,
			colName TEXT not null,
			sInterval INTEGER,
			startTime TIMESTAMP,
			sSchema TEXT not null
			);
		CREATE INDEX IF NOT EXISTS tableindex__SHARDING_META ON __SHARDING_META ( tableName );
		CREATE INDEX IF NOT EXISTS colindex__SHARDING_META ON __SHARDING_META ( colName );`)
	if err != nil {
		return errors.Wrapf(err, "create sharding meta table failed: %v", err)
	}

	sc.conf = make(map[string]*ShardingConf)
	rows, err := sc.rawDB.Query("select tableName, colName, sInterval, startTime, sSchema from __SHARDING_META")
	if err != nil {
		return errors.Wrapf(err, "query sharding meta table")
	}
	defer rows.Close()
	if rows.Next() {
		err = rows.Scan(&tableName, &colName, &sInterval, &startTime, &sSchema)
		if err != nil {
			return errors.Wrapf(err, "load shard conf failed: %v", err)
		}
		if sInterval < 10 {
			return errors.New("shard interval should greater or equal 10 seconds")
		}
		conf := &ShardingConf{
			ShardTableName: sqlparser.TableName{
				Name:      sqlparser.NewTableIdent(tableName),
				Qualifier: sqlparser.TableIdent{},
			},
			ShardColName:   sqlparser.NewColIdent(colName),
			ShardInterval:  sInterval,
			ShardStartTime: startTime,
			ShardSchema:    sSchema,
		}
		sc.conf[tableName] = conf
	}

	return
}

func (sc *ShardingConn) SetShardConfig(
	tableName string, colName string, shardInterval int64, starttime int64, shardSchema string) (result sql.Result, err error) {

	sc.Lock()
	defer sc.Unlock()
	//TODO(auxten) do some check if the column is suitable as sharding key
	_, err = sc.rawDB.Exec(shardSchema)
	if err != nil {
		return
	}

	var rowCount int
	row := sc.rawDB.QueryRow("select count(1) from " + tableName)
	err = row.Scan(&rowCount)
	if err == nil {
		// if row count is not 0, sharding conf can not be changed
		if rowCount == 0 {
			result, err = sc.rawDB.Exec(`
            INSERT INTO __SHARDING_META(tablename,colname,sinterval,starttime,sschema)
			  VALUES(?,?,?,?,?)
			  ON CONFLICT(tablename) DO UPDATE SET
			    colname=excluded.colname,
			    sinterval=excluded.sinterval,
				starttime=excluded.starttime,
				sschema=excluded.sschema;`,
				tableName, colName, shardInterval, starttime, shardSchema)
			if err == nil {
				err = sc.loadShardConfig()
				return
			}
		}
	}

	log.Errorf("set sharding conf for table %s failed: %v", tableName, err)
	return
}

func (sc *ShardingConn) Close() (err error) {
	sc.rawDB.Close()
	err = sc.rawConn.Close()
	return
}

func (sc *ShardingConn) ExecContext(
	ctx context.Context,
	query string,
	args []driver.NamedValue) (result driver.Result, err error) {

	// trim space for easier life
	query = strings.TrimSpace(query)

	// SHARDCONFIG tableName colName shardInterval shardStartTime schema
	if strings.HasPrefix(query, "SHARDCONFIG") {
		shardConfig := strings.SplitN(query, " ", 6)
		log.Debugf("SHARDCONFIG %#v", shardConfig)
		if len(shardConfig) == 6 {
			var shardInterval, startTime int64
			var r sql.Result
			shardInterval, err = strconv.ParseInt(shardConfig[3], 10, 64)
			if err != nil {
				return nil, errors.New("SHARDCONFIG 3rd args should in int64")
			}
			startTime, err = strconv.ParseInt(shardConfig[4], 10, 64)
			if err != nil {
				return nil, errors.New("SHARDCONFIG 4th args should in int64")
			}
			r, err = sc.SetShardConfig(shardConfig[1], shardConfig[2], shardInterval, startTime, shardConfig[5])
			if err != nil {
				log.Errorf("set shard config failed: %v", err)
				return
			}
			li, _ := r.LastInsertId()
			ra, _ := r.LastInsertId()
			return &ShardingResult{
				LastInsertIdi: li,
				RowsAffectedi: ra,
				Err:           nil,
			}, nil
		} else {
			return nil, errors.New("SHARDCONFIG should have 5 args")
		}
	}

	var (
		queries []string
		stmts   []sqlparser.Statement
		argss   [][]driver.NamedValue
	)

	queries, stmts, argss, err = sc.splitQueryArgs(query, args)
	if err != nil {
		return
	}

	// Build query plans
	plans := make([]*Plan, len(queries))
	for i, q := range queries {
		var plan *Plan
		plan, err = BuildFromStmt(q, argss[i], stmts[i], sc)
		if err != nil {
			return nil,
				errors.Wrapf(err, "build shard statement for %v failed", q)
		}
		plans[i] = plan
	}

	//q.Q(plans)
	// Execute query plans
	shardResult := &ShardingResult{}

	for _, p := range plans {
		var r driver.Result
		r, err = p.Instructions.ExecContext(ctx, nil)
		if err != nil {
			return nil,
				errors.Wrapf(err, "exec plan %#v for %s %#v failed",
					p, p.OriginQuery, p.OriginArgs)
		} else {
			var ra int64
			ra, shardResult.Err = r.RowsAffected()
			shardResult.RowsAffectedi = ra
			shardResult.LastInsertIdi, _ = r.LastInsertId()
		}
	}

	return shardResult, err
}

func (sc *ShardingConn) QueryContext(
	ctx context.Context,
	query string,
	args []driver.NamedValue) (rows driver.Rows, err error) {

	// trim space for easier life
	query = strings.TrimSpace(query)

	var (
		queries []string
		stmts   []sqlparser.Statement
		argss   [][]driver.NamedValue
		plan    *Plan
		sqlRows *sql.Rows
	)

	queries, stmts, argss, err = sc.splitQueryArgs(query, args)
	if err != nil {
		return
	}

	//q.Q(queries, stmts, argss)

	// Only build query plan for the last query
	queryCount := len(queries)
	qLast := queries[queryCount-1]
	aLast := argss[queryCount-1]
	sLast := stmts[queryCount-1]
	plan, err = BuildFromStmt(qLast, aLast, sLast, sc)
	if err != nil {
		return nil,
			errors.Wrapf(err, "build shard statement for %v failed", qLast)
	}

	switch plan.Instructions.(type) {
	case *BasePrimitive:
		rows, err = plan.Instructions.(*BasePrimitive).QueryContext(ctx)
	case *Select:
		err = plan.PrepareMergeTable(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "filling merge table")
		}
		sqlRows, err = plan.QueryContext(ctx)
		if err != nil {
			return nil,
				errors.Wrapf(err, "exec plan %v for %s %#v failed",
					plan, plan.OriginQuery, plan.OriginArgs)
		}
		rows = &ShardingRows{
			plan:    plan,
			stmt:    nil,
			rawRows: sqlRows,
		}

	default:
		panic("unknown plan instruction type")
	}

	return
}

func (sc *ShardingConn) PrepareContext(ctx context.Context, query string) (stmt driver.Stmt, err error) {
	rawStmt, err := sc.rawConn.PrepareContext(ctx, query)
	if err == nil {
		stmt = &ShardingStmt{
			c:        sc,
			rawStmt:  rawStmt.(*sqlite3.SQLiteStmt),
			rawQuery: query,
			rawDB:    sc.rawDB,
		}
	}
	return
}

func (sc *ShardingConn) getTableShards(tableName string) (shards []string, err error) {
	q := fmt.Sprintf(`select name from sqlite_master where name like "%s%s%%";`,
		tableName, SHARD_SUFFIX)
	rows, err := sc.rawDB.Query(q)
	if err != nil {
		err = errors.Wrapf(err, "get table shards by: %s", q)
		return
	}

	shards = make([]string, 0, 16)
	defer rows.Close()
	for rows.Next() {
		var table string
		err = rows.Scan(&table)
		if err != nil {
			err = errors.Wrapf(err, "get table shards by: %s", q)
			return
		}
		shards = append(shards, table)
	}
	err = rows.Err()
	if err != nil {
		err = errors.Wrapf(err, "get table shards by: %s", q)
	}
	return
}

func (sc *ShardingConn) getTableSchema(tableName string) (schema string, err error) {
	//TODO(auxten): check and add "IF NOT EXISTS"
	if conf, ok := sc.conf[tableName]; ok {
		if strings.Contains(conf.ShardSchema, ShardSchemaToken) {
			return conf.ShardSchema, nil
		} else {
			return "", errors.Errorf("not found '%s' in schema: %s", ShardSchemaToken, conf.ShardSchema)
		}
	} else {
		return "", errors.Errorf("not found schema for table: %s", tableName)
	}
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
	return s.c.QueryContext(ctx, s.rawQuery, args)
}

func (sc *ShardingConn) splitQueryArgs(query string, args []driver.NamedValue) (
	queries []string, stmts []sqlparser.Statement, argss [][]driver.NamedValue, err error) {
	var (
		tokenizer   = sqlparser.NewStringTokenizer(query)
		lastPos     int
		singleQuery string
		sqliteStmt  driver.Stmt
		numInputs   int
		j           int
		lastArgs    int
		a           driver.NamedValue
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

		singleQuery = strings.TrimSpace(query[lastPos : tokenizer.Position-1])
		lastPos = tokenizer.Position
		queries = append(queries, singleQuery)
		stmts = append(stmts, stmt)
	}

	argss = make([][]driver.NamedValue, len(queries))

	for i, q := range queries {
		sqliteStmt, err = sc.rawConn.Prepare(q)
		if err == nil {
			numInputs = sqliteStmt.NumInput()
			sqliteStmt.Close()
			// Copy to fix args ordinal error
			newArgs := make([]driver.NamedValue, numInputs)
			if len(args) < lastArgs+numInputs {
				return nil, nil, nil,
					errors.Wrapf(err, "not enough args to execute query: got %d", len(args))
			}
			for j, a = range args[lastArgs : lastArgs+numInputs] {
				newArgs[j] = driver.NamedValue{
					Name:    a.Name,
					Ordinal: j + 1,
					Value:   a.Value,
				}
			}
			argss[i] = newArgs
			lastArgs += numInputs
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
	rows, _ := r.rawRows.Columns()
	return rows
}

func (r *ShardingRows) Close() (err error) {
	if r.rawRows != nil {
		err = r.rawRows.Close()
	}
	if r.plan != nil {
		_ = r.plan.destroyMergeTable()
	}
	return
}

func (r *ShardingRows) Next(dest []driver.Value) error {
	if r.rawRows.Next() {
		s := make([]interface{}, len(dest))
		for i, _ := range dest {
			s[i] = &dest[i]
		}
		return r.rawRows.Scan(s...)
	} else {
		return io.EOF
	}
}

/************************* Deprecated interface func below *************************/
func (sc *ShardingConn) Begin() (driver.Tx, error) {
	panic("ConnBeginTx was not called")
}

func (sc *ShardingConn) Exec(query string, args []driver.Value) (driver.Result, error) {
	panic("ExecContext was not called")
}

func (sc *ShardingConn) Query(query string, args []driver.Value) (driver.Rows, error) {
	panic("QueryContext was not called")
}

func (sc *ShardingConn) Prepare(query string) (driver.Stmt, error) {
	panic("PrepareContext was not called")
}

func (s *ShardingStmt) Exec(args []driver.Value) (driver.Result, error) {
	panic("ExecContext was not called")
}

func (s *ShardingStmt) Query(args []driver.Value) (driver.Rows, error) {
	panic("QueryContext was not called")
}

/************************* Deprecated interface func above *************************/
