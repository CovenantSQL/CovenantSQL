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
	"sync"

	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/CovenantSQL/sqlparser"
	"github.com/pkg/errors"
)

const SHARD_SUFFIX = "_ts_"

// buildInsertPlan builds the route for an INSERT statement.
func buildInsertPlan(query string,
	ins *sqlparser.Insert,
	args []driver.NamedValue,
	c *ShardingConn,
) (instructions Primitive, err error) {
	log.Debugf("buildInsertPlan got %s\n insert:%#v\n args: %#v", query, ins, args)
	//q.Q(ins, args)

	if conf, ok := c.conf[ins.Table.Name.CompliantName()]; ok {
		if !conf.ShardColName.IsEmpty() && conf.ShardInterval > 0 {
			if ins.Action == sqlparser.ReplaceStr {
				return nil, errors.New("unsupported: REPLACE INTO with sharded schema")
			}
			if ins.OnDup != nil {
				return nil, errors.New("unsupported: OnDup with sharded schema")
			}
			shardColIndex := ins.Columns.FindColumn(conf.ShardColName)
			if shardColIndex < 0 {
				return nil, fmt.Errorf("sharding column not found in %s", query)
			} else {
				if rows, ok := ins.Rows.(sqlparser.Values); ok {
					allInserts := &Insert{
						Mutex:        sync.Mutex{},
						Instructions: make([]*SingleRowPrimitive, 0, len(rows)),
					}

					for _, row := range rows {
						if shardColIndex <= len(row)-1 {
							var (
								insertTS  int64
								singleIns *SingleRowPrimitive
								rowArgs   []interface{}
							)
							val := row[shardColIndex]
							if sqlVal, ok := val.(*sqlparser.SQLVal); ok {
								insertTS, err = getShardTS(sqlVal, args)
								if err != nil {
									return nil,
										errors.Wrapf(err, "get shard timestamp failed for: %s", query)
								}

								for _, col := range row {
									if colVal, ok := col.(*sqlparser.SQLVal); ok {
										if colVal.Type == sqlparser.ValArg {
											var colArgIndex int64
											colArgIndex, err = getValArgIndex(colVal)
											if err == nil {
												for _, a := range args {
													if a.Name == string(colVal.Val) || a.Ordinal == int(colArgIndex) {
														namedArg := sql.Named(string(colVal.Val[1:]), a.Value)
														rowArgs = append(rowArgs, namedArg)
													}
												}
											}
										}
									}
								}
								singleIns, err = c.prepareShardInstruction(insertTS, ins, row, rowArgs, conf)
								//singleIns, err = c.prepareShardInstruction(insertTS, ins, row, args, conf)
								if err != nil {
									return nil, errors.Wrap(err, "prepare shard instruction failed")
								}
								allInserts.Instructions = append(allInserts.Instructions, singleIns)
								return allInserts, nil
							} else {
								//TODO(auxten) val can be sqlparser.FuncExpr, process SQL like this:
								// insert into foo(id, name, time) values(61, 'foo', strftime('%s','now'));
								return nil,
									fmt.Errorf("non SQLVal type in column is not supported: %s", query)
							}

						} else {
							return nil,
								fmt.Errorf("bug: shardColIndex outof range %s", query)
						}
					}
				} else {
					return nil,
						fmt.Errorf("non Values type in Rows is not supported: %s", query)
				}
			}
		} else if strings.HasPrefix(
			ins.Table.Name.String(), conf.ShardTableName.Name.String()+SHARD_SUFFIX) {
			return nil, errors.New("unsupported: query on sharding table directly")
		} else {
			return nil, fmt.Errorf("sharding conf set but not configured: %#v", conf)
		}
	}

	return &BasePrimitive{
		query:   query,
		args:    args,
		rawConn: c.rawConn,
	}, nil
}

type Insert struct {
	sync.Mutex
	Instructions []*SingleRowPrimitive
}

// SingleRowPrimitive is the primitive just insert one row to one shard
type SingleRowPrimitive struct {
	// query is the original query.
	query string
	// namedArgs is the named args
	namedArgs []interface{}
	// rawDB is the userspace sql conn
	rawDB *sql.DB
}

func (s *SingleRowPrimitive) ExecContext(ctx context.Context) (result driver.Result, err error) {
	panic("only run with transaction")
}

