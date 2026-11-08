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
	"errors"
	"fmt"
	"regexp"
	"strings"

	sqlite3 "github.com/CovenantSQL/go-sqlite3-encrypt"
	"github.com/CovenantSQL/sqlparser"

	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

var (
	// transactionStatement matches a pattern that is exactly one SQLite BEGIN, COMMIT, END or
	// ROLLBACK, but not ROLLBACK TO, with comments anywhere between its words and at most one
	// trailing semicolon. sqlparser has no grammar for them, and they hold nothing to check.
	transactionStatement = func() *regexp.Regexp {
		gap := `(?:\s|--[^\n]*|/\*(?:[^*]|\*+[^*/])*\*+/)` // whitespace or a comment
		name := `(?:[a-z_][a-z0-9_$]*|"(?:[^"]|"")*"|` + "`(?:[^`]|``)*`" + `|\[[^\]]*\]|'(?:[^']|'')*')`
		return regexp.MustCompile(strings.NewReplacer("~", gap, "NAME", name).Replace(
			`(?i)^~*(?:begin(?:~+(?:deferred|immediate|exclusive))?|commit|end|rollback)` +
				`(?:~+transaction(?:~+NAME)?)?~*(?:;~*)?$`))
	}()

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
		"unixepoch": {
			"now":       true,
			"localtime": true,
		},
		"timediff": {
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
	if transactionStatement.MatchString(pattern) {
		return false, pattern, nil, nil
	}
	var statements []sqlparser.Statement
	if _, statements, err = sqlparser.ParseMultiple(sqlparser.NewStringTokenizer(pattern)); err != nil {
		err = fmt.Errorf("parse sql failed: %w", err)
		return
	}
	// SQLite gets the pattern as it is, and the drivers refuse a pattern that SQLite reads as
	// more than one statement, so the statement checked here is the one that runs.
	switch {
	case len(statements) == 0:
		err = errors.New("query has no statement")
		return
	case len(statements) > 1:
		err = fmt.Errorf("query has %d statements: %w", len(statements), sqlite3.ErrMultipleStatements)
		return
	}
	p = pattern

	walkNodes := []sqlparser.SQLNode{statements[0]}

	switch stmt := statements[0].(type) {
	case *sqlparser.Show:
		var query string

		// The table name goes into the rewritten query as a string literal, which
		// PRAGMA table_info also accepts. SQLite stops reading a query at a NUL, so a
		// name with one would cut the query short.
		name := stmt.OnTable.Name.String()
		if strings.ContainsRune(name, 0) {
			err = fmt.Errorf("table name %q contains a NUL", name)
			return
		}
		name = "'" + strings.ReplaceAll(name, "'", "''") + "'"

		switch stmt.Type {
		case "table":
			if stmt.ShowCreate {
				query = fmt.Sprintf(`SELECT sql
FROM sqlite_master
WHERE type = "table" AND tbl_name = %s
AND tbl_name NOT LIKE "sqlite%%"`, name)
			} else {
				query = fmt.Sprintf(`PRAGMA table_info(%s)`, name)
			}
		case "index":
			query = fmt.Sprintf(`SELECT name
FROM sqlite_master
WHERE type = "index" AND tbl_name = %s
AND name NOT LIKE "sqlite%%"`, name)
		case "tables":
			query = `SELECT name FROM sqlite_master WHERE type = "table" AND name NOT LIKE "sqlite%"`
		}

		log.WithFields(log.Fields{
			"from": pattern,
			"to":   query,
		}).Debug("query translated")

		p = query
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
		// for new table
		if strings.HasPrefix(strings.ToLower(stmt.NewName.Name.String()), "sqlite") {
			// invalid table name
			err = fmt.Errorf("%s: %w", stmt.NewName.Name.String(), ErrInvalidTableName)
			return
		}
		// for alter table/alter index
		if strings.HasPrefix(strings.ToLower(stmt.Table.Name.String()), "sqlite") {
			// invalid table name
			err = fmt.Errorf("%s: %w", stmt.NewName.Name.String(), ErrInvalidTableName)
			return
		}
	}

	// scan query and test if there is any stateful query logic like time expression or random function
	err = sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		switch n := node.(type) {
		case *sqlparser.SQLVal:
			if n.Type == sqlparser.ValArg && bytes.EqualFold([]byte("CURRENT_TIMESTAMP"), n.Val) {
				// current_timestamp literal in default expression
				err = fmt.Errorf("DEFAULT CURRENT_TIMESTAMP not supported: %w", ErrStatefulQueryParts)
				return
			}
		case *sqlparser.TimeExpr:
			tb := sqlparser.NewTrackedBuffer(nil)
			err = fmt.Errorf("time expression %s not supported: %w",
				tb.WriteNode(n).String(), ErrStatefulQueryParts)
			return
		case *sqlparser.FuncExpr:
			if strings.HasPrefix(n.Name.Lowered(), "sqlite") {
				tb := sqlparser.NewTrackedBuffer(nil)
				err = fmt.Errorf("function call %s not supported: %w",
					tb.WriteNode(n).String(), ErrStatefulQueryParts)
				return
			}
			if sanitizeArgs, ok := sanitizeFunctionMap[n.Name.Lowered()]; ok {
				// need to sanitize this function
				tb := sqlparser.NewTrackedBuffer(nil)
				sanitizeErr := fmt.Errorf("stateful function call %s not supported: %w",
					tb.WriteNode(n).String(), ErrStatefulQueryParts)

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
				}, n.Exprs)

				return
			}
		}
		return true, nil
	}, walkNodes...)
	if err != nil {
		err = fmt.Errorf("parse sql failed: %w", err)
		return
	}

	ifs = make([]interface{}, len(args))
	for i, v := range args {
		ifs[i] = sql.NamedArg{
			Name:  v.Name,
			Value: v.Value,
		}
	}
	return
}
