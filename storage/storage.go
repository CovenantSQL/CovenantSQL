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

// Package storage implements simple key-value storage interfaces based on sqlite3.
//
// Although a sql.DB should be safe for concurrent use according to
// https://golang.org/pkg/database/sql/#OpenDB, the go-sqlite3 implementation only guarantees
// the safety of concurrent readers. See https://github.com/mattn/go-sqlite3/issues/148 for details.
//
// As a result, here are some suggestions:
//
//	1. Perform as many concurrent GetValue(s) operations as you like;
//	2. Use only one goroutine to perform SetValue(s)/DelValue(s) operations;
//	3. Or implement a simple busy waiting yourself on a go-sqlite3.ErrLocked error if you must use
//	   concurrent writers.
package storage

import (
	"database/sql"
	"fmt"
	"sync"

	// Register go-sqlite3 engine.
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

// Storage represents a key-value storage.
type Storage struct {
	dsn   string
	table string
	db    *sql.DB
}

// KV represents a key-value pair.
type KV struct {
	Key   string
	Value []byte
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

	if _, err = db.Exec(stmt); err != nil {
		return st, err
	}

	st = &Storage{dsn, table, db}
	return st, err
}

// SetValue sets or replace the value to key.
func (s *Storage) SetValue(key string, value []byte) (err error) {
	stmt := fmt.Sprintf("INSERT OR REPLACE INTO `%s` (`key`, `value`) VALUES (?, ?)", s.table)
	_, err = s.db.Exec(stmt, key, value)

	return err
}

// SetValueIfNotExist sets the value to key if it doesn't exist.
func (s *Storage) SetValueIfNotExist(key string, value []byte) (err error) {
	stmt := fmt.Sprintf("INSERT OR IGNORE INTO `%s` (`key`, `value`) VALUES (?, ?)", s.table)
	_, err = s.db.Exec(stmt, key, value)

	return err
}

// DelValue deletes the value of key.
func (s *Storage) DelValue(key string) (err error) {
	stmt := fmt.Sprintf("DELETE FROM `%s` WHERE `key` = ?", s.table)
	_, err = s.db.Exec(stmt, key)

	return err
}

// GetValue fetches the value of key.
func (s *Storage) GetValue(key string) (value []byte, err error) {
	stmt := fmt.Sprintf("SELECT `value` FROM `%s` WHERE `key` = ?", s.table)

	if err = s.db.QueryRow(stmt, key).Scan(&value); err == sql.ErrNoRows {
		err = nil
	}

	return value, err
}

// SetValues sets or replaces the key-value pairs in kvs.
//
// Note that this is not a transaction. We use a prepared statement to send these queries. Each
// call may fail while part of the queries succeed.
func (s *Storage) SetValues(kvs []KV) (err error) {
	stmt := fmt.Sprintf("INSERT OR REPLACE INTO `%s` (`key`, `value`) VALUES (?, ?)", s.table)
	pStmt, err := s.db.Prepare(stmt)

	if err != nil {
		return err
	}

	defer pStmt.Close()

	for _, row := range kvs {
		if _, err = pStmt.Exec(row.Key, row.Value); err != nil {
			return err
		}
	}

	return nil
}

// SetValuesIfNotExist sets the key-value pairs in kvs if the key doesn't exist.
//
// Note that this is not a transaction. We use a prepared statement to send these queries. Each
// call may fail while part of the queries succeed.
func (s *Storage) SetValuesIfNotExist(kvs []KV) (err error) {
	stmt := fmt.Sprintf("INSERT OR IGNORE INTO `%s` (`key`, `value`) VALUES (?, ?)", s.table)
	pStmt, err := s.db.Prepare(stmt)

	if err != nil {
		return err
	}

	defer pStmt.Close()

	for _, row := range kvs {
		if _, err = pStmt.Exec(row.Key, row.Value); err != nil {
			return err
		}
	}

	return nil
}

// DelValues deletes the values of the keys.
//
// Note that this is not a transaction. We use a prepared statement to send these queries. Each
// call may fail while part of the queries succeed.
func (s *Storage) DelValues(keys []string) (err error) {
	stmt := fmt.Sprintf("DELETE FROM `%s` WHERE `key` = ?", s.table)
	pStmt, err := s.db.Prepare(stmt)

	if err != nil {
		return err
	}

	defer pStmt.Close()

	for _, key := range keys {
		if _, err = pStmt.Exec(key); err != nil {
			return err
		}
	}

	return nil
}

// GetValues fetches the values of keys.
//
// Note that this is not a transaction. We use a prepared statement to send these queries. Each
// call may fail while part of the queries succeed and some values may be altered during the
// queries. But the results will be returned only if all the queries succeed.
func (s *Storage) GetValues(keys []string) (kvs []KV, err error) {
	stmt := fmt.Sprintf("SELECT `value` FROM `%s` WHERE `key` = ?", s.table)
	pStmt, err := s.db.Prepare(stmt)

	if err != nil {
		return nil, err
	}

	defer pStmt.Close()

	kvs = make([]KV, len(keys))

	for index, key := range keys {
		kvs[index].Key = key

		if err = pStmt.QueryRow(key).Scan(&kvs[index].Value); err != nil && err != sql.ErrNoRows {
			return nil, err
		}
	}

	return kvs, nil
}

// SetValuesTx sets or replaces the key-value pairs in kvs as a transaction.
func (s *Storage) SetValuesTx(kvs []KV) (err error) {
	// Begin transaction
	tx, err := s.db.Begin()

	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	// Prepare statement
	stmt := fmt.Sprintf("INSERT OR REPLACE INTO `%s` (`key`, `value`) VALUES (?, ?)", s.table)
	pStmt, err := tx.Prepare(stmt)

	if err != nil {
		return err
	}

	defer pStmt.Close()

	// Execute queries
	for _, row := range kvs {
		if _, err = pStmt.Exec(row.Key, row.Value); err != nil {
			return err
		}
	}

	return nil
}

// SetValuesIfNotExistTx sets the key-value pairs in kvs if the key doesn't exist as a transaction.
func (s *Storage) SetValuesIfNotExistTx(kvs []KV) (err error) {
	// Begin transaction
	tx, err := s.db.Begin()

	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	// Prepare statement
	stmt := fmt.Sprintf("INSERT OR IGNORE INTO `%s` (`key`, `value`) VALUES (?, ?)", s.table)
	pStmt, err := tx.Prepare(stmt)

	if err != nil {
		return err
	}

	defer pStmt.Close()

	// Execute queries
	for _, row := range kvs {
		if _, err = pStmt.Exec(row.Key, row.Value); err != nil {
			return err
		}
	}

	return nil
}

// DelValuesTx deletes the values of the keys as a transaction.
func (s *Storage) DelValuesTx(keys []string) (err error) {
	// Begin transaction
	tx, err := s.db.Begin()

	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	// Prepare statement
	stmt := fmt.Sprintf("DELETE FROM `%s` WHERE `key` = ?", s.table)
	pStmt, err := tx.Prepare(stmt)

	if err != nil {
		return err
	}

	defer pStmt.Close()

	// Execute queries
	for _, key := range keys {
		if _, err = pStmt.Exec(key); err != nil {
			return err
		}
	}

	return nil
}

// GetValuesTx fetches the values of keys as a transaction.
func (s *Storage) GetValuesTx(keys []string) (kvs []KV, err error) {
	// Begin transaction
	tx, err := s.db.Begin()

	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	// Prepare statement
	stmt := fmt.Sprintf("SELECT `value` FROM `%s` WHERE `key` = ?", s.table)
	pStmt, err := tx.Prepare(stmt)

	if err != nil {
		return nil, err
	}

	defer pStmt.Close()

	// Execute queries
	kvs = make([]KV, len(keys))

	for index, key := range keys {
		kvs[index].Key = key
		err = pStmt.QueryRow(key).Scan(&kvs[index].Value)

		if err != nil && err != sql.ErrNoRows {
			return nil, err
		}
	}

	return kvs, nil
}
