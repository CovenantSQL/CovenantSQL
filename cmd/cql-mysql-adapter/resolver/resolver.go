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

package resolver

import (
	"fmt"
	"io"
	"strings"
	"sync/atomic"

	"github.com/CovenantSQL/sqlparser"
	"github.com/pkg/errors"
)

var (
	// ErrQueryLogicError defines query logic error.
	ErrQueryLogicError = errors.New("contains logic error in query")
	// ErrTooMuchQueries defines two much queries in a single request.
	ErrTooMuchQueries = errors.New("too much queries")

	columnsShowCreateTable = []string{"sql"}
	columnsShowTableInfo   = []string{"cid", "name", "type", "notnull", "dflt_value", "pk"}
	columnsShowIndex       = []string{"name"}
	columnsShowTables      = []string{"name"}
	columnsExplain         = []string{"addr", "opcode", "p1", "p2", "p3", "p4", "p5", "comment"}
)

// TableColumnMapping defines a table column mapping object.
type TableColumnMapping struct {
	TableDict    map[string]int
	TableColumns []*TableColumns
}

// TableColumns defines a table columns mapping object.
type TableColumns struct {
	Table   string
	Columns []string
}

// NewTableColumnMapping returns new table column mapping object.
func NewTableColumnMapping() *TableColumnMapping {
	return &TableColumnMapping{
		TableDict: make(map[string]int),
	}
}

// Add table and columns to column mapping object.
func (m *TableColumnMapping) Add(table string, columns []string) {
	if o, ok := m.TableDict[table]; ok {
		if len(m.TableColumns) > o && m.TableColumns[o].Table == table {
			m.TableColumns[o].Columns = append(m.TableColumns[o].Columns[:0], columns...)
		}
	} else {
		m.TableColumns = append(m.TableColumns, &TableColumns{
			Table:   table,
			Columns: columns,
		})
		m.TableDict[table] = len(m.TableColumns) - 1
	}
}

// AppendColumn appends new records table to column mapping.
func (m *TableColumnMapping) AppendColumn(table string, columns []string) {
	if o, ok := m.TableDict[table]; ok {
		if len(m.TableColumns) > o && m.TableColumns[o].Table == table {
			m.TableColumns[o].Columns = append(m.TableColumns[o].Columns, columns...)
		}
	} else {
		m.TableColumns = append(m.TableColumns, &TableColumns{
			Table:   table,
			Columns: columns,
		})
		m.TableDict[table] = len(m.TableColumns) - 1
	}
}

// UnionColumn union and distinct records in table to column mapping.
func (m *TableColumnMapping) UnionColumn(table string, columns []string) {
	if o, ok := m.TableDict[table]; ok {
		if len(m.TableColumns) > o && m.TableColumns[o].Table == table {
			columnMap := make(map[string]bool, len(m.TableColumns[o].Columns))
			for _, c := range m.TableColumns[o].Columns {
				columnMap[c] = true
			}
			for _, c := range columns {
				if !columnMap[c] {
					m.TableColumns[o].Columns = append(m.TableColumns[o].Columns, c)
				}
			}
		}
	} else {
		m.TableColumns = append(m.TableColumns, &TableColumns{
			Table:   table,
			Columns: columns,
		})
		m.TableDict[table] = len(m.TableColumns) - 1
	}
}

// Clear clears the table column mapping object.
func (m *TableColumnMapping) Clear() {
	m.TableDict = make(map[string]int)
	m.TableColumns = m.TableColumns[:0]
}

// Union union distinct two table column mapping object.
func (m *TableColumnMapping) Union(o *TableColumnMapping) {
	if o == nil {
		return
	}

	for _, v := range o.TableColumns {
		if _, exists := m.TableDict[v.Table]; !exists {
			// add
			m.Add(v.Table, v.Columns)
		}
	}
}

// Update update current column mapping object using new records.
func (m *TableColumnMapping) Update(o *TableColumnMapping) {
	if o == nil {
		return
	}

	for _, v := range o.TableColumns {
		m.Add(v.Table, v.Columns)
	}
}

// Merge append new records to current column mapping object.
func (m *TableColumnMapping) Merge(o *TableColumnMapping) {
	if o == nil {
		return
	}

	for _, v := range o.TableColumns {
		m.AppendColumn(v.Table, v.Columns)
	}
}

// MergeDistinct union distinct two table column mapping object records.
func (m *TableColumnMapping) MergeDistinct(o *TableColumnMapping) {
	if o == nil {
		return
	}

	for _, v := range o.TableColumns {
		m.UnionColumn(v.Table, v.Columns)
	}
}

