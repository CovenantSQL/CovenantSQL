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
	"database/sql"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/CovenantSQL/CovenantSQL/client"
	my "github.com/siddontang/go-mysql/mysql"
)

var (
	dbIDRegex        = regexp.MustCompile("^[a-zA-Z0-9_\\.]+$")
	emptyResultQuery = regexp.MustCompile("^(?i)\\s*(?:(?:SELECT\\s+)?@@(?:\\w+\\.)?|SHOW\\s+VARIABLES|SHOW\\s+DATABASES|SET|ROLLBACK).*$")
	useDatabaseQuery = regexp.MustCompile("^(?i)\\s*USE\\s+(\\w+)\\s*$")
	readQuery        = regexp.MustCompile("^(?i)\\s*(?:SELECT|SHOW|DESC)")
)

// Cursor is a mysql connection handler, like a cursor of normal database.
type Cursor struct {
	curDBLock     sync.Mutex
	curDB         string
	curDBInstance *sql.DB
}

// NewCursor returns a new cursor.
func NewCursor() (c *Cursor) {
	return &Cursor{}
}

func (c *Cursor) buildResultSet(rows *sql.Rows) (r *my.Result, err error) {
	// get columns
	var columns []string
	if columns, err = rows.Columns(); err != nil {
		err = my.NewError(my.ER_UNKNOWN_ERROR, err.Error())
		return
	}

	// read all rows
	var resultData [][]interface{}
	if resultData, err = readAllRows(rows); err != nil {
		err = my.NewError(my.ER_UNKNOWN_ERROR, err.Error())
		return
	}

	var resultSet *my.Resultset
	if resultSet, err = my.BuildSimpleTextResultset(columns, resultData); err != nil {
		err = my.NewError(my.ER_UNKNOWN_ERROR, err.Error())
		return
	}

	r = &my.Result{
		Status:       0,
		InsertId:     0,
		AffectedRows: 0,
		Resultset:    resultSet,
	}
	return
}

func (c *Cursor) ensureDatabase() (conn *sql.DB, err error) {
	c.curDBLock.Lock()
	defer c.curDBLock.Unlock()

	if c.curDB == "" {
		err = my.NewError(my.ER_NO_DB_ERROR, "select database before any query")
		return
	}

	conn = c.curDBInstance

	return
}

func (c *Cursor) detectColumnType(typeStr string) (typeByte uint8) {
	typeStr = strings.ToUpper(typeStr)

	if strings.Contains(typeStr, "INT") {
		return my.MYSQL_TYPE_LONGLONG
	} else if strings.Contains(typeStr, "CHAR") || strings.Contains(typeStr, "CLOB") ||
		strings.Contains(typeStr, "TEXT") {
		return my.MYSQL_TYPE_VAR_STRING
	} else if strings.Contains(typeStr, "BLOB") || typeStr == "" {
		return my.MYSQL_TYPE_LONG_BLOB
	} else if strings.Contains(typeStr, "REAL") || strings.Contains(typeStr, "FLOA") ||
		strings.Contains(typeStr, "DOUB") {
		return my.MYSQL_TYPE_DOUBLE
	} else if strings.Contains(typeStr, "BOOLEAN") {
		return my.MYSQL_TYPE_BIT
	} else if strings.Contains(typeStr, "TIMESTAMP") || strings.Contains(typeStr, "DATETIME") {
		return my.MYSQL_TYPE_TIMESTAMP
	} else if strings.Contains(typeStr, "TIME") {
		return my.MYSQL_TYPE_TIME
	} else if strings.Contains(typeStr, "DATE") {
		return my.MYSQL_TYPE_DATE
	} else {
		return my.MYSQL_TYPE_LONG_BLOB
	}
}

// UseDB handle COM_INIT_DB command, you can check whether the dbName is valid, or other.
func (c *Cursor) UseDB(dbName string) (err error) {
	c.curDBLock.Lock()
	defer c.curDBLock.Unlock()

	// test if the database name is a valid database id
	if !dbIDRegex.MatchString(dbName) {
		// invalid database
		return my.NewError(my.ER_BAD_DB_ERROR, fmt.Sprintf("invalid database: %v", dbName))
	}

	// connect database
	cfg := client.NewConfig()
	cfg.DatabaseID = dbName

	var db *sql.DB

	if db, err = sql.Open("covenantsql", cfg.FormatDSN()); err != nil {
		return
	}

	c.curDB = dbName
	c.curDBInstance = db

	return
}

