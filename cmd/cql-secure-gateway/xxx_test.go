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

package main

import (
	"crypto/rand"
	"database/sql"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	_ "github.com/CovenantSQL/go-sqlite3-encrypt"
	"github.com/CovenantSQL/sqlparser"
	"github.com/pkg/errors"
)

type wrapFakeDB struct {
	*sql.DB
}

func (d *wrapFakeDB) Query(query string, args ...interface{}) (rows *sql.Rows, err error) {
	var (
		stmt     sqlparser.Statement
		showStmt *sqlparser.Show
		ok       bool
	)
	if stmt, err = sqlparser.Parse(query); err != nil {
		return
	}

	if showStmt, ok = stmt.(*sqlparser.Show); !ok {
		err = errors.New("invalid query")
		return
	}

	switch showStmt.Type {
	case "tables":
		return d.DB.Query("SELECT name FROM sqlite_master WHERE type = \"table\"")
	case "table":
		return d.DB.Query("PRAGMA table_info(" + showStmt.OnTable.Name.String() + ")")
	default:
		err = errors.New("invalid query")
		return
	}
}

func fakeConn(testCreateTables []string) (db *wrapFakeDB, err error) {
	var realDB *sql.DB
	if realDB, err = sql.Open("sqlite3", ":memory:"); err != nil {
		return
	}

	for _, q := range testCreateTables {
		if _, err = realDB.Exec(q); err != nil {
			return
		}
	}

	db = &wrapFakeDB{
		DB: realDB,
	}

	return
}

func randomDBID() string {
	h := hash.Hash{}
	rand.Read(h[:])
	return h.String()
}
