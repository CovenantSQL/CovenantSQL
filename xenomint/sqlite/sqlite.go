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
	"time"

	"github.com/CovenantSQL/CovenantSQL/storage"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	sqlite3 "github.com/CovenantSQL/go-sqlite3-encrypt"
)

const (
	serializableDriver = "sqlite3-custom"
	dirtyReadDriver    = "sqlite3-dirty-reader"
)

func init() {
	sleepFunc := func(t int64) int64 {
		log.Info("sqlite func sleep start")
		time.Sleep(time.Duration(t))
		log.Info("sqlite func sleep end")
		return t
	}
	sql.Register(dirtyReadDriver, &sqlite3.SQLiteDriver{
		ConnectHook: func(c *sqlite3.SQLiteConn) (err error) {
			if _, err = c.Exec("PRAGMA read_uncommitted=1", nil); err != nil {
				return
			}
			if err = c.RegisterFunc("sleep", sleepFunc, true); err != nil {
				return
			}
			return
		},
	})
	sql.Register(serializableDriver, &sqlite3.SQLiteDriver{
		ConnectHook: func(c *sqlite3.SQLiteConn) (err error) {
			if err = c.RegisterFunc("sleep", sleepFunc, true); err != nil {
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
