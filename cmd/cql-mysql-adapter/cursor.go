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
	"io"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/CovenantSQL/CovenantSQL/client"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/CovenantSQL/sqlparser"
	my "github.com/siddontang/go-mysql/mysql"
)

// preset column counts for special statements
const (
	columnCountShowCreateTable = 1 // sql
	columnCountShowTableInfo   = 6 // cid|name|type|notnull|dflt_value|pk
	columnCountShowIndex       = 1 // name
	columnCountShowTables      = 1 // name
	columnCountExplain         = 8 // addr|opcode|p1|p2|p3|p4|p5|comment
)

var (
	dbIDRegex                     = regexp.MustCompile("^[a-zA-Z0-9_\\.]+$")
	specialSelectQuery            = regexp.MustCompile("^(?i)SELECT\\s+(DATABASE|USER)\\(\\)\\s*;?\\s*$")
	emptyResultQuery              = regexp.MustCompile("^(?i)\\s*(?:/\\*.*?\\*/)?\\s*(?:SET|ROLLBACK).*$")
	emptyResultWithResultSetQuery = regexp.MustCompile("^(?i)\\s*(?:/\\*.*?\\*/)?\\s*(?:(?:SELECT\\s+)?@@(?:\\w+\\.)?|SHOW\\s+WARNINGS).*$")
	showVariablesQuery            = regexp.MustCompile("^(?i)\\s*(?:/\\*.*?\\*/)?\\s*SHOW\\s+VARIABLES.*$")
	showDatabasesQuery            = regexp.MustCompile("^(?i)\\s*(?:/\\*.*?\\*/)?\\s*SHOW\\s+DATABASES.*$")
	useDatabaseQuery              = regexp.MustCompile("^(?i)\\s*USE\\s+`?(\\w+)`?\\s*$")
	readQuery                     = regexp.MustCompile("^(?i)\\s*(?:SELECT|SHOW|DESC)")
	mysqlServerVariables          = map[string]interface{}{
		"max_allowed_packet":       255 * 255 * 255,
		"auto_increment_increment": 1,
		"transaction_isolation":    "SERIALIZABLE",
		"tx_isolation":             "SERIALIZABLE",
		"transaction_read_only":    0,
		"tx_read_only":             0,
		"autocommit":               1,
		"character_set_server":     "utf8",
		"collation_server":         "utf8_general_ci",
	}
)

// Cursor is a mysql connection handler, like a cursor of normal database.
type Cursor struct {
	server        *Server
	curDBLock     sync.Mutex
	curDB         string
	curDBInstance *sql.DB
}

