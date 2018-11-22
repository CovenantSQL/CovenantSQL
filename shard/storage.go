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
	"reflect"

	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/CovenantSQL/go-sqlite3-encrypt"
)

var _ = log.Printf

const (
	// DBScheme defines the dsn scheme.
	DBScheme = "covenantsqlts"
	// DBSchemeAlias defines the alias dsn scheme.
	DBSchemeAlias = "cqlts"
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
	rawConn *sqlite3.SQLiteConn
}

type ShardingTx struct {
	c     *ShardingConn
	rawTx *sqlite3.SQLiteTx
}

type ShardingStmt struct {
	c       *ShardingConn
	rawStmt *sqlite3.SQLiteStmt
}

type ShardingRows struct {
	stmt    *ShardingStmt
	rawRows *sqlite3.SQLiteRows
}

// ShardingResult implements sql.Result.
type ShardingResult struct {
	rawResult *sqlite3.SQLiteResult
}

func (d *ShardingDriver) Open(dsn string) (conn driver.Conn, err error) {
	//TODO(auxten)
	if d.RawDriver == nil {
		d.RawDriver = &sqlite3.SQLiteDriver{}
	}
	rawConn, err := d.RawDriver.Open(dsn)
	if err == nil {
		conn = &ShardingConn{
			rawConn: rawConn.(*sqlite3.SQLiteConn),
		}
	}
	return
}

func (c *ShardingConn) BeginTx(ctx context.Context, opts driver.TxOptions) (tx driver.Tx, err error) {
	//TODO(auxten)
	rawTx, err := c.rawConn.BeginTx(ctx, opts)
	if err == nil {
		tx = &ShardingTx{
			rawTx: rawTx.(*sqlite3.SQLiteTx),
		}
	}
	return
}

func (c *ShardingConn) Close() (err error) {
	//TODO(auxten)
	err = c.rawConn.Close()
	return
}

func (c *ShardingConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (result driver.Result, err error) {
	//TODO(auxten)
	rawResult, err := c.rawConn.ExecContext(ctx, query, args)
	if err == nil {
		result = &ShardingResult{
			rawResult: rawResult.(*sqlite3.SQLiteResult),
		}
	}
	return
}

func (c *ShardingConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (rows driver.Rows, err error) {
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
	//TODO(auxten)
	rawStmt, err := c.rawConn.PrepareContext(ctx, query)
	if err == nil {
		stmt = &ShardingStmt{
			rawStmt: rawStmt.(*sqlite3.SQLiteStmt),
		}
	}
	return
}

func (s *ShardingStmt) Close() (err error) {
	//TODO(auxten)
	err = s.rawStmt.Close()
	return
}

// NumInput return a number of parameters.
func (s *ShardingStmt) NumInput() int {
	return s.rawStmt.NumInput()
}

func (s *ShardingStmt) ExecContext(ctx context.Context, args []driver.NamedValue) (result driver.Result, err error) {
	//TODO(auxten)
	rawResult, err := s.rawStmt.ExecContext(ctx, args)
	if err == nil {
		result = &ShardingResult{
			rawResult: rawResult.(*sqlite3.SQLiteResult),
		}
	}
	return
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
	//TODO(auxten)
	return tx.rawTx.Commit()
}

func (tx *ShardingTx) Rollback() error {
	//TODO(auxten)
	return tx.rawTx.Rollback()
}

// LastInsertId teturn last inserted ID.
func (r *ShardingResult) LastInsertId() (int64, error) {
	//TODO(auxten)
	return r.rawResult.LastInsertId()
}

// RowsAffected return how many rows affected.
func (r *ShardingResult) RowsAffected() (int64, error) {
	//TODO(auxten)
	return r.rawResult.RowsAffected()
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
