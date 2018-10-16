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

package storage

import (
	"database/sql"

	"github.com/CovenantSQL/CovenantSQL/client"
)

// ThunderDBStorage defines the thunderdb database abstraction.
type ThunderDBStorage struct{}

// NewCovenantSQLStorage returns new thunderdb storage handler.
func NewCovenantSQLStorage() (s *ThunderDBStorage) {
	s = &ThunderDBStorage{}
	return
}

// Create implements the Storage abstraction interface.
func (s *ThunderDBStorage) Create(nodeCnt int) (dbID string, err error) {
	var dsn string
	if dsn, err = client.Create(client.ResourceMeta{Node: uint16(nodeCnt)}); err != nil {
		return
	}

	var cfg *client.Config
	if cfg, err = client.ParseDSN(dsn); err != nil {
		return
	}

	dbID = cfg.DatabaseID
	return
}

// Drop implements the Storage abstraction interface.
func (s *ThunderDBStorage) Drop(dbID string) (err error) {
	cfg := client.NewConfig()
	cfg.DatabaseID = dbID
	err = client.Drop(cfg.FormatDSN())
	return
}

// Query implements the Storage abstraction interface.
func (s *ThunderDBStorage) Query(dbID string, query string) (columns []string, types []string, result [][]interface{}, err error) {
	var conn *sql.DB
	if conn, err = s.getConn(dbID); err != nil {
		return
	}
	defer conn.Close()

	var rows *sql.Rows
	if rows, err = conn.Query(query); err != nil {
		return
	}
	defer rows.Close()

	if columns, err = rows.Columns(); err != nil {
		return
	}

	var colTypes []*sql.ColumnType

	if colTypes, err = rows.ColumnTypes(); err != nil {
		return
	}

	types = make([]string, len(colTypes))

	for i, c := range colTypes {
		if c != nil {
			types[i] = c.DatabaseTypeName()
		}
	}

	result, err = readAllRows(rows)
	return
}

// Exec implements the Storage abstraction interface.
func (s *ThunderDBStorage) Exec(dbID string, query string) (err error) {
	var conn *sql.DB
	if conn, err = s.getConn(dbID); err != nil {
		return
	}
	defer conn.Close()

	_, err = conn.Exec(query)

	return
}

func (s *ThunderDBStorage) getConn(dbID string) (db *sql.DB, err error) {
	cfg := client.NewConfig()
	cfg.DatabaseID = dbID

	return sql.Open("covenantsql", cfg.FormatDSN())
}
