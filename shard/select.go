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
	"sync"

	"github.com/CovenantSQL/sqlparser"
	"github.com/pkg/errors"
)

// buildSelectPlan builds the route for an SELECT statement.
func buildSelectPlan(query string,
	sel *sqlparser.Select,
	args []driver.NamedValue,
	c *ShardingConn,
) (instructions Primitive, err error) {
	if sel.GroupBy != nil || sel.Having != nil {
		return nil, errors.New("unsupported: GROUP or HAVING in SELECT")
	}
	if sel.Distinct != "" {
		return nil, errors.New("unsupported: DISTINCT in SELECT")
	}

	if len(sel.From) == 1 {
		if tableExpr, ok := sel.From[0].(*sqlparser.AliasedTableExpr); ok {
			if tableName, ok := tableExpr.Expr.(sqlparser.TableName); ok {
				if conf, ok := c.conf[tableName.Name.CompliantName()]; ok {
					if !conf.ShardColName.IsEmpty() && conf.ShardInterval > 0 {
						// split select to shard tables
						var shards []string
						shards, err = c.getTableShards(tableName.Name.CompliantName())
						selectInstructions := &Select{
							Mutex:        sync.Mutex{},
							shardTable:   tableName.Name.CompliantName(),
							Instructions: make([]*SingleSelectPrimitive, len(shards)),
						}

						//TODO(auxten) deep copy a new select is better
						originFrom := sel.From[0]
						for i, shard := range shards {
							sel.From[0] = &sqlparser.AliasedTableExpr{
								Expr: sqlparser.TableName{
									Name:      sqlparser.NewTableIdent(shard),
									Qualifier: sqlparser.TableIdent{},
								},
								As: sel.From[0].(*sqlparser.AliasedTableExpr).As,
							}
							buf := sqlparser.NewTrackedBuffer(nil)
							sel.Format(buf)
							//FIXME(auxten) just use the same select in shard table for now
							fixedArgs := toNamedArgs(args)
							selectInstructions.Instructions[i] = &SingleSelectPrimitive{
								query:     buf.String(),
								namedArgs: fixedArgs,
								rawDB:     c.rawDB,
							}
						}
						sel.From[0] = originFrom
						return selectInstructions, nil
					} else {
						return nil,
							errors.New("shard column or interval not configured")
					}
				} else {
					/*
						not in shard conf, just build BasePrimitive with raw query
						at the ending return
					*/
				}
			} else {
				return nil, errors.Errorf("unsupported: tableName type: %T", tableExpr.Expr)
			}
		} else {
			return nil, errors.New("unsupported: FROM table type")
		}
	} else {
		return nil, errors.New("unsupported: SELECT from multi table")
	}

	return &BasePrimitive{
		query:   query,
		args:    args,
		rawConn: c.rawConn,
		rawDB:   c.rawDB,
	}, nil
}

// SingleSelectPrimitive is the primitive just insert one row to one shard
type SingleSelectPrimitive struct {
	// query is the original query.
	query string
	// namedArgs is the named args
	namedArgs []interface{}
	// rawDB is the user space sql conn
	rawDB *sql.DB
}

func (s *SingleSelectPrimitive) QueryContext(ctx context.Context) (driver.Rows, error) {
	rows, err := s.rawDB.QueryContext(ctx, s.query, s.namedArgs...)
	if err != nil {
		return nil, err
	}
	return &ShardingRows{
		rawRows: rows,
	}, nil
}

func (s *SingleSelectPrimitive) ExecContext(ctx context.Context, tx *sql.Tx) (result driver.Result, err error) {
	panic("should not call exec in select")
}

type Select struct {
	sync.Mutex
	shardTable   string
	Instructions []*SingleSelectPrimitive
}

func (sel *Select) ExecContext(ctx context.Context, _ *sql.Tx) (driver.Result, error) {
	panic("should not call exec in select")
}

func (sel *Select) QueryContext(ctx context.Context) (driver.Rows, error) {
	panic("not implemented")
}
