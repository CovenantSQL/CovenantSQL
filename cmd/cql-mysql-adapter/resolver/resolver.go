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
	"sync/atomic"

	"github.com/CovenantSQL/sqlparser"
	"github.com/pkg/errors"
)

var (
	ErrQueryLogicError = errors.New("contains logic error in query")
	ErrTooMuchQueries  = errors.New("too much queries")

	columnsShowCreateTable = []string{"sql"}
	columnsShowTableInfo   = []string{"cid", "name", "type", "notnull", "dflt_value", "pk"}
	columnsShowIndex       = []string{"name"}
	columnsShowTables      = []string{"name"}
	columnsExplain         = []string{"addr", "opcode", "p1", "p2", "p3", "p4", "p5", "comment"}
)

type tableColumnMapping struct {
	tableDict    map[string]int
	tableColumns []*tableColumns
}

type tableColumns struct {
	table   string
	columns []string
}

func newTableColumnMapping() *tableColumnMapping {
	return &tableColumnMapping{
		tableDict: make(map[string]int),
	}
}

func (m *tableColumnMapping) add(table string, columns []string) {
	if o, ok := m.tableDict[table]; ok {
		if len(m.tableColumns) > o && m.tableColumns[o].table == table {
			m.tableColumns[o].columns = append(m.tableColumns[o].columns[:0], columns...)
		}
	} else {
		m.tableColumns = append(m.tableColumns, &tableColumns{
			table:   table,
			columns: columns,
		})
		m.tableDict[table] = len(m.tableColumns) - 1
	}
}

func (m *tableColumnMapping) appendColumn(table string, columns []string) {
	if o, ok := m.tableDict[table]; ok {
		if len(m.tableColumns) > o && m.tableColumns[o].table == table {
			m.tableColumns[o].columns = append(m.tableColumns[o].columns, columns...)
		}
	} else {
		m.tableColumns = append(m.tableColumns, &tableColumns{
			table:   table,
			columns: columns,
		})
		m.tableDict[table] = len(m.tableColumns) - 1
	}
}

func (m *tableColumnMapping) clear() {
	m.tableDict = make(map[string]int)
	m.tableColumns = m.tableColumns[:0]
}

func (m *tableColumnMapping) union(o *tableColumnMapping) {
	if o == nil {
		return
	}

	for _, v := range o.tableColumns {
		if _, exists := m.tableDict[v.table]; !exists {
			// add
			m.add(v.table, v.columns)
		}
	}
}

func (m *tableColumnMapping) update(o *tableColumnMapping) {
	if o == nil {
		return
	}

	for _, v := range o.tableColumns {
		m.add(v.table, v.columns)
	}
}

func (m *tableColumnMapping) merge(o *tableColumnMapping) {
	if o == nil {
		return
	}

	for _, v := range o.tableColumns {
		m.appendColumn(v.table, v.columns)
	}
}

func (m *tableColumnMapping) get(table string) (columns []string, exists bool) {
	var o int
	if o, exists = m.tableDict[table]; exists {
		if len(m.tableColumns) > o && m.tableColumns[o].table == table {
			columns = m.tableColumns[o].columns[:]
			return
		}
	}

	return
}

type Resolver struct {
	meta *metaHandler
}

func NewResolver() *Resolver {
	return &Resolver{
		meta: NewMetaHandler(),
	}
}

func (r *Resolver) Close() {
	r.meta.stop()
}

func (r *Resolver) RegisterDB(dbID string, conn dbHandler) {
	r.meta.addConn(dbID, conn)
}

func (r *Resolver) ResolveQuery(dbID string, query string) (queries []*Query, err error) {
	return r.resolveQuery(dbID, query, -1)
}

func (r *Resolver) ResolveSingleQuery(dbID string, query string) (q *Query, err error) {
	var queries []*Query
	if queries, err = r.resolveQuery(dbID, query, 1); err == nil && len(queries) > 0 {
		q = queries[0]
	}
	return
}