// HandleQuery handle COM_QUERY comamnd, like SELECT, INSERT, UPDATE, etc...
// if Result has a Resultset (SELECT, SHOW, etc...), we will send this as the repsonse, otherwise, we will send Result.
func (c *Cursor) HandleQuery(query string) (r *my.Result, err error) {
	var conn *sql.DB

	if conn, err = c.ensureDatabase(); err != nil {
		return
	}

	// send empty result for variables query/table listing
	if emptyResultQuery.MatchString(query) {
		// return empty result
		return &my.Result{
			Status:       0,
			InsertId:     0,
			AffectedRows: 0,
			Resultset:    nil,
		}, nil
	}

	// use database query, same logic as COM_INIT_DB
	if matches := useDatabaseQuery.FindStringSubmatch(query); len(matches) > 1 {
		dbID := matches[1]

		if err = c.UseDB(dbID); err == nil {
			r = &my.Result{
				Status:       0,
				InsertId:     0,
				AffectedRows: 0,
				Resultset:    nil,
			}
		}

		return
	}

	// as normal query
	if readQuery.MatchString(query) {
		var rows *sql.Rows
		if rows, err = conn.Query(query); err != nil {
			err = my.NewError(my.ER_UNKNOWN_ERROR, err.Error())
			return
		}

		// build result set
		return c.buildResultSet(rows)
	}

	var result sql.Result
	if result, err = conn.Exec(query); err != nil {
		err = my.NewError(my.ER_UNKNOWN_ERROR, err.Error())
		return
	}

	lastInsertId, _ := result.LastInsertId()
	affectedRows, _ := result.RowsAffected()

	r = &my.Result{
		Status:       0,
		InsertId:     uint64(lastInsertId),
		AffectedRows: uint64(affectedRows),
		Resultset:    nil,
	}

	return
}

// HandleFieldList handle COM_FILED_LIST command.
func (c *Cursor) HandleFieldList(table string, fieldWildcard string) (fields []*my.Field, err error) {
	var conn *sql.DB

	if conn, err = c.ensureDatabase(); err != nil {
		return
	}

	// send show tables command
	var columns *sql.Rows
	if columns, err = conn.Query(fmt.Sprintf("DESC `%s`", table)); err != nil {
		// wrap error
		err = my.NewError(my.ER_UNKNOWN_ERROR, err.Error())
		return
	}

	defer columns.Close()

	// transform the sql wildcard to glob pattern
	var fieldGlob string

	if fieldWildcard != "" {
		fieldGlob = strings.NewReplacer("_", "?", "%", "*").Replace(fieldWildcard)
	}

	var cid, defaultValue interface{}
	var columnName, typeString string
	var isNotNull, isPK bool

	for columns.Next() {
		if err = columns.Scan(&cid, &columnName, &typeString, &isNotNull, &defaultValue, &isPK); err != nil {
			return
		}

		if fieldGlob != "" {
			if matched, _ := filepath.Match(fieldGlob, columnName); !matched {
				continue
			}
		}

		// process flag
		colFlag := uint16(0)

		if isNotNull {
			colFlag |= my.NOT_NULL_FLAG
		}
		if isPK {
			colFlag |= my.NOT_NULL_FLAG
			colFlag |= my.PRI_KEY_FLAG
		}

		fields = append(fields, &my.Field{
			Name:         []byte(columnName),
			OrgName:      []byte(columnName),
			Table:        []byte(table),
			OrgTable:     []byte(table),
			Schema:       []byte(c.curDB),
			Flag:         colFlag,
			Charset:      uint16(my.DEFAULT_COLLATION_ID),
			ColumnLength: 0, // no column length specified
			Type:         c.detectColumnType(typeString),
		})
	}

	return
}

// HandleStmtPrepare handle COM_STMT_PREPARE, params is the param number for this statement, columns is the column number
// context will be used later for statement execute.
func (c *Cursor) HandleStmtPrepare(query string) (params int, columns int, context interface{}, err error) {
	// TODO(xq26144), not implemented
	// According to the libmysql standard: https://github.com/mysql/mysql-server/blob/8.0/libmysql/libmysql.cc#L1599
	// the COM_STMT_PREPARE should return the correct bind parameter count (which can be implemented by newly created parser)
	// and should return the correct number of return fields (which can not be implemented right now with new query plan logic embedded)

	err = my.NewError(my.ER_NOT_SUPPORTED_YET, "stmt prepare is not supported yet")
	return
}

// HandleStmtExecute handle COM_STMT_EXECUTE, context is the previous one set in prepare
// query is the statement prepare query, and args is the params for this statement.
func (c *Cursor) HandleStmtExecute(context interface{}, query string, args []interface{}) (result *my.Result, err error) {
	// same to COM_STMT_PREPARE
	err = my.NewError(my.ER_NOT_SUPPORTED_YET, "stmt execute is not supported yet")
	return
}

// HandleStmtClose handle COM_STMT_CLOSE, context is the previous one set in prepare
// this handler has no response.
func (c *Cursor) HandleStmtClose(context interface{}) (err error) {
	return
}

// HandleOtherCommand handle any other command that is not currently handled by the library,
// default implementation for this method will return an ER_UNKNOWN_ERROR.
func (c *Cursor) HandleOtherCommand(cmd byte, data []byte) (err error) {
	return my.NewError(my.ER_UNKNOWN_ERROR, fmt.Sprintf("command %d is not supported now", cmd))
}