// Get get table columns from the mapping object.
func (m *TableColumnMapping) Get(table string) (columns []string, exists bool) {
	var o int
	if o, exists = m.TableDict[table]; exists {
		if len(m.TableColumns) > o && m.TableColumns[o].Table == table {
			columns = m.TableColumns[o].Columns[:]
			return
		}
	}

	return
}

// Resolver defines the query resolver processor object.
type Resolver struct {
	Meta *MetaHandler
}

// NewResolver returns a new query resolver processor object.
func NewResolver() *Resolver {
	return &Resolver{
		Meta: NewMetaHandler(),
	}
}

// ReloadMeta reloads underlying meta handler.
func (r *Resolver) ReloadMeta() {
	r.Meta.ReloadMeta()
}

// Close close the resolver especially the underlying meta handler.
func (r *Resolver) Close() {
	// stop meta refresher
	r.Meta.Stop()
	// close all underlying database connection
	r.Meta.Close()
}

// RegisterDB register database connection to resolver.
func (r *Resolver) RegisterDB(dbID string, conn DBHandler) (added bool) {
	return r.Meta.AddConn(dbID, conn)
}

// GetDB returns database connection of a database.
func (r *Resolver) GetDB(dbID string) (DBHandler, bool) {
	return r.Meta.GetConn(dbID)
}

// ResolveQuery parses string query and returns multiple query resolver result.
func (r *Resolver) ResolveQuery(dbID string, query string) (queries []*Query, err error) {
	return r.resolveQuery(dbID, query, -1)
}

// ResolveSingleQuery parse string query and make sure there is only one query in the query string.
func (r *Resolver) ResolveSingleQuery(dbID string, query string) (q *Query, err error) {
	var queries []*Query
	if queries, err = r.resolveQuery(dbID, query, 1); err == nil && len(queries) > 0 {
		q = queries[0]
	}
	return
}

// Resolve process sqlparser statement and returns resolved result.
func (r *Resolver) Resolve(dbID string, stmt sqlparser.Statement) (q *Query, err error) {
	q = &Query{
		Stmt:     stmt,
		Database: dbID,
	}
	if q.ParamCount, err = r.BuildParamCount(stmt); err != nil {
		return
	}
	var tableSeq int32
	if q.ResultColumns, err = r.BuildResultColumns(&tableSeq, dbID, stmt); err != nil {
		return
	}

	return
}

func (r *Resolver) resolveQuery(dbID string, query string, maxQueries int) (queries []*Query, err error) {
	var (
		tokenizer  = sqlparser.NewStringTokenizer(query)
		stmt       sqlparser.Statement
		q          *Query
		lastPos    int
		queryCount int
	)

	// resolve :name to ValArg, ? to PosArg type
	tokenizer.SeparatePositionalArgs = true

	for {
		stmt, err = sqlparser.ParseNext(tokenizer)

		if err != nil && err != io.EOF {
			return
		}

		if err == io.EOF {
			err = nil
			break
		}

		queryCount++

		if maxQueries >= 0 && queryCount > maxQueries {
			err = ErrTooMuchQueries
			return
		}

		if q, err = r.Resolve(dbID, stmt); err != nil {
			return
		}

		q.Query = strings.TrimSpace(query[lastPos : tokenizer.Position-1])
		lastPos = tokenizer.Position
		queries = append(queries, q)
	}

	return
}

// BuildParamCount process sqlparser statement and returns the required parameter count in query.
func (r *Resolver) BuildParamCount(stmt sqlparser.Statement) (params int, err error) {
	params = 0
	argDedup := make(map[string]bool)
	err = sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		if v, ok := node.(*sqlparser.SQLVal); ok && (v.Type == sqlparser.ValArg || v.Type == sqlparser.PosArg) {
			argVal := string(v.Val)
			if !argDedup[argVal] {
				argDedup[argVal] = true
				params++
			}
		}
		return true, nil
	}, stmt)
	return
}

