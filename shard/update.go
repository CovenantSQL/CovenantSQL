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

	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/CovenantSQL/sqlparser"
	"github.com/pkg/errors"
)

// buildUpdatePlan builds the route for an UPDATE statement.
func buildUpdatePlan(query string,
	update *sqlparser.Update,
	args []driver.NamedValue,
	c *ShardingConn,
) (instructions Primitive, err error) {
	log.Debugf("buildUpdatePlan got %s\n update:%#v\n args: %#v", query, update, args)
	if update.OrderBy != nil || update.Limit != nil {
		return nil, errors.New("unsupported: OrderBy or LIMIT in UPDATE")
	}

	if len(update.TableExprs) == 1 {
		var (
			tableExpr       *sqlparser.AliasedTableExpr
			simpleTableExpr sqlparser.SimpleTableExpr
			originTableName sqlparser.TableIdent
			ok              bool
		)
		if tableExpr, ok = update.TableExprs[0].(*sqlparser.AliasedTableExpr); !ok {
			return nil, errors.New("unsupported: FROM table type")
		}
		if simpleTableExpr, ok = tableExpr.Expr.(sqlparser.SimpleTableExpr); !ok {
			return nil, errors.New("unsupported: FROM table type")
		}
		originTableName = sqlparser.GetTableName(simpleTableExpr)

		if conf, ok := c.conf[originTableName.CompliantName()]; ok {
			if !conf.ShardColName.IsEmpty() && conf.ShardInterval > 0 {

				for _, expr := range update.Exprs {
					if expr.Name.Name.Equal(conf.ShardColName) {
						return nil,
							errors.New("unsupported: UPDATE shard column")
					}
				}

				// split update to shard tables
				var shards []string
				shards, err = c.getTableShards(originTableName.CompliantName())
				updateInstructions := &Update{
					Mutex:        sync.Mutex{},
					Instructions: make([]*SinglePrimitive, len(shards)),
				}

				originFrom := update.TableExprs[0]
				//TODO(auxten) deep copy a new update is better
				for i, shard := range shards {
					update.TableExprs[0] = &sqlparser.AliasedTableExpr{
						Expr: sqlparser.TableName{
							Name:      sqlparser.NewTableIdent(shard),
							Qualifier: sqlparser.TableIdent{},
						},
					}

					buf := sqlparser.NewTrackedBuffer(nil)
					update.Format(buf)
					//FIXME(auxten) just use the same update in shard table for now
					fixedArgs := toNamedArgs(args)
					updateInstructions.Instructions[i] = &SinglePrimitive{
						query:     buf.String(),
						namedArgs: fixedArgs,
						rawDB:     c.rawDB,
					}
				}
				update.TableExprs[0] = originFrom
				return updateInstructions, nil

			} else {
				return nil, errors.Errorf("sharding conf set but not configured: %#v", conf)
			}
		} else {
			// not sharding table
		}
	} else {
		return nil, errors.Errorf("update target must be 1: %s", query)
	}

	return &BasePrimitive{
		query:   query,
		args:    args,
		rawConn: c.rawConn,
		rawDB:   c.rawDB,
	}, nil
}

type Update struct {
	sync.Mutex
	Instructions []*SinglePrimitive
}

func (update *Update) ExecContext(ctx context.Context, tx *sql.Tx) (driver.Result, error) {
	update.Lock()
	defer update.Unlock()
	return execInstructionsTx(ctx, update.Instructions)
}

func (update *Update) QueryContext(ctx context.Context) (driver.Rows, error) {
	panic("should not call query in update")
}