func (r *Resolver) Resolve(dbID string, stmt sqlparser.Statement) (q *Query, err error) {
	q = &Query{
		Stmt: stmt,
	}
	if q.ParamCount, err = r.buildParamCount(stmt); err != nil {
		return
	}
	var tableSeq int32
	if q.ResultColumns, err = r.buildResultColumns(&tableSeq, dbID, stmt); err != nil {
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

		q = &Query{
			Stmt:  stmt,
			Query: query[lastPos : tokenizer.Position-1],
		}
		lastPos = tokenizer.Position + 1
		if q.ParamCount, err = r.buildParamCount(stmt); err != nil {
			return
		}
		var tableSeq int32
		if q.ResultColumns, err = r.buildResultColumns(&tableSeq, dbID, stmt); err != nil {
			return
		}
		queries = append(queries, q)
	}

	return
}

func (r *Resolver) buildParamCount(stmt sqlparser.Statement) (params int, err error) {
	params = 0
	argDedup := make(map[string]bool)
	err = sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		if v, ok := node.(*sqlparser.SQLVal); ok && v.Type == sqlparser.ValArg {
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
	columnMapping *tableColumnMapping, err error) {
	var (
		tb                 = sqlparser.NewTrackedBuffer(nil)
		tempTableName      string
		tempColumnsMapping *tableColumnMapping
	)
	columnMapping = newTableColumnMapping()

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
		for _, v := range tempColumnsMapping.tableColumns {
			// merge columns in sub columns
			// case: SELECT * FROM (test, test2) AS e;
			// will produce columns in test and test2
			columnMapping.appendColumn(tempTableName, v.columns)
		}
	case *sqlparser.ParenTableExpr:
		for _, e := range s.Exprs {
			if tempColumnsMapping, err = r.buildResultColumnsWithTableName(tableSeq, dbID, e); err != nil {
				return
			}
			columnMapping.merge(tempColumnsMapping)
		}
	case *sqlparser.JoinTableExpr:
		// no semi join, all fields should be accessible in projection
		if tempColumnsMapping, err = r.buildResultColumnsWithTableName(tableSeq, dbID, s.LeftExpr); err != nil {
			return
		}
		columnMapping.merge(tempColumnsMapping)
		if tempColumnsMapping, err = r.buildResultColumnsWithTableName(tableSeq, dbID, s.RightExpr); err != nil {
			return
		}
		columnMapping.merge(tempColumnsMapping)
	case *sqlparser.Subquery:
		// allocate anonymous name
		var tempColumns []string
		tempTableName = fmt.Sprintf("_t%d", atomic.AddInt32(tableSeq, 1))
		if tempColumns, err = r.buildResultColumns(tableSeq, dbID, s.Select); err != nil {
			return
		}
		columnMapping.add(tempTableName, tempColumns)
	case sqlparser.TableName:
		tb.Reset()
		var tempColumns []string
		tempTableName = tb.WriteNode(s).String()
		// load table columns from meta
		if tempColumns, err = r.meta.getTable(dbID, tempTableName); err != nil {
			err = errors.Wrapf(err, "no such table: %v.%s", dbID, tempTableName)
			return
		}
		columnMapping.add(tempTableName, tempColumns)
	default:
		// invalid query
		err = errors.Wrapf(ErrQueryLogicError, "invalid query")
	}
	return
}

func (r *Resolver) buildResultColumns(tableSeq *int32, dbID string, stmt sqlparser.Statement) (columns []string, err error) {
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
		if tempColumns, err = r.buildResultColumns(tableSeq, dbID, s.Left); err != nil {
			return
		}
		columns = append(columns, tempColumns...)
	case *sqlparser.Select:
		// further processing
		tb := sqlparser.NewTrackedBuffer(nil)

		// check if contains star expression
		var (
			containsStarExpression bool
			tableColumns           = newTableColumnMapping()
		)

		for _, re := range s.SelectExprs {
			if _, ok := re.(*sqlparser.StarExpr); ok {
				containsStarExpression = true
			}
		}

		// load from
		if containsStarExpression {
			for _, rfe := range s.From {
				var tempTableColumns *tableColumnMapping
				if tempTableColumns, err = r.buildResultColumnsWithTableName(tableSeq, dbID, rfe); err != nil {
					return
				}

				tableColumns.merge(tempTableColumns)
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
					if subColumns, exists := tableColumns.get(relyTableName); exists {
						columns = append(columns, subColumns...)
					} else {
						// not exists, err
						err = errors.Wrapf(ErrQueryLogicError, "table %v not found", relyTableName)
						return
					}
				} else {
					// combine all columns
					for _, v := range tableColumns.tableColumns {
						columns = append(columns, v.columns...)
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
