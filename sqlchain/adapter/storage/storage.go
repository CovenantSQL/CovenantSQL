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
	"io"
)

// Storage defines the storage abstraction layer interface.
type Storage interface {
	// Create operation.
	Create(nodeCnt int) (dbID string, err error)
	// Drop operation.
	Drop(dbID string) (err error)
	// Query for result.
	Query(dbID string, query string, args ...interface{}) (columns []string, types []string, rows [][]interface{}, err error)
	// Exec for update.
	Exec(dbID string, query string, args ...interface{}) (affectedRows int64, lastInsertID int64, err error)
}

// golang does trick convert, use rowScanner to return the original result type in sqlite3 driver.
type rowScanner struct {
	fieldCnt int
	column   int           // current column
	fields   []interface{} // temp fields
	scanArgs []interface{}
}

func newRowScanner(fieldCnt int) (s *rowScanner) {
	s = &rowScanner{
		fieldCnt: fieldCnt,
		column:   0,
		fields:   make([]interface{}, fieldCnt),
		scanArgs: make([]interface{}, fieldCnt),
	}

	for i := 0; i != fieldCnt; i++ {
		s.scanArgs[i] = s
	}

	return
}

func (s *rowScanner) Scan(src interface{}) error {
	if s.fieldCnt <= s.column {
		// read complete
		return io.EOF
	}

	// convert to string if data type is []byte
	if srcInBytes, ok := src.([]byte); ok {
		s.fields[s.column] = string(srcInBytes)
	} else {
		s.fields[s.column] = src
	}

	s.column++

	return nil
}

func (s *rowScanner) GetRow() []interface{} {
	return s.fields
}

func (s *rowScanner) ScanArgs() []interface{} {
	// reset
	s.column = 0
	s.fields = make([]interface{}, s.fieldCnt)
	return s.scanArgs
}

func readAllRows(rows *sql.Rows) (result [][]interface{}, err error) {
	var columns []string
	if columns, err = rows.Columns(); err != nil {
		return
	}

	rs := newRowScanner(len(columns))
	result = make([][]interface{}, 0)

	for rows.Next() {
		err = rows.Scan(rs.ScanArgs()...)
		if err != nil {
			return
		}

		result = append(result, rs.GetRow())
	}

	err = rows.Err()

	return
}
