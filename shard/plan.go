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
	"strings"

	"github.com/pkg/errors"
)

const (
	TempTableRowBufSize = 16
	MaxMergeRowBufSize  = 1024
)

// Plan represents the execution strategy for a given query.
// For now it's a simple wrapper around the real instructions.
// An instruction (aka Primitive) is typically a tree where
// each node does its part by combining the results of the
// sub-nodes.
type Plan struct {
	// OriginQuery is the original query.
	OriginQuery string
	OriginArgs  []driver.NamedValue
	// Instructions contains the instructions needed to
	// fulfil the query.
	Instructions Primitive
	// TempDB is used to merge shard results
	TempDB *sql.DB
	c      *ShardingConn
}

func (plan *Plan) PrepareMergeTable(ctx context.Context) (err error) {
	instructions := plan.Instructions.(*Select).Instructions
	if len(instructions) == 0 {
		return errors.New("empty plan instruction set")
	}
	// create temp table
	schema, err := plan.c.getTableSchema(plan.Instructions.(*Select).shardTable)
	if err != nil {
		return errors.Wrap(err, "get shard table schema")
	}
	err = plan.CreateMergeTable(schema)
	if err != nil {
		return errors.Wrap(err, "create merge table")
	}
	// parallel select, scan, process "auto increase id", insert

	// IT'S IMPORTANT to make errCh buf fit the instructionsSize + 1, otherwise
	//  deadlock may caused by multiple selector & merger blocking at putting error into errCh
	errCh := make(chan error, len(instructions)+1)
	rowChs := make([]<-chan []interface{}, len(instructions))
	stopCh := make(chan struct{})

	var (
		lastQuery              string
		colNames, lastColNames []string
		rowCh                  <-chan []interface{}
	)
	for i, ins := range instructions {
		colNames, _, rowCh, err = plan.selector(ctx, errCh, stopCh, ins, TempTableRowBufSize)
		if err != nil {
			err = errors.Wrapf(err,
				`query single select "%s", args %#v`, ins.query, ins.namedArgs)
			return
		}
		if i != 0 && !isColNamesEqual(lastColNames, colNames) {
			return errors.Errorf(`column names not match between shard queries: "%s", "%s"`,
				lastQuery, ins.query)
		}
		lastQuery = ins.query
		lastColNames = colNames
		rowChs[i] = rowCh
	}

	err = plan.merger(ctx, errCh, colNames, stopCh, rowChs)
	if err != nil {
		return errors.Wrap(err, "select plan merger")
	}

	return
}

func (plan *Plan) QueryContext(ctx context.Context) (*sql.Rows, error) {
	s := toUserSpaceArgs(plan.OriginArgs)
	return plan.TempDB.QueryContext(ctx, plan.OriginQuery, s...)
}

func (plan *Plan) destroyMergeTable() (err error) {
	if plan.TempDB != nil {
		//TODO(auxten) drop all tables
		plan.TempDB.Close()
	}
	return
}

func (plan *Plan) CreateMergeTable(originSchema string) (err error) {
	plan.TempDB, err = sql.Open("sqlite3", ":memory:")
	if err != nil {
		return errors.Wrap(err, "create temp mem database")
	}
	_, err = plan.TempDB.Exec(originSchema)
	if err != nil {
		_ = plan.TempDB.Close()
		plan.TempDB = nil
		return errors.Wrap(err, "create temp mem table")
	}
	return
}

// start row insertion to temp table
func (plan *Plan) merger(ctx context.Context,
	errCh chan error,
	colNames []string,
	stopCh chan struct{},
	rowChs []<-chan []interface{}) (err error) {

	sel := plan.Instructions.(*Select)
	insCount := len(sel.Instructions)
	bufSize := insCount * TempTableRowBufSize
	if bufSize >= MaxMergeRowBufSize {
		bufSize = MaxMergeRowBufSize
	}

	go func() {
		var (
			err             error
			workingSelector = insCount
		)
		for {
			select {
			case <-ctx.Done():
				errCh <- errors.Wrapf(ctx.Err(),
					"merger context ends")
				return
			case <-stopCh:
				return
			default:
				if workingSelector == 0 {
					// all selectors succeed
					close(errCh)
					return
				}
				// drain all the selector and do a batch insert
				rowBuf := make([][]interface{}, 0, bufSize)
				for i, rowCh := range rowChs {
					if rowCh != nil {
						// iterate all row chan
					drainChLoop:
						for {
							// drain the row chan
							select {
							case row, got := <-rowCh:
								if !got { //closed and empty
									// set this chan to nil to omit in the next loop
									rowChs[i] = nil
									workingSelector--
									break drainChLoop
								} else {
									rowBuf = append(rowBuf, row)
								}
							case <-stopCh:
								return
							default:
								break drainChLoop
							}
						}
					}
				}
				err = plan.mergeInsert(ctx, colNames, rowBuf)
				if err != nil {
					errCh <- errors.Wrapf(ctx.Err(),
						"insert rows to temp table")
					return
				}
			}
		}
	}()

	// wait for merger worker return
	var anyErr bool
	err, anyErr = <-errCh
	if anyErr {
		// force stop all
		close(stopCh)
	}
	// errCh closed and empty means temp table preparation succeed
	return

}