func (ins *Insert) ExecContext(ctx context.Context) (result driver.Result, err error) {
	ins.Lock()
	defer ins.Unlock()
	var (
		shardingResult = &ShardingResult{}
		txs            = make([]*sql.Tx, len(ins.Instructions))
		rs             = make([]sql.Result, len(ins.Instructions))
	)

	for i, ins := range ins.Instructions {
		var (
			tx *sql.Tx
			r  sql.Result
		)
		tx, err = ins.rawDB.BeginTx(ctx, nil)
		if err != nil {
			log.Errorf("begin tx failed: %v", err)
			break
		}
		txs[i] = tx
		r, err = tx.ExecContext(ctx, ins.query, ins.namedArgs...)
		if err != nil {
			log.Errorf("execute tx failed: %v", err)
			break
		}
		rs[i] = r
	}

	// if any error, rollback all
	if err != nil {
		for _, tx := range txs {
			if tx == nil {
				break
			}
			err = tx.Rollback()
			if err != nil {
				log.Errorf("rollback tx failed: %v", err)
			}
		}
		return nil, err
	} else {
		for _, tx := range txs {
			if tx == nil {
				break
			}
			err = tx.Commit()
			if err != nil {
				log.Errorf("commit tx failed: %v", err)
			}
		}

		for _, r := range rs {
			if r == nil {
				break
			}
			var ra int64
			ra, err = r.RowsAffected()
			if err != nil {
				log.Errorf("get rows affected failed: %v", err)
				return nil, err
			}
			shardingResult.RowsAffectedi += ra
			shardingResult.LastInsertIdi, err = r.LastInsertId()
			if err != nil {
				log.Errorf("get last insert id failed: %v", err)
				return nil, err
			}
		}
		return shardingResult, nil
	}
}

func (sc *ShardingConn) prepareShardTable(t *sqlparser.TableName, shardID int64) (err error) {
	sTable := shardTableName(t, shardID)
	if _, ok := sc.shardingTables.Load(sTable); !ok {
		sc.Lock()
		conf, gotConf := sc.conf[t.Name.CompliantName()]
		sc.Unlock()
		if gotConf {
			var shardSchema string
			shardSchema, err = generateShardSchema(conf.ShardSchema, shardID)
			if err == nil {
				_, err = sc.rawDB.Exec(shardSchema)
				if err == nil {
					sc.shardingTables.Store(sTable, true)
					return
				} else {
					return fmt.Errorf("creating shard table %s with %s failed: %v",
						sTable, shardSchema, err)
				}
			} else {
				return
			}

		} else {
			return fmt.Errorf("original schema for %s not found", t.Name.CompliantName())
		}
	}
	return
}

func (sc *ShardingConn) prepareShardInstruction(insertTS int64,
	ins *sqlparser.Insert,
	row sqlparser.ValTuple,
	rowArgs []interface{},
	conf *ShardingConf) (p *SingleRowPrimitive, err error) {

	timestampDiff := insertTS - conf.ShardStarttime.Unix()
	if timestampDiff < 0 {
		return nil, fmt.Errorf("insert time %d before shard start time", insertTS)
	}
	shardID := timestampDiff / conf.ShardInterval
	log.Debugf("shardID: %d, timestamp diff: %d", shardID, timestampDiff)

	err = sc.prepareShardTable(&ins.Table, shardID)
	if err != nil {
		return nil,
			errors.Wrapf(err, "preparing shard table for %s %d failed",
				ins.Table.Name.CompliantName(), shardID)
	}

	newIns := &sqlparser.Insert{
		Action:   ins.Action,
		Comments: nil,
		Ignore:   ins.Ignore,
		Table: sqlparser.TableName{
			Name:      sqlparser.NewTableIdent(shardTableName(&ins.Table, shardID)),
			Qualifier: sqlparser.NewTableIdent(ins.Table.Qualifier.String()),
		},
		Partitions: ins.Partitions,
		Columns:    ins.Columns,
		Rows:       sqlparser.Values{row},
		OnDup:      nil,
	}
	buf := sqlparser.NewTrackedBuffer(nil)
	newIns.Format(buf)
	p = &SingleRowPrimitive{
		query:     buf.String(),
		namedArgs: rowArgs,
		rawDB:     sc.rawDB,
	}
	log.Debugf("New SQL: %s %v", p.query, p.namedArgs)
	return
}
