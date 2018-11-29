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

package resolver

import "github.com/CovenantSQL/sqlparser"

// Query defines a resolver result query.
type Query struct {
	Database      string
	Stmt          sqlparser.Statement
	Query         string
	ParamCount    int
	ResultColumns []string
}

// GetDatabase returns current database id.
func (q *Query) GetDatabase() string {
	return q.Database
}

// GetParamCount returns current query parameter count.
func (q *Query) GetParamCount() int {
	return q.ParamCount
}

// GetResultColumnCount returns current query result column count.
func (q *Query) GetResultColumnCount() int {
	return len(q.ResultColumns)
}

// GetQuery returns parser refined query string.
func (q *Query) GetQuery() string {
	return q.Query
}

// GetStmt returns parser resolved statement ast.
func (q *Query) GetStmt() sqlparser.Statement {
	return q.Stmt
}

// IsDDL returns whether a resolved query is DDL or not.
func (q *Query) IsDDL() bool {
	if q.Stmt != nil {
		_, ok := q.Stmt.(*sqlparser.DDL)
		return ok
	}

	return false
}

// IsRead returns whether a resolved query is READ query.
func (q *Query) IsRead() bool {
	if q.Stmt != nil {
		switch q.Stmt.(type) {
		case *sqlparser.Show, *sqlparser.Explain, sqlparser.SelectStatement:
			return true
		}
	}

	return false
}

// IsShow returns whether a resolved query is a show query.
func (q *Query) IsShow() bool {
	if q.Stmt != nil {
		_, ok := q.Stmt.(*sqlparser.Show)
		return ok
	}

	return false
}

// IsExplain returns whether a resolved query is a explain query.
func (q *Query) IsExplain() bool {
	if q.Stmt != nil {
		_, ok := q.Stmt.(*sqlparser.Explain)
		return ok
	}

	return false
}