func (r *Resolver) buildResultColumnsWithTableName(tableSeq *int32, dbID string, stmt sqlparser.SQLNode) (
	columnMapping *TableColumnMapping, err error) {
	var (
		tb                 = sqlparser.NewTrackedBuffer(nil)
		tempTableName      string
		tempColumnsMapping *TableColumnMapping
	)
	columnMapping = NewTableColumnMapping()

	switch s := stmt.(type) {
	case *sqlparser.AliasedTableExpr:
		if !s.As.IsEmpty() {
			tempTableName = s.As.String()
		} else {
			tempTableName = fmt.Sprintf("_t%d", atomic.AddInt32(tableSeq, 1))
		}
		if tempColumnsMapping, err = r.buildResultColumnsWithTableName(tableSeq, dbID, s.Expr); err != nil {
			return
		}
		for _, v := range tempColumnsMapping.TableColumns {
			// merge columns in sub columns
			// case: SELECT * FROM (test, test2) AS e;
			// will produce columns in test and test2
			columnMapping.AppendColumn(tempTableName, v.Columns)
		}
	case *sqlparser.ParenTableExpr:
		for _, e := range s.Exprs {
			if tempColumnsMapping, err = r.buildResultColumnsWithTableName(tableSeq, dbID, e); err != nil {
				return
			}
			columnMapping.Merge(tempColumnsMapping)
		}
	case *sqlparser.JoinTableExpr:
		// no semi join, all fields should be accessible in projection
		if tempColumnsMapping, err = r.buildResultColumnsWithTableName(tableSeq, dbID, s.LeftExpr); err != nil {
			return
		}
		columnMapping.Merge(tempColumnsMapping)
		if tempColumnsMapping, err = r.buildResultColumnsWithTableName(tableSeq, dbID, s.RightExpr); err != nil {
			return
		}
		columnMapping.Merge(tempColumnsMapping)
	case *sqlparser.Subquery:
		// allocate anonymous name
		var tempColumns []string
		tempTableName = fmt.Sprintf("_t%d", atomic.AddInt32(tableSeq, 1))
		if tempColumns, err = r.BuildResultColumns(tableSeq, dbID, s.Select); err != nil {
			return
		}
		columnMapping.Add(tempTableName, tempColumns)
	case sqlparser.TableName:
		tb.Reset()
		var tempColumns []string
		tempTableName = tb.WriteNode(s).String()
		// load table columns from meta
		if tempColumns, err = r.Meta.GetTable(dbID, tempTableName); err != nil {
			err = errors.Wrapf(err, "no such table: %v.%s", dbID, tempTableName)
			return
		}
		columnMapping.Add(tempTableName, tempColumns)
	default:
		// invalid query
		err = errors.Wrapf(ErrQueryLogicError, "invalid query")
	}
	return
}

// BuildResultColumns returns the result columns (detected name and count) of sqlparser statement.
func (r *Resolver) BuildResultColumns(tableSeq *int32, dbID string, stmt sqlparser.Statement) (columns []string, err error) {
	switch s := stmt.(type) {
	case *sqlparser.Show:
		switch s.Type {
		case "table":
			if s.ShowCreate {
				columns = append(columns, columnsShowCreateTable...)
			} else {
				columns = append(columns, columnsShowTableInfo...)
			}
		case "index":
			columns = append(columns, columnsShowIndex...)
		case "tables":
			columns = append(columns, columnsShowTables...)
		}
	case *sqlparser.Explain:
		columns = append(columns, columnsExplain...)
	case *sqlparser.Union:
		// columns should be equalized between left and right fork
		var tempColumns []string
		if tempColumns, err = r.BuildResultColumns(tableSeq, dbID, s.Left); err != nil {
			return
		}
		columns = append(columns, tempColumns...)
	case *sqlparser.Select:
		// further processing
		tb := sqlparser.NewTrackedBuffer(nil)

		// check if contains star expression
		var (
			containsStarExpression bool
			tableColumns           = NewTableColumnMapping()
		)

		for _, re := range s.SelectExprs {
			if _, ok := re.(*sqlparser.StarExpr); ok {
				containsStarExpression = true
			}
		}

		// load from
		if containsStarExpression {
			for _, rfe := range s.From {
				var tempTableColumns *TableColumnMapping
				if tempTableColumns, err = r.buildResultColumnsWithTableName(tableSeq, dbID, rfe); err != nil {
					return
				}

				tableColumns.Merge(tempTableColumns)
			}
		}

		for _, re := range s.SelectExprs {
			switch e := re.(type) {
			case *sqlparser.AliasedExpr:
				tb.Reset()
				var column string
				if !e.As.IsEmpty() {
					column = e.As.String()
				} else {
					column = tb.WriteNode(e.Expr).String()
				}
				columns = append(columns, column)
			case *sqlparser.StarExpr:
				// * column
				// build args from the relied statement or table
				if !e.TableName.IsEmpty() {
					tb.Reset()
					relyTableName := tb.WriteNode(e.TableName).String()
					if subColumns, exists := tableColumns.Get(relyTableName); exists {
						columns = append(columns, subColumns...)
					} else {
						// not exists, err
						err = errors.Wrapf(ErrQueryLogicError, "table %v not found", relyTableName)
						return
					}
				} else {
					// combine all columns
					for _, v := range tableColumns.TableColumns {
						columns = append(columns, v.Columns...)
					}
				}
			}
		}
	case *sqlparser.ParenSelect:
		// this branch should not be reached
	default:
		// query contains no columns
	}

	return
}
