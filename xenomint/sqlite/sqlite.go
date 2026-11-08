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

package sqlite

import (
	"database/sql"
	"strings"
	"time"

	sqlite3 "github.com/CovenantSQL/go-sqlite3-encrypt"

	"github.com/CovenantSQL/CovenantSQL/crypto/symmetric"
	"github.com/CovenantSQL/CovenantSQL/storage"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

const (
	serializableDriver = "sqlite3-custom"
	dirtyReadDriver    = "sqlite3-dirty-reader"
)

// deniedFuncs are the functions that would make the miners of a database diverge: random
// values, loaded code, the SQLite build and file layout, and the changes a connection has
// counted since it opened. The query sanitizer rejects most of them too, but only where
// sqlparser reaches; SQLite asks the authorizer wherever it compiles a function call,
// including column defaults, CHECK constraints, triggers and views. Other sqlite_* functions
// stay allowed, because ALTER TABLE runs SQLite's own sqlite_rename_*.
var deniedFuncs = map[string]bool{
	"random": true, "randomblob": true, "load_extension": true,
	"sqlite_version": true, "sqlite_source_id": true, "sqlite_compileoption_used": true,
	"sqlite_compileoption_get": true, "sqlite_offset": true, "total_changes": true,
}

// allowedPragmas are the only PRAGMAs these connections run after the authorizer is installed:
// the dirty reader's read_uncommitted and the table_info that SHOW is rewritten to. The DSN
// PRAGMAs run before ConnectHook. Any other PRAGMA may report the build or the host, and some,
// like rekey and case_sensitive_like, act as soon as SQLite compiles them, even under EXPLAIN.
var allowedPragmas = map[string]bool{"read_uncommitted": true, "table_info": true}

func authorize(op int, arg1, arg2, _ string) int {
	switch {
	case op == sqlite3.SQLITE_FUNCTION && deniedFuncs[strings.ToLower(arg2)],
		op == sqlite3.SQLITE_PRAGMA && !allowedPragmas[strings.ToLower(arg1)]:
		return sqlite3.SQLITE_DENY
	}
	return sqlite3.SQLITE_OK
}

func init() {
	decryptFunc := func(in, pass, salt []byte) (out []byte, err error) {
		out, err = symmetric.DecryptWithPassword(in, pass, salt)
		return
	}

	sleepFunc := func(t int64) int64 {
		log.Info("sqlite func sleep start")
		time.Sleep(time.Duration(t))
		log.Info("sqlite func sleep end")
		return t
	}

	regCustomFunc := func(c *sqlite3.SQLiteConn) (err error) {
		c.RegisterAuthorizer(authorize)
		if err = c.RegisterFunc("sleep", sleepFunc, true); err != nil {
			return
		}
		if err = c.RegisterFunc("decrypt", decryptFunc, true); err != nil {
			return
		}
		return
	}

	sql.Register(dirtyReadDriver, &sqlite3.SQLiteDriver{
		ConnectHook: func(c *sqlite3.SQLiteConn) (err error) {
			if err = regCustomFunc(c); err != nil {
				return
			}
			if _, err = c.Exec("PRAGMA read_uncommitted=1", nil); err != nil {
				return
			}
			return
		},
	})
	sql.Register(serializableDriver, &sqlite3.SQLiteDriver{
		ConnectHook: func(c *sqlite3.SQLiteConn) (err error) {
			if err = regCustomFunc(c); err != nil {
				return
			}
			return
		},
	})
}

// SQLite3 is the sqlite3 implementation of the xenomint/interfaces.Storage interface.
type SQLite3 struct {
	filename    string
	dirtyReader *sql.DB
	reader      *sql.DB
	writer      *sql.DB
}

// NewSqlite returns a new SQLite3 instance attached to filename.
func NewSqlite(filename string) (s *SQLite3, err error) {
	var (
		instance  = &SQLite3{filename: filename}
		shmRODSN  string
		privRODSN string
		shmRWDSN  string
		dsn       *storage.DSN
	)

	if dsn, err = storage.NewDSN(filename); err != nil {
		return
	}
	// Every replica of a database executes the same queries, so none of them may
	// see the current time: 'now' in any form gives NULL.
	dsn.AddParam("vfs", "nocurrenttime")

	dsnRO := dsn.Clone()
	dsnRO.AddParam("_journal_mode", "WAL")
	dsnRO.AddParam("_query_only", "on")
	dsnRO.AddParam("cache", "shared")
	shmRODSN = dsnRO.Format()

	dsnPrivRO := dsn.Clone()
	dsnPrivRO.AddParam("_journal_mode", "WAL")
	dsnPrivRO.AddParam("_query_only", "on")
	privRODSN = dsnPrivRO.Format()

	dsnSHMRW := dsn.Clone()
	dsnSHMRW.AddParam("_journal_mode", "WAL")
	dsnSHMRW.AddParam("cache", "shared")
	shmRWDSN = dsnSHMRW.Format()

	if instance.dirtyReader, err = sql.Open(dirtyReadDriver, shmRODSN); err != nil {
		return
	}
	if instance.reader, err = sql.Open(serializableDriver, privRODSN); err != nil {
		return
	}
	if instance.writer, err = sql.Open(serializableDriver, shmRWDSN); err != nil {
		return
	}
	s = instance
	return
}

// DirtyReader implements DirtyReader method of the xenomint/interfaces.Storage interface.
func (s *SQLite3) DirtyReader() *sql.DB {
	return s.dirtyReader
}

// Reader implements Reader method of the xenomint/interfaces.Storage interface.
func (s *SQLite3) Reader() *sql.DB {
	return s.reader
}

// Writer implements Writer method of the xenomint/interfaces.Storage interface.
func (s *SQLite3) Writer() *sql.DB {
	return s.writer
}

// Close implements Close method of the xenomint/interfaces.Storage interface.
func (s *SQLite3) Close() (err error) {
	if err = s.dirtyReader.Close(); err != nil {
		return
	}
	if err = s.reader.Close(); err != nil {
		return
	}
	if err = s.writer.Close(); err != nil {
		return
	}
	return
}
