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

	sqlite3 "github.com/CovenantSQL/go-sqlite3-encrypt"
	"github.com/CovenantSQL/sqlparser"
)

// Primitive is the interface that needs to be satisfied by
// all primitives of a plan.
type Primitive interface {
	ExecContext(ctx context.Context, tx *sql.Tx) (driver.Result, error)
	QueryContext(ctx context.Context) (driver.Rows, error)
}

// BasePrimitive is the primitive just execute the origin query with origin args on rawConn
type BasePrimitive struct {
	// query is the original query.
	query string
	// args is the original args.
	args []driver.NamedValue
	// rawConn is the raw sqlite3 conn
	rawConn *sqlite3.SQLiteConn
	// rawDB is the userspace sql conn
	rawDB *sql.DB
}

func (dp *BasePrimitive) QueryContext(ctx context.Context) (driver.Rows, error) {
	return dp.rawConn.QueryContext(ctx, dp.query, dp.args)
}

func (dp *BasePrimitive) ExecContext(ctx context.Context, tx *sql.Tx) (result driver.Result, err error) {
	return dp.rawConn.ExecContext(ctx, dp.query, dp.args)
}

// BuildFromStmt builds a plan based on the AST provided.
func BuildFromStmt(query string, args []driver.NamedValue, stmt sqlparser.Statement, c *ShardingConn) (plan *Plan, err error) {
	plan = &Plan{
		OriginQuery: query,
		OriginArgs:  args,
		c:           c,
	}
	switch stmt := stmt.(type) {
	case *sqlparser.Select:
		plan.Instructions, err = buildSelectPlan(query, stmt, args, c)
	case *sqlparser.Insert:
		plan.Instructions, err = buildInsertPlan(query, stmt, args, c)
	case *sqlparser.Delete:
		plan.Instructions, err = buildDeletePlan(query, stmt, args, c)
	case *sqlparser.Update:
		plan.Instructions, err = buildUpdatePlan(query, stmt, args, c)
	case *sqlparser.DDL,
		*sqlparser.Show,
		*sqlparser.Set,
		*sqlparser.Union,
		*sqlparser.DBDDL:
		// FIXME(auxten) if contains any statement have not support, we just
		// execute it for test
		plan.Instructions = &BasePrimitive{
			query:   query,
			args:    args,
			rawConn: c.rawConn,
			rawDB:   c.rawDB,
		}

	default:
		panic(fmt.Sprintf("BUG: unexpected statement type: %T", stmt))
	}
	if err != nil {
		return nil, err
	}
	return plan, nil
}
