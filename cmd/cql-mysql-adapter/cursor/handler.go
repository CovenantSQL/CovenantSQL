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

package cursor

import (
	"database/sql"

	"github.com/CovenantSQL/sqlparser"

	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
)

// Query defines resolved query utilized by cursor object.
type Query interface {
	IsRead() bool
	IsDDL() bool
	IsShow() bool
	IsExplain() bool
	GetDatabase() string
	GetQuery() string
	GetStmt() sqlparser.Statement
	GetParamCount() int
	GetResultColumnCount() int
}

// Rows define mocked rows for handler to process.
type Rows interface {
	Close() error
	Columns() ([]string, error)
	Err() error
	Next() bool
	Scan(...interface{}) error
}

// Handler defines callback function utilized by cursor object.
type Handler interface {
	EnsureDatabase(dbID string) error
	Resolve(user string, dbID string, query string) (Query, error)
	Query(q Query, args ...interface{}) (hash.Hash, Rows, error)
	Exec(q Query, args ...interface{}) (hash.Hash, sql.Result, error)
	QueryString(dbID string, query string, args ...interface{}) (hash.Hash, Rows, error)
	ExecString(dbID string, query string, args ...interface{}) (hash.Hash, sql.Result, error)
}