// NewCursor returns a new cursor.
func NewCursor(s *Server) (c *Cursor) {
	return &Cursor{server: s}
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

func (c *Cursor) handleSpecialQuery(query string) (r *my.Result, processed bool, err error) {
	if emptyResultQuery.MatchString(query) { // send empty result for variables query/table listing
		// return empty result
		r = &my.Result{
			Status:       0,
			InsertId:     0,
			AffectedRows: 0,
			Resultset:    nil,
		}
		processed = true
	} else if emptyResultWithResultSetQuery.MatchString(query) { // send empty result include non-nil result set
		// return empty result with empty result set
		var (
			resultSet *my.Resultset
			columns   []string
			row       []interface{}
		)

		for k, v := range mysqlServerVariables {
			if strings.Contains(query, k) {
				columns = append(columns, k)
				row = append(row, v)
			}
		}

		if len(columns) == 0 {
			columns = append(columns, "_")
		}

		if row != nil {
			resultSet, _ = my.BuildSimpleTextResultset(columns, [][]interface{}{row})
		} else {
			resultSet, _ = my.BuildSimpleTextResultset(columns, [][]interface{}{})
		}

		if resultSet.RowDatas == nil {
			// force non-empty result set
			resultSet.RowDatas = make([]my.RowData, 0)
		}

		r = &my.Result{
			Status:       0,
			InsertId:     0,
			AffectedRows: 0,
			Resultset:    resultSet,
		}
		processed = true
	} else if showVariablesQuery.MatchString(query) { // send show variables result with custom config
		var rows [][]interface{}

		for k, v := range mysqlServerVariables {
			rows = append(rows, []interface{}{k, v})
		}

		resultSet, _ := my.BuildSimpleTextResultset([]string{"Variable_name", "Value"}, rows)
		r = &my.Result{
			Status:       0,
			InsertId:     0,
			AffectedRows: 0,
			Resultset:    resultSet,
		}
		processed = true
	} else if showDatabasesQuery.MatchString(query) { // send show databases result
		// return result including current database
		var curDBStr string
		c.curDBLock.Lock()
		curDBStr = c.curDB
		c.curDBLock.Unlock()

		var resultSet *my.Resultset

		if curDBStr != "" {
			resultSet, _ = my.BuildSimpleTextResultset([]string{"Database"}, [][]interface{}{{curDBStr}})
		} else {
			resultSet, _ = my.BuildSimpleTextResultset([]string{"Database"}, nil)
		}

		r = &my.Result{
			Status:       0,
			InsertId:     0,
			AffectedRows: 0,
			Resultset:    resultSet,
		}
		processed = true
	} else if matches := useDatabaseQuery.FindStringSubmatch(query); len(matches) > 1 { // use database query, same logic as COM_INIT_DB
		dbID := matches[1]

		processed = true
		if err = c.UseDB(dbID); err == nil {
			r = &my.Result{
				Status:       0,
				InsertId:     0,
				AffectedRows: 0,
				Resultset:    nil,
			}
		}
	} else if matches := specialSelectQuery.FindStringSubmatch(query); len(matches) > 1 {
		// special select database
		// for libmysql trivial implementations
		// https://github.com/mysql/mysql-server/blob/4f1d7cf5fcb11a3f84cff27e37100d7295e7d5ca/client/mysql.cc#L4266

		var resultSet *my.Resultset

		switch strings.ToUpper(matches[1]) {
		case "DATABASE":
			c.curDBLock.Lock()
			resultSet, _ = my.BuildSimpleTextResultset(
				[]string{"DATABASE()"},
				[][]interface{}{{c.curDB}},
			)
			c.curDBLock.Unlock()
		case "USER":
			resultSet, _ = my.BuildSimpleTextResultset(
				[]string{"USER()"},
				[][]interface{}{{c.server.mysqlUser}},
			)
		}

		r = &my.Result{
			Status:       0,
			InsertId:     0,
			AffectedRows: 0,
			Resultset:    resultSet,
		}
		processed = true
	}

	return
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
// if Result has a Resultset (SELECT, SHOW, etc...), we will send this as the response, otherwise, we will send Result.
func (c *Cursor) HandleQuery(query string) (r *my.Result, err error) {
	var processed bool

	log.WithField("query", query).Info("received query")

	if r, processed, err = c.handleSpecialQuery(query); processed {
		return
	}

	var conn *sql.DB

	if conn, err = c.ensureDatabase(); err != nil {
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

	lastInsertID, _ := result.LastInsertId()
	affectedRows, _ := result.RowsAffected()

	r = &my.Result{
		Status:       0,
		InsertId:     uint64(lastInsertID),
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

	var (
		cid, defaultValue      interface{}
		columnName, typeString string
		isNotNull, isPK        bool
	)

	for columns.Next() {
		if err = columns.Scan(&cid, &columnName, &typeString, &isNotNull, &defaultValue, &isPK); err != nil {
			err = my.NewError(my.ER_UNKNOWN_ERROR, err.Error())
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
	// for special queries
	var processed bool
	var r *my.Result

	log.WithField("query", query).Info("received query")

	if r, processed, err = c.handleSpecialQuery(query); processed {
		// set context as result
		context = r
		return
	}

	var conn *sql.DB

	if conn, err = c.ensureDatabase(); err != nil {
		return
	}

	// tokenize commands
	var (
		tokenizer   = sqlparser.NewStringTokenizer(query)
		stmt        sqlparser.Statement
		lastPos     int
		singleQuery string
	)

	for {
		stmt, err = sqlparser.ParseNext(tokenizer)

		// parse statement failed
		if err != nil && err != io.EOF {
			return
		}

		if err == io.EOF {
			err = nil
			break
		}

		if lastPos != 0 {
			// multiple queries
			err = my.NewError(my.ER_SYNTAX_ERROR, "could not prepare multiple statements")
			return
		}

		singleQuery = query[lastPos : tokenizer.Position-1]
		lastPos = tokenizer.Position + 1
	}

	if strings.TrimSpace(singleQuery) == "" {
		// empty query
		err = my.NewError(my.ER_SYNTAX_ERROR, "query is empty")
		return
	}

	// calculate param count for special statements
	switch s := stmt.(type) {
	case *sqlparser.Show:
		params = 0
		switch s.Type {
		case "table":
			if s.ShowCreate {
				columns = columnCountShowCreateTable
			} else {
				columns = columnCountShowTableInfo
			}
		case "index":
			columns = columnCountShowIndex
		case "tables":
			columns = columnCountShowTables
		default:
			err = my.NewError(my.ER_UNKNOWN_ERROR, "unknown query")
		}
		return
	case *sqlparser.Explain:
		params = 0
		columns = columnCountExplain
		return
	case *sqlparser.DDL:
		err = my.NewError(my.ER_UNKNOWN_ERROR, "could not prepare ddl statement")
		return
	}

	// HACK(xq262144), this feature is supported by sqlite only, replace with query plan analysis for mysql engine
	// normal statement, using explain to calculate param and result column count
	// using opcode descriptions in https://www.sqlite.org/opcode.html to resolve

	// resolve parameter count
	var paramCount uint32
	sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		if v, ok := node.(*sqlparser.SQLVal); ok && v.Type == sqlparser.ValArg {
			atomic.AddUint32(&paramCount, 1)
		}
		return true, nil
	}, stmt)

	// build dummy args, golang implementation requires argument to execute explain statement
	dummyArgs := make([]interface{}, paramCount)

	var rows *sql.Rows
	if rows, err = conn.Query("EXPLAIN "+query, dummyArgs...); err != nil {
		err = my.NewError(my.ER_UNKNOWN_ERROR, err.Error())
		return
	}

	defer rows.Close()

	var (
		opcode                    string
		p1, p2                    int
		addr, p3, p4, p5, comment interface{} // not care about this fields
	)

	for rows.Next() {
		if err = rows.Scan(&addr, &opcode, &p1, &p2, &p3, &p4, &p5, &comment); err != nil {
			err = my.NewError(my.ER_UNKNOWN_ERROR, err.Error())
			return
		}

		// p1 for variables is
		if opcode == "Variable" && p1 > params {
			params = p1
		}

		// p2 is the result column count
		if opcode == "ResultRow" {
			columns = p2
		}
	}

	return
}

// HandleStmtExecute handle COM_STMT_EXECUTE, context is the previous one set in prepare
// query is the statement prepare query, and args is the params for this statement.
func (c *Cursor) HandleStmtExecute(context interface{}, query string, args []interface{}) (r *my.Result, err error) {
	// special query
	if context != nil {
		var ok bool
		if r, ok = context.(*my.Result); ok {
			return
		}
	}

	var conn *sql.DB

	if conn, err = c.ensureDatabase(); err != nil {
		return
	}

	// normal query
	if readQuery.MatchString(query) {
		var rows *sql.Rows
		if rows, err = conn.Query(query, args...); err != nil {
			err = my.NewError(my.ER_UNKNOWN_ERROR, err.Error())
			return
		}

		// build result set
		return c.buildResultSet(rows)
	}

	var result sql.Result
	if result, err = conn.Exec(query, args...); err != nil {
		err = my.NewError(my.ER_UNKNOWN_ERROR, err.Error())
		return
	}

	lastInsertID, _ := result.LastInsertId()
	affectedRows, _ := result.RowsAffected()

	r = &my.Result{
		Status:       0,
		InsertId:     uint64(lastInsertID),
		AffectedRows: uint64(affectedRows),
		Resultset:    nil,
	}

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
