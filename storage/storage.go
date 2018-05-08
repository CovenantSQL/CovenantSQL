/*
 * Copyright 2018 The ThunderDB Authors.
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

// Package storage implements simple key-value storage interfaces based on sqlite3
// Although DB should be safe for concurrent use according to
// https://golang.org/pkg/database/sql/#OpenDB, there are some issue with go-sqlite3 implementation.
// See https://github.com/mattn/go-sqlite3/issues/148 for details.
// As a result, concurrent use in this package is not recommended for now.
package storage

import (
	"database/sql"
	"fmt"
	"sync"

	// Register go-sqlite3 engine
	_ "github.com/mattn/go-sqlite3"
)

var (
	index = struct {
		mu *sync.Mutex
		db map[string]*sql.DB
	}{
		&sync.Mutex{},
		make(map[string]*sql.DB),
	}
)

func openDB(dsn string) (db *sql.DB, err error) {
	index.mu.Lock()
	defer index.mu.Unlock()

	db = index.db[dsn]
	if db == nil {
		db, err = sql.Open("sqlite3", dsn)

		if err != nil {
			return nil, err
		}

		index.db[dsn] = db
	}

	return db, err
}

// Storage represents a key-value storage
type Storage struct {
	dsn   string
	table string
	db    *sql.DB
}

// OpenStorage opens a database using the specified DSN and ensures that the specified table exists.
func OpenStorage(dsn string, table string) (st *Storage, err error) {
	// Open database
	var db *sql.DB
	db, err = openDB(dsn)

	if err != nil {
		return st, err
	}

	// Ensure table
	stmt := fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%s` (`key` TEXT PRIMARY KEY, `value` BLOB)",
		table)
	_, err = db.Exec(stmt)

	if err != nil {
		return st, err
	}

	st = &Storage{dsn, table, db}
	return st, err
}

// SetValue sets or replace the value to key
func (s *Storage) SetValue(key string, value []byte) (err error) {
	stmt := fmt.Sprintf("INSERT OR REPLACE INTO `%s` (`key`, `value`) VALUES (?, ?)", s.table)
	_, err = s.db.Exec(stmt, key, value)

	return err
}

// SetValueIfNotExist sets the value to key if it doesn't exist
func (s *Storage) SetValueIfNotExist(key string, value []byte) (err error) {
	stmt := fmt.Sprintf("INSERT OR IGNORE INTO `%s` (`key`, `value`) VALUES (?, ?)", s.table)
	_, err = s.db.Exec(stmt, key, value)

	return err
}

// DelValue deletes the value of key
func (s *Storage) DelValue(key string) (err error) {
	stmt := fmt.Sprintf("DELETE FROM `%s` WHERE key = ?", s.table)
	_, err = s.db.Exec(stmt, key)

	return err
}

// GetValue fetches the value of key
func (s *Storage) GetValue(key string) (value []byte, err error) {
	stmt := fmt.Sprintf("SELECT `value` FROM `%s` WHERE key = ?", s.table)
	err = s.db.QueryRow(stmt, key).Scan(&value)

	if err == sql.ErrNoRows {
		err = nil
	}

	return value, err
}
