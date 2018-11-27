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
	"database/sql/driver"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/CovenantSQL/sqlparser"
	"github.com/pkg/errors"
	"github.com/siddontang/go/log"
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
				fmt.Errorf("unsupported: %s named args",
					string(sqlVal.Val))
		}

		switch arg.Value.(type) {
		case int64:
			insertTS, _ = arg.Value.(int64)
		case float64:
			insertTSf, _ := arg.Value.(float64)
			insertTS = int64(insertTSf)
		case string:
			insertTime, err = ParseTime(string(sqlVal.Val))
			if err != nil {
				return -1,
					errors.Wrapf(err, "unsupported: sharding key in arg: %v", arg)
			}
			insertTS = insertTime.Unix()
		case time.Time:
			insertTS = arg.Value.(time.Time).Unix()
		case bool, []byte:
			return -1,
				fmt.Errorf("unsupported: sharding key in arg: %v", arg)
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
	return -1, fmt.Errorf("no val index got in %s", string(sqlVal.Val))
}

func shardSuffix(shardID int64) string {
	return fmt.Sprintf("%s%010d", SHARD_SUFFIX, shardID)
}

func shardTableName(t *sqlparser.TableName, shardID int64) string {
	return fmt.Sprintf("%s%s", t.Name.String(), shardSuffix(shardID))
}

func generateShardSchema(originSchema string, shardID int64) (string, error) {
	//TODO(auxten): check and add "IF NOT EXISTS"
	if !strings.Contains(originSchema, ShardSchemaToken) {
		return "", fmt.Errorf("not found '%s' in schema: %s", ShardSchemaToken, originSchema)
	}
	return strings.Replace(originSchema, ShardSchemaToken, shardSuffix(shardID), -1), nil
}