func (plan *Plan) mergeInsert(ctx context.Context,
	colNames []string,
	rows [][]interface{}) (err error) {

	//TODO(auxten) maybe bulk insert will be faster
	var (
		stmt *sql.Stmt
	)

	sel := plan.Instructions.(*Select)

	columnsStr := strings.Join(colNames, ",")
	placeholderStr := strings.Repeat("?,", len(colNames))
	placeholderStr = placeholderStr[:len(placeholderStr)-1]
	insertSQL := fmt.Sprintf(`insert into %s (%s) values (%s);`,
		sel.shardTable, columnsStr, placeholderStr)
	stmt, err = plan.TempDB.PrepareContext(ctx, insertSQL)
	if err != nil {
		err = errors.Wrapf(err, "prepare: %s", insertSQL)
		return
	}
	defer stmt.Close()
	for _, r := range rows {
		_, err = stmt.ExecContext(ctx, r...)
		if err != nil {
			err = errors.Wrapf(err, "execute: %s with args %v", insertSQL, r)
			return
		}
	}

	return
}

// selector start worker to do the select in shard tables. If all preparation complete successfully, return err == nil
// and a rowCh to return rows got from shard tables.
// After a return, worker will be running as a goroutine
// if all done successfully rowCh will be closed, errCh left open
//   otherwise, put error into errCh, and errCh, rowCh both left open
// if stopCh get a message, just do nothing more and return
func (plan *Plan) selector(ctx context.Context, errCh chan error, stopCh chan struct{}, singleSel *SingleSelectPrimitive, rowBufSize int) (
	columnNames []string, columnTypes []*sql.ColumnType, rowCh <-chan []interface{}, err error) {
	if singleSel == nil {
		// if nil single select primitive, just return and do nothing
		return
	}
	rows, err := singleSel.QueryContext(ctx)
	if err != nil {
		err = errors.Wrap(err, "query context single select")
		return
	}
	rawRows := rows.(*ShardingRows).rawRows

	// Get column info from each shard table, tobe verified by caller
	columnNames, err = rawRows.Columns()
	if err != nil {
		err = errors.Wrap(err, "get column names failed")
		return
	}
	columnTypes, err = rawRows.ColumnTypes()
	if err != nil {
		err = errors.Wrap(err, "get column types failed")
		columnNames = nil
		return
	}

	out := make(chan []interface{}, rowBufSize)

	// start row selection from shard table
	go func() {
		var err error
		for {
			select {
			case <-ctx.Done():
				errCh <- errors.Wrapf(ctx.Err(),
					"scan context ends for query: %s, args %#v", singleSel.query, singleSel.namedArgs)
				return
			case <-stopCh:
				return
			default:
				if rawRows.Next() {
					values := make([]interface{}, len(columnNames))
					for i := range values {
						values[i] = new(interface{})
					}

					err := rawRows.Scan(values...)
					if err != nil {
						errCh <- errors.Wrapf(err,
							"scanning rows for query: %s, args %#v", singleSel.query, singleSel.namedArgs)
						return
					}

					for i := range columnNames {
						values[i] = *(values[i].(*interface{}))
					}
					out <- values
				} else {
					err = rawRows.Err()
					if err != nil {
						errCh <- errors.Wrapf(err,
							"during scan for query: %s, args %#v", singleSel.query, singleSel.namedArgs)
					} else {
						close(out)
					}
					return
				}
			}
		}
	}()

	rowCh = out
	return
}

func isColNamesEqual(c1, c2 []string) bool {
	if len(c1) != len(c2) {
		return false
	}
	for i, c := range c1 {
		if c != c2[i] {
			return false
		}
	}
	return true
}
