/*
 * Copyright 2018 The CovenantSQL Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package xenomint

import (
	"bytes"
	"database/sql"
	"strings"

	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/CovenantSQL/sqlparser"
	"github.com/pkg/errors"
)

var (
	sanitizeFunctionMap = map[string]map[string]bool{
		"load_extension": nil,
		"unlikely":       nil,
		"likelihood":     nil,
		"likely":         nil,
		"affinity":       nil,
		"typeof":         nil,
		"random":         nil,
		"randomblob":     nil,
		"unknown":        nil,
		"date": {
			"now":       true,
			"localtime": true,
		},
		"time": {
			"now":       true,
			"localtime": true,
		},
		"datetime": {
			"now":       true,
			"localtime": true,
		},
		"julianday": {
			"now":       true,
			"localtime": true,
		},
		"strftime": {
			"now":       true,
			"localtime": true,
		},

		// all sqlite functions is already ignored, including
		//"sqlite_offset":             nil,
		//"sqlite_version":            nil,
		//"sqlite_source_id":          nil,
		//"sqlite_log":                nil,
		//"sqlite_compileoption_used": nil,
		//"sqlite_rename_table":       nil,
		//"sqlite_rename_trigger":     nil,
		//"sqlite_rename_parent":      nil,
		//"sqlite_record":             nil,
	}
)

func convertQueryAndBuildArgs(pattern string, args []types.NamedArg) (containsDDL bool, p string, ifs []interface{}, err error) {
	var (
		tokenizer  = sqlparser.NewStringTokenizer(pattern)
		queryParts []string
		statements []sqlparser.Statement
		i          int
		origQuery  string
		query      string
	)

	if queryParts, statements, err = sqlparser.ParseMultiple(tokenizer); err != nil {
		err = errors.Wrap(err, "parse sql failed")
		return
	}

	for i = range queryParts {
		walkNodes := []sqlparser.SQLNode{statements[i]}

		switch stmt := statements[i].(type) {
		case *sqlparser.Show:
			origQuery = queryParts[i]

			switch stmt.Type {
			case "table":
				if stmt.ShowCreate {
					query = "SELECT sql FROM sqlite_master WHERE type = \"table\" AND tbl_name = \"" +
						stmt.OnTable.Name.String() + "\""
				} else {
					query = "PRAGMA table_info(" + stmt.OnTable.Name.String() + ")"
				}
			case "index":
				query = "SELECT name FROM sqlite_master WHERE type = \"index\" AND tbl_name = \"" +
					stmt.OnTable.Name.String() + "\""
			case "tables":
				query = "SELECT name FROM sqlite_master WHERE type = \"table\""
			}

			log.WithFields(log.Fields{
				"from": origQuery,
				"to":   query,
			}).Debug("query translated")

			queryParts[i] = query
		case *sqlparser.DDL:
			containsDDL = true
			if stmt.TableSpec != nil {
				// walk table default values for invalid stateful expressions
				for _, c := range stmt.TableSpec.Columns {
					if c == nil || c.Type.Default == nil {
						continue
					}

					walkNodes = append(walkNodes, c.Type.Default)
				}
			}
		}

		// scan query and test if there is any stateful query logic like time expression or random function
		err = sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
			switch n := node.(type) {
			case *sqlparser.SQLVal:
				if n.Type == sqlparser.ValArg && bytes.EqualFold([]byte("CURRENT_TIMESTAMP"), n.Val) {
					// current_timestamp literal in default expression
					err = errors.Wrap(ErrStatefulQueryParts, "DEFAULT CURRENT_TIMESTAMP not supported")
					return
				}
			case *sqlparser.TimeExpr:
				tb := sqlparser.NewTrackedBuffer(nil)
				err = errors.Wrapf(ErrStatefulQueryParts, "time expression %s not supported",
					tb.WriteNode(n).String())
				return
			case *sqlparser.FuncExpr:
				if strings.HasPrefix(n.Name.Lowered(), "sqlite") {
					tb := sqlparser.NewTrackedBuffer(nil)
					err = errors.Wrapf(ErrStatefulQueryParts, "function call %s not supported",
						tb.WriteNode(n).String())
					return
				}
				if sanitizeArgs, ok := sanitizeFunctionMap[n.Name.Lowered()]; ok {
					// need to sanitize this function
					tb := sqlparser.NewTrackedBuffer(nil)
					sanitizeErr := errors.Wrapf(ErrStatefulQueryParts, "stateful function call %s not supported",
						tb.WriteNode(n).String())

					if sanitizeArgs == nil {
						err = sanitizeErr
						return
					}

					err = sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, walkErr error) {
						if v, ok := node.(*sqlparser.SQLVal); ok {
							if v.Type == sqlparser.StrVal {
								argStr := strings.ToLower(string(v.Val))

								if sanitizeArgs[argStr] {
									walkErr = sanitizeErr
								}
								return
							}
						}
						return true, nil
					})

					return
				}
			}
			return true, nil
		}, walkNodes...)
		if err != nil {
			err = errors.Wrapf(err, "parse sql failed")
			return
		}
	}

	p = strings.Join(queryParts, "; ")

	ifs = make([]interface{}, len(args))
	for i, v := range args {
		ifs[i] = sql.NamedArg{
			Name:  v.Name,
			Value: v.Value,
		}
	}
	return
}
