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
	"strconv"
	"strings"
	"time"

	"github.com/CovenantSQL/sqlparser"
	"github.com/pkg/errors"

	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

func getShardTS(sqlVal *sqlparser.SQLVal, args []driver.NamedValue) (insertTS int64, err error) {
	var (
		insertTime time.Time
		argIndex   int64
		arg        *driver.NamedValue
	)

	switch sqlVal.Type {
	case sqlparser.StrVal:
		insertTime, err = ParseTime(string(sqlVal.Val))
		if err != nil {
			return -1,
				errors.Wrap(err, "unsupported: time column")
		}
		insertTS = insertTime.Unix()
	case sqlparser.IntVal:
		insertTS, err = strconv.ParseInt(string(sqlVal.Val), 10, 64)
		if err != nil {
			return -1,
				errors.Wrap(err, "unsupported: time int val")
		}

	case sqlparser.ValArg:
		if len(sqlVal.Val) > 1 {
			for _, a := range args {
				// Name has higher priority
				if a.Name == string(sqlVal.Val[1:]) {
					arg = &driver.NamedValue{
						Name:    a.Name,
						Ordinal: a.Ordinal,
						Value:   a.Value,
					}
					break
				}
			}
			if arg == nil {
				argIndex, err = getValArgIndex(sqlVal)
				if err != nil {
					log.Warn(err)
				} else {
					for _, a := range args {
						if a.Ordinal == int(argIndex) {
							arg = &driver.NamedValue{
								Name:    a.Name,
								Ordinal: a.Ordinal,
								Value:   a.Value,
							}
							break
						}
					}
				}
			}
		}

		if arg == nil {
			return -1,
				errors.Errorf("unsupported: %s named args",
					string(sqlVal.Val))
		}

		switch v := arg.Value.(type) {
		case int64:
			insertTS = v
		case float64:
			insertTSf := v
			insertTS = int64(insertTSf)
		case string:
			insertTime, err = ParseTime(v)
			if err != nil {
				return -1,
					errors.Wrapf(err, "unsupported: sharding key in arg: %v", arg)
			}
			insertTS = insertTime.Unix()
		case time.Time:
			insertTS = v.Unix()
		case bool, []byte:
			return -1,
				errors.Errorf("unsupported: sharding key in arg: %v", arg)
		default:
			return -1,
				errors.Errorf("unsupported: sharding key type: %T", v)
		}

	case sqlparser.FloatVal, sqlparser.HexNum, sqlparser.HexVal, sqlparser.BitVal:
		return -1,
			errors.New("unsupported: sharding key")
	default:
		panic("unexpected SQL val type")
	}

	return
}

func getValArgIndex(sqlVal *sqlparser.SQLVal) (argIndex int64, err error) {
	if strings.HasPrefix(string(sqlVal.Val), ":v") {
		argIndex, err = strconv.ParseInt(string(sqlVal.Val[2:]), 10, 64)
		if err != nil {
			return -1,
				errors.Wrapf(err, "unsupported: %s named args in",
					string(sqlVal.Val))
		}
		return
	}
	return -1, errors.Errorf("no val index got in %s", string(sqlVal.Val))
}

func shardSuffix(shardID int64) string {
	return fmt.Sprintf("%s%010d", SHARD_SUFFIX, shardID)
}

func shardTableName(t *sqlparser.TableName, shardID int64) string {
	return fmt.Sprintf("%s%s", t.Name.String(), shardSuffix(shardID))
}

func toUserSpaceArgs(args []driver.NamedValue) (uArgs []interface{}) {
	uArgs = make([]interface{}, len(args))
	for i, a := range args {
		if a.Name == "" {
			uArgs[i] = a.Value
		} else {
			uArgs[i] = sql.Named(a.Name, a.Value)
		}
	}
	return
}

func toNamedArgs(args []driver.NamedValue) (rowArgs []interface{}) {
	rowArgs = make([]interface{}, len(args))
	for i, a := range args {
		var namedArg sql.NamedArg
		if a.Name == "" {
			namedArg = sql.Named(fmt.Sprintf("v%d", a.Ordinal), a.Value)
		} else {
			namedArg = sql.Named(a.Name, a.Value)
		}
		rowArgs[i] = namedArg
	}
	return
}

func toValuesNamedArgs(args []driver.NamedValue, columns []sqlparser.Expr) (rowArgs []interface{}) {
	rowArgs = make([]interface{}, 0, len(columns))
	for _, col := range columns {
		if colVal, ok := col.(*sqlparser.SQLVal); ok {
			if colVal.Type == sqlparser.ValArg {
				var colArgIndex int64
				if len(colVal.Val) > 1 {
					colName := string(colVal.Val[1:])
					colArgIndex, _ = getValArgIndex(colVal)
					for _, a := range args {
						if a.Name == colName || a.Ordinal == int(colArgIndex) {
							namedArg := sql.Named(colName, a.Value)
							rowArgs = append(rowArgs, namedArg)
						}
					}
				}
			}
		}
	}
	return
}

func generateShardSchema(originSchema string, shardID int64) (string, error) {
	return strings.Replace(originSchema, ShardSchemaToken, shardSuffix(shardID), -1), nil
}

func execInstructionsTx(ctx context.Context, ins []*SinglePrimitive) (result driver.Result, err error) {
	var (
		shardingResult = &ShardingResult{}
		tx             *sql.Tx
		rs             = make([]sql.Result, len(ins))
	)

	for i, singleRowPrimitive := range ins {
		var (
			r sql.Result
		)
		if i == 0 {
			tx, err = singleRowPrimitive.rawDB.BeginTx(ctx, nil)
			if err != nil {
				err = errors.Wrap(err, "begin tx failed")
				break
			}
		}
		r, err = singleRowPrimitive.ExecContext(ctx, tx)
		if err != nil {
			err = errors.Wrap(err, "execute tx failed")
			break
		}
		rs[i] = r
	}

	// if any error, rollback all
	if err != nil {
		if tx != nil {
			er := tx.Rollback()
			if er != nil {
				err = errors.Wrapf(err, "rollback tx failed: %v", er)
			}
		}
		return
	} else {
		if tx != nil {
			err = tx.Commit()
			if err != nil {
				err = errors.Wrap(err, "commit tx failed")
				return
			}
		}

		for _, r := range rs {
			if r == nil {
				break
			}
			var ra int64
			ra, err = r.RowsAffected()
			if err != nil {
				err = errors.Wrap(err, "get rows affected failed")
				return
			}
			shardingResult.RowsAffectedi += ra
			shardingResult.LastInsertIdi, err = r.LastInsertId()
			if err != nil {
				err = errors.Wrap(err, "get last insert id failed")
				return
			}
		}
		return shardingResult, nil
	}
}
