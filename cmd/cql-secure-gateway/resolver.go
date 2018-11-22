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
	"strings"

	"github.com/CovenantSQL/CovenantSQL/cmd/cql-mysql-adapter/resolver"
	"github.com/CovenantSQL/sqlparser"
	"github.com/pkg/errors"
)

var (
	ErrInvalidStatement    = errors.New("invalid statement")
	ErrInvalidColumn       = errors.New("invalid column")
	ErrInvalidTable        = errors.New("invalid table")
	ErrAmbiguousColumnName = errors.New("ambiguous column name")
)

type Resolver struct {
	*resolver.Resolver
}

type Query struct {
	*resolver.Query
}

type Columns []*Column

type Computation struct {
	Op       string
	Operands Columns
}

func (c Columns) ResolveColName(col *sqlparser.ColName) (column *Column, err error) {
	if col == nil {
		return
	}

	if col.Qualifier.IsEmpty() {
		return c.ResolveColIdent(col.Name)
	}

	for _, item := range c {
		if strings.EqualFold(item.ColName.Qualifier.Name.String(), col.Qualifier.Name.String()) &&
			item.ColName.Name.Equal(col.Name) {
			column = item
			return
		}
	}

	// not found
	tb := sqlparser.NewTrackedBuffer(nil)
	err = errors.Wrapf(ErrInvalidColumn, "column %s not found", tb.WriteNode(col).String())
	return
}

func (c Columns) ResolveColIdent(col sqlparser.ColIdent) (column *Column, err error) {
	found := false
	for _, item := range c {
		if item.ColName.Name.Equal(col) {
			// found
			if found {
				// ambiguous
				tb1 := sqlparser.NewTrackedBuffer(nil)
				tb2 := sqlparser.NewTrackedBuffer(nil)
				err = errors.Wrapf(ErrAmbiguousColumnName,
					"ambiguous column %s, candidates: %s, %s", col.String(),
					tb1.WriteNode(&column.ColName).String(),
					tb2.WriteNode(&item.ColName).String())
				return
			}

			found = true
			column = item
		}
	}

	if !found {
		err = errors.Wrapf(ErrInvalidColumn, "column %s not found", col.String())
		return
	}

	return
}

func NewColumns(cols ...*Column) Columns {
	return Columns(cols)
}

type Column struct {
	ColName     sqlparser.ColName
	IsPhysical  bool
	Computation *Computation
}

type ColumnResult struct {
	TableName string
	ColName   string
	Ops       [][]string
}

func (r *Resolver) buildExpression(dbID string, expr sqlparser.Expr, originColumns Columns) (cols Columns, err error) {
	if expr == nil {
		return
	}

	var (
		tempColumn  *Column
		tempColumns Columns
	)

	switch e := expr.(type) {
	case *sqlparser.ColName:
		// find column in original columns
		if tempColumn, err = originColumns.ResolveColName(e); err != nil {
			return
		}
		cols = append(cols, tempColumn)
	case *sqlparser.AndExpr:
		if tempColumns, err = r.buildExpression(dbID, e.Left, originColumns); err != nil {
			return
		}
		cols = append(cols, tempColumns...)
		if tempColumns, err = r.buildExpression(dbID, e.Right, originColumns); err != nil {
			return
		}
		cols = append(cols, tempColumns...)
		cols = NewColumns(&Column{
			IsPhysical: false,
			Computation: &Computation{
				Op:       "and",
				Operands: cols,
			},
		})
	case *sqlparser.OrExpr:
		if tempColumns, err = r.buildExpression(dbID, e.Left, originColumns); err != nil {
			return
		}
		cols = append(cols, tempColumns...)
		if tempColumns, err = r.buildExpression(dbID, e.Right, originColumns); err != nil {
			return
		}
		cols = append(cols, tempColumns...)
		cols = NewColumns(&Column{
			IsPhysical: false,
			Computation: &Computation{
				Op:       "or",
				Operands: cols,
			},
		})
	case *sqlparser.NotExpr:
		if tempColumns, err = r.buildExpression(dbID, e.Expr, originColumns); err != nil {
			return
		}
		cols = NewColumns(&Column{
			IsPhysical: false,
			Computation: &Computation{
				Op:       "not",
				Operands: tempColumns,
			},
		})
	case *sqlparser.ParenExpr:
		return r.buildExpression(dbID, e.Expr, originColumns)

		/* literal constant */
	case sqlparser.BoolVal:
		cols = NewColumns(&Column{
			IsPhysical: false,
			Computation: &Computation{
				Op: "bool_literal",
			},
		})
	case *sqlparser.NullVal:
		cols = NewColumns(&Column{
			IsPhysical: false,
			Computation: &Computation{
				Op: "null_literal",
			},
		})
	case *sqlparser.TimeExpr:
		// treat time expression as literal
		cols = NewColumns(&Column{
			IsPhysical: false,
			Computation: &Computation{
				Op: "time_literal",
			},
		})
	case sqlparser.ValTuple:
		for _, v := range e {
			if tempColumns, err = r.buildExpression(dbID, v, originColumns); err != nil {
				return
			}
			cols = append(cols, tempColumns...)
		}
	case *sqlparser.SQLVal:
		switch e.Type {
		case sqlparser.StrVal, sqlparser.HexVal:
			cols = NewColumns(&Column{
				IsPhysical: false,
				Computation: &Computation{
					Op: "string_literal",
				},
			})
		case sqlparser.IntVal, sqlparser.FloatVal, sqlparser.HexNum, sqlparser.BitVal:
			cols = NewColumns(&Column{
				IsPhysical: false,
				Computation: &Computation{
					Op: "numeric_literal",
				},
			})
		case sqlparser.ValArg:
			cols = NewColumns(&Column{
				IsPhysical: false,
				Computation: &Computation{
					Op: "parameter",
				},
			})
		default:
			tb := sqlparser.NewTrackedBuffer(nil)
			err = errors.Wrapf(ErrInvalidField, "invalid field %s", tb.WriteNode(e).String())
		}

		/* expression with operators */
	case *sqlparser.BinaryExpr:
		if tempColumns, err = r.buildExpression(dbID, e.Left, originColumns); err != nil {
			return
		}
		cols = append(cols, tempColumns...)
		if tempColumns, err = r.buildExpression(dbID, e.Right, originColumns); err != nil {
			return
		}
		cols = append(cols, tempColumns...)
		cols = NewColumns(&Column{
			IsPhysical: false,
			Computation: &Computation{
				Op:       "binary_op",
				Operands: cols,
			},
		})
	case *sqlparser.CaseExpr:
		conditionExprs := []sqlparser.Expr{e.Expr}
		resultExprs := []sqlparser.Expr{e.Else}
		for _, w := range e.Whens {
			conditionExprs = append(conditionExprs, w.Cond)
			resultExprs = append(resultExprs, w.Val)
		}

		for _, expr := range conditionExprs {
			if tempColumns, err = r.buildExpression(dbID, expr, originColumns); err != nil {
				return
			}
			cols = append(cols, tempColumns...)
			cols = append(cols, &Column{
				IsPhysical: false,
				Computation: &Computation{
					Op:       "case_condition",
					Operands: tempColumns,
				},
			})
		}

		for _, expr := range resultExprs {
			if tempColumns, err = r.buildExpression(dbID, expr, originColumns); err != nil {
				return
			}
			cols = append(cols, tempColumns...)
		}
		cols = NewColumns(&Column{
			IsPhysical: false,
			Computation: &Computation{
				Op:       "case",
				Operands: cols,
			},
		})
	case *sqlparser.ComparisonExpr:
		for _, expr := range []sqlparser.Expr{e.Left, e.Right, e.Escape} {
			if tempColumns, err = r.buildExpression(dbID, expr, originColumns); err != nil {
				return
			}
			cols = append(cols, tempColumns...)
		}

		cols = NewColumns(&Column{
			IsPhysical: false,
			Computation: &Computation{
				Op:       "compare",
				Operands: cols,
			},
		})
	case *sqlparser.ConvertExpr:
		if tempColumns, err = r.buildExpression(dbID, e.Expr, originColumns); err != nil {
			return
		}
		cols = NewColumns(&Column{
			IsPhysical: false,
			Computation: &Computation{
				Op:       "convert",
				Operands: tempColumns,
			},
		})
	case *sqlparser.ExistsExpr:
		// inject current symbols to select expression parents
		if e.Subquery != nil {
			if tempColumns, err = r.buildSelectStatement(dbID, e.Subquery.Select, originColumns); err != nil {
				return
			}
			cols = NewColumns(&Column{
				IsPhysical: false,
				Computation: &Computation{
					Op:       "exists",
					Operands: tempColumns,
				},
			})
		}
	case *sqlparser.FuncExpr:
		if tempColumns, err = r.buildSelectExprs(dbID, e.Exprs, e.Distinct, originColumns); err != nil {
			return
		}
		cols = NewColumns(&Column{
			IsPhysical: false,
			Computation: &Computation{
				Op:       "function",
				Operands: tempColumns,
			},
		})
	case *sqlparser.GroupConcatExpr:
		if tempColumns, err = r.buildSelectExprs(dbID, e.Exprs, e.Distinct != "", originColumns); err != nil {
			return
		}
		cols = append(cols, tempColumns...)
		if tempColumns, err = r.buildOrderBy(dbID, e.OrderBy, cols); err != nil {
			return
		}
		cols = append(cols, tempColumns...)
		cols = NewColumns(&Column{
			IsPhysical: false,
			Computation: &Computation{
				Op:       "group_concat",
				Operands: cols,
			},
		})
	case *sqlparser.IntervalExpr:
		if tempColumns, err = r.buildExpression(dbID, e.Expr, originColumns); err != nil {
			return
		}
		cols = NewColumns(&Column{
			IsPhysical: false,
			Computation: &Computation{
				Op:       "interval",
				Operands: tempColumns,
			},
		})
	case *sqlparser.IsExpr:
		if tempColumns, err = r.buildExpression(dbID, e.Expr, originColumns); err != nil {
			return
		}
		cols = NewColumns(&Column{
			IsPhysical: false,
			Computation: &Computation{
				Op:       "is",
				Operands: tempColumns,
			},
		})
	case *sqlparser.SubstrExpr:
		if tempColumns, err = r.buildExpression(dbID, e.From, originColumns); err != nil {
			return
		}
		cols = append(cols, tempColumns...)
		if tempColumns, err = r.buildExpression(dbID, e.To, originColumns); err != nil {
			return
		}
		cols = append(cols, tempColumns...)
		if tempColumns, err = r.buildExpression(dbID, e.Name, originColumns); err != nil {
			return
		}
		cols = append(cols, tempColumns...)
		cols = NewColumns(&Column{
			IsPhysical: false,
			Computation: &Computation{
				Op:       "substr",
				Operands: cols,
			},
		})
	case *sqlparser.RangeCond:
		for _, expr := range []sqlparser.Expr{e.Left, e.From, e.To} {
			if tempColumns, err = r.buildExpression(dbID, expr, originColumns); err != nil {
				return
			}
			cols = append(cols, tempColumns...)
		}
		cols = NewColumns(&Column{
			IsPhysical: false,
			Computation: &Computation{
				Op:       "range",
				Operands: cols,
			},
		})
	case *sqlparser.UnaryExpr:
		if tempColumns, err = r.buildExpression(dbID, e.Expr, originColumns); err != nil {
			return
		}
		cols = NewColumns(&Column{
			IsPhysical: false,
			Computation: &Computation{
				Op:       "unary_op",
				Operands: cols,
			},
		})
	case *sqlparser.Subquery:
		// should contains single result
		if tempColumns, err = r.buildSelectStatement(dbID, e.Select, originColumns); err != nil {
			return
		}

		if len(tempColumns) != 1 {
			err = errors.Wrapf(ErrInvalidStatement, "sub query returns %d columns", len(tempColumns))
			return
		}

		cols = NewColumns(&Column{
			IsPhysical: false,
			Computation: &Computation{
				Op:       "sub_query",
				Operands: tempColumns,
			},
		})
	default:
		tb := sqlparser.NewTrackedBuffer(nil)
		err = errors.Wrapf(ErrInvalidField, "invalid expression %s", tb.WriteNode(expr).String())
		return
	}

	return
}

func (r *Resolver) buildOrder(dbID string, order *sqlparser.Order, originColumns Columns) (cols Columns, err error) {
	if order == nil {
		return
	}

	if cols, err = r.buildExpression(dbID, order.Expr, originColumns); err != nil {
		return
	}

	if len(cols) != 1 {
		err = errors.Wrapf(ErrInvalidStatement, "sub query returns %d columns", len(cols))
		return
	}

	// add order by operator on it
	cols = NewColumns(&Column{
		IsPhysical: false,
		Computation: &Computation{
			Op:       "order_by",
			Operands: cols,
		},
	})
	return
}

func (r *Resolver) buildGroupBy(dbID string, groupBy sqlparser.GroupBy, originColumns Columns) (cols Columns, err error) {
	var tempColumns Columns
	for _, ge := range groupBy {
		if tempColumns, err = r.buildExpression(dbID, ge, originColumns); err != nil {
			return
		}
		cols = append(cols, tempColumns...)

		// add group by column operator to it
		cols = append(cols, &Column{
			IsPhysical: false,
			Computation: &Computation{
				Op:       "group_by",
				Operands: tempColumns,
			},
		})
	}
	return
}

func (r *Resolver) buildOrderBy(dbID string, orders sqlparser.OrderBy, originColumns Columns) (cols Columns, err error) {
	var tempColumns Columns
	for _, oe := range orders {
		if tempColumns, err = r.buildOrder(dbID, oe, originColumns); err != nil {
			return
		}
		cols = append(cols, tempColumns...)
	}
	return
}

func (r *Resolver) buildWhere(dbID string, whereCond *sqlparser.Where, originColumns Columns) (cols Columns, err error) {
	if whereCond == nil {
		return
	}

	return r.buildExpression(dbID, whereCond.Expr, originColumns)
}

func (r *Resolver) buildAllColumnInTable(dbID string, tableName sqlparser.TableName) (cols Columns, err error) {
	var (
		columns []string
		tblName = tableName.Name.String()
	)

	if columns, err = r.Meta.GetTable(dbID, tblName); err != nil {
		return
	}

	for _, c := range columns {
		cols = append(cols, &Column{
			ColName: sqlparser.ColName{
				Qualifier: tableName,
				Name:      sqlparser.NewColIdent(c),
			},
			IsPhysical: true,
		})
	}
	return
}

func (r *Resolver) buildAllColumnInTableExpr(dbID string, tableExpr sqlparser.TableExpr, extraColumns Columns) (cols Columns, err error) {
	switch te := tableExpr.(type) {
	case *sqlparser.AliasedTableExpr:
		var tempColumns Columns

		switch se := te.Expr.(type) {
		case *sqlparser.Subquery:
			if tempColumns, err = r.buildSelectStatement(dbID, se.Select, nil); err != nil {
				return
			}
		case sqlparser.TableName:
			if tempColumns, err = r.buildAllColumnInTable(dbID, se); err != nil {
				return
			}
		default:
			tb := sqlparser.NewTrackedBuffer(nil)
			err = errors.Wrapf(ErrInvalidStatement, "invalid table expression %s", tb.WriteNode(te.Expr).String())
			return
		}

		// table alias
		if !te.As.IsEmpty() {
			for _, c := range tempColumns {
				cols = append(cols, &Column{
					ColName: sqlparser.ColName{
						Qualifier: sqlparser.TableName{
							Name: te.As,
						},
						Name: c.ColName.Name,
					},
					IsPhysical: false,
					Computation: &Computation{
						Op:       "alias",
						Operands: Columns([]*Column{c}),
					},
				})
			}
		} else {
			cols = tempColumns
		}

		return
	case *sqlparser.JoinTableExpr:
		var (
			leftColumns  Columns
			rightColumns Columns
			tempColumns  Columns
		)
		if leftColumns, err = r.buildAllColumnInTableExpr(dbID, te.LeftExpr, extraColumns); err != nil {
			return
		}
		cols = append(cols, leftColumns...)
		if rightColumns, err = r.buildAllColumnInTableExpr(dbID, te.RightExpr, extraColumns); err != nil {
			return
		}
		cols = append(cols, rightColumns...)
		// ensure using column exists in both sets
		for _, uc := range te.Condition.Using {
			// should be in left and right
			args := make(Columns, 0, 2)

			for _, cs := range []Columns{leftColumns, rightColumns} {
				found := false

				for _, c := range cs {
					if uc.Equal(c.ColName.Name) {
						found = true
						args = append(args, c)
						break
					}
				}

				if !found {
					err = errors.Wrapf(ErrInvalidColumn, "column not found %s", uc.String())
					return
				}
			}

			cols = append(cols, &Column{
				IsPhysical: false,
				Computation: &Computation{
					Op:       "join_using",
					Operands: args,
				},
			})
		}
		// ensure join on condition usage
		if tempColumns, err = r.buildExpression(dbID, te.Condition.On, cols); err != nil {
			return
		}
		cols = append(cols, tempColumns...)
		return
	case *sqlparser.ParenTableExpr:
		var tempColumns Columns
		for _, tableExpr := range te.Exprs {
			if tempColumns, err = r.buildAllColumnInTableExpr(dbID, tableExpr, extraColumns); err != nil {
				return
			}

			cols = append(cols, tempColumns...)
		}
		return
	default:
		tb := sqlparser.NewTrackedBuffer(nil)
		err = errors.Wrapf(ErrInvalidStatement, "invalid table expression %s", tb.WriteNode(tableExpr).String())
		return
	}
}

func (r *Resolver) buildAllColumnInTableExprs(dbID string, tableExprs sqlparser.TableExprs, extraColumns Columns) (cols Columns, err error) {
	var tempColumns Columns
	for _, tableExpr := range tableExprs {
		if tempColumns, err = r.buildAllColumnInTableExpr(dbID, tableExpr, extraColumns); err != nil {
			return
		}
		cols = append(cols, tempColumns...)
	}
	return
}

func (r *Resolver) buildLimit(dbID string, limitCond *sqlparser.Limit) (cols Columns, err error) {
	if limitCond == nil {
		return
	}

	var tempColumns Columns
	if tempColumns, err = r.buildExpression(dbID, limitCond.Offset, nil); err != nil {
		return
	}
	cols = append(cols, tempColumns...)
	if tempColumns, err = r.buildExpression(dbID, limitCond.Rowcount, nil); err != nil {
		return
	}
	cols = append(cols, tempColumns...)
	return
}

func (r *Resolver) buildUpdateExprs(dbID string, updateExprs sqlparser.UpdateExprs, originColumns Columns) (cols Columns, err error) {
	for _, ue := range updateExprs {
		colTableName := ue.Name.Qualifier.Name.String()
		found := false

		for _, oc := range originColumns {
			if colTableName != "" && !strings.EqualFold(oc.ColName.Qualifier.Name.String(), colTableName) {
				// table name not matched
				continue
			}

			if ue.Name.Name.Equal(oc.ColName.Name) {
				// matched
				found = true
				cols = append(cols, oc)
			}
		}

		if !found {
			tb := sqlparser.NewTrackedBuffer(nil)
			err = errors.Wrapf(ErrInvalidColumn, "no such column %s", tb.WriteNode(ue).String())
			return
		}
	}

	return
}

func (r *Resolver) buildInsertColumns(dbID string, insertCols sqlparser.Columns, originColumns Columns) (cols Columns, err error) {
	for _, ic := range insertCols {
		found := false

		for _, oc := range originColumns {
			if oc.ColName.Name.Equal(ic) {
				// matched
				cols = append(cols, oc)
				found = true
				break
			}
		}

		if !found {
			err = errors.Wrapf(ErrInvalidColumn, "no such column %s", ic.String())
			return
		}
	}

	return
}

func (r *Resolver) buildInsertRows(dbID string, insertRows sqlparser.InsertRows) (cols Columns, err error) {
	switch n := insertRows.(type) {
	case sqlparser.SelectStatement:
		return r.buildSelectStatement(dbID, n, nil)
	case sqlparser.Values:
		// parse values
		var tempCols Columns
		for _, v := range n {
			if tempCols, err = r.buildExpression(dbID, sqlparser.Expr(v), nil); err != nil {
				return
			}
			cols = append(cols, tempCols...)
		}
		return
	default:
		tb := sqlparser.NewTrackedBuffer(nil)
		err = errors.Wrapf(ErrInvalidStatement, "invalid insert values type %s", tb.WriteNode(insertRows).String())
		return
	}
}

func (r *Resolver) buildSelectExpr(dbID string, selectExpr sqlparser.SelectExpr, hasDistinct bool, originColumns Columns) (cols Columns, err error) {
	var tempCols Columns
	switch e := selectExpr.(type) {
	case *sqlparser.AliasedExpr:
		if tempCols, err = r.buildExpression(dbID, e.Expr, originColumns); err != nil {
			return
		}

		// must be one column result
		if len(tempCols) != 1 {
			err = errors.Wrapf(ErrInvalidStatement, "sub query returns %d columns", len(tempCols))
			return
		}

		if e.As.IsEmpty() {
			// no alias, keep as the original column result
			cols = append(cols, tempCols[0])
			return
		}

		// add alias process
		cols = append(cols, &Column{
			ColName: sqlparser.ColName{
				Name: sqlparser.NewColIdent(e.As.String()),
			},
			IsPhysical: false,
			Computation: &Computation{
				Op:       "alias",
				Operands: Columns{tempCols[0]},
			},
		})
	case *sqlparser.StarExpr:
		// filter origin columns
		if e.TableName.IsEmpty() {
			cols = append(cols, originColumns...)
		} else {
			// filter by table name
			starColumns := make(Columns, 0, len(originColumns))

			for _, c := range originColumns {
				if strings.EqualFold(e.TableName.Name.String(), c.ColName.Qualifier.Name.String()) {
					starColumns = append(starColumns, c)
				}
			}

			if len(starColumns) == 0 {
				err = errors.Wrapf(ErrInvalidTable, "no such table %s", e.TableName.Name.String())
				return
			}

			cols = append(cols, starColumns...)
		}
	}

	cols = func() Columns {
		newCols := NewColumns()
		for _, c := range cols {
			newCols = append(newCols, &Column{
				ColName:    c.ColName,
				IsPhysical: false,
				Computation: &Computation{
					Op:       "distinct",
					Operands: cols,
				},
			})
		}
		return newCols
	}()

	return
}

func (r *Resolver) buildSelectExprs(dbID string, selectExprs sqlparser.SelectExprs, hasDistinct bool, originColumns Columns) (cols Columns, err error) {
	var tempCols Columns
	for _, expr := range selectExprs {
		if tempCols, err = r.buildSelectExpr(dbID, expr, hasDistinct, originColumns); err != nil {
			return
		}
		cols = append(cols, tempCols...)
	}

	return
}

func (r *Resolver) filterDependentCols(originColumns Columns) (cols Columns) {
	cols = make(Columns, 0, len(originColumns))

	for _, c := range originColumns {
		if !c.IsPhysical {
			// do not add dependency that only contains aliases
			if c.Computation != nil && c.Computation.Op == "alias" {
				tempCols := r.filterDependentCols(c.Computation.Operands)
				if len(tempCols) == 0 {
					continue
				}
			}
			cols = append(cols, c)
		}
	}

	return
}

func (r *Resolver) buildSelectStatement(dbID string, stmt sqlparser.SelectStatement, extraColumns Columns) (cols Columns, err error) {
	var (
		tempCols   Columns
		originCols Columns
	)

	switch n := stmt.(type) {
	case *sqlparser.Select:
		var (
			resultCols   Columns
			symTableCols = NewColumns(extraColumns...)
		)
		// in select origin cols may not be physical cols, physical cols should be filtered
		if originCols, err = r.buildAllColumnInTableExprs(dbID, n.From, extraColumns); err != nil {
			return
		}
		// append all virtual fields to dependent cols
		tempCols = r.filterDependentCols(originCols)
		cols = append(cols, tempCols...)
		// use all physical table columns as symbol table columns
		symTableCols = append(symTableCols, originCols...)
		// add extra columns from outer symbol table
		for _, c := range extraColumns {
			if _, err := symTableCols.ResolveColName(&c.ColName); err != nil {
				symTableCols = append(symTableCols, c)
			}
		}
		// select expression is the projection of origin cols
		if resultCols, err = r.buildSelectExprs(dbID, n.SelectExprs, n.Distinct != "", originCols); err != nil {
			return
		}
		cols = append(cols, resultCols...)
		// build sym table cols
		for _, c := range resultCols {
			if _, err := symTableCols.ResolveColIdent(c.ColName.Name); err != nil {
				// not exists
				symTableCols = append(symTableCols, c)
			}
		}
		if tempCols, err = r.buildWhere(dbID, n.Where, symTableCols); err != nil {
			return
		}
		cols = append(cols, tempCols...)
		if tempCols, err = r.buildGroupBy(dbID, n.GroupBy, symTableCols); err != nil {
			return
		}
		cols = append(cols, tempCols...)
		if tempCols, err = r.buildWhere(dbID, n.Having, symTableCols); err != nil {
			return
		}
		cols = append(cols, tempCols...)
		if tempCols, err = r.buildOrderBy(dbID, n.OrderBy, symTableCols); err != nil {
			return
		}
		cols = append(cols, tempCols...)
		if tempCols, err = r.buildLimit(dbID, n.Limit); err != nil {
			return
		}
		cols = append(cols, tempCols...)
		// iterate result cols
		cols = func() Columns {
			newCols := NewColumns()
			for _, c := range resultCols {
				newCols = append(newCols, &Column{
					ColName:    c.ColName,
					IsPhysical: false,
					Computation: &Computation{
						Op:       "filter",
						Operands: cols,
					},
				})
			}
			return newCols
		}()
	case *sqlparser.Union:
		if originCols, err = r.buildSelectStatement(dbID, n.Left, nil); err != nil {
			return
		}
		cols = append(cols, originCols...)
		if tempCols, err = r.buildSelectStatement(dbID, n.Right, nil); err != nil {
			return
		}
		cols = append(cols, tempCols...)
		if tempCols, err = r.buildOrderBy(dbID, n.OrderBy, cols); err != nil {
			return
		}
		cols = append(cols, tempCols...)
		if tempCols, err = r.buildLimit(dbID, n.Limit); err != nil {
			return
		}
		cols = append(cols, tempCols...)
		cols = func() Columns {
			newCols := NewColumns()
			for _, c := range originCols {
				newCols = append(newCols, &Column{
					ColName:    c.ColName,
					IsPhysical: false,
					Computation: &Computation{
						Op:       "union",
						Operands: cols,
					},
				})
			}
			return newCols
		}()
	case *sqlparser.ParenSelect:
		// not supported in sqlite
		err = errors.Wrap(ErrInvalidStatement, "invalid paren select statement")
	}
	return
}

func (r *Resolver) GetDependentPhysicalColumns(dbID string, stmt sqlparser.Statement) (columns []*ColumnResult, err error) {
	var (
		cols      Columns
		columnMap = make(map[string]map[string]int)
		visit     func(opStack []string, cols Columns)
	)
	columns = make([]*ColumnResult, 0)
	if cols, err = r.BuildPhysicalColumnsTransformations(dbID, stmt); err != nil {
		return
	}
	visit = func(opStack []string, cols Columns) {
		for _, c := range cols {
			if !c.IsPhysical {
				if c.Computation != nil {
					subStack := make([]string, len(opStack)+1)
					subStack[0] = c.Computation.Op
					copy(subStack[1:], opStack)
					visit(subStack, c.Computation.Operands)
				}
			} else {
				tableName := strings.ToLower(c.ColName.Qualifier.Name.String())
				colName := c.ColName.Name.Lowered()

				if o, exists := columnMap[tableName][colName]; !exists {
					if _, exists := columnMap[tableName]; !exists {
						columnMap[tableName] = make(map[string]int)
					}

					columnMap[tableName][colName] = len(columns)
					columns = append(columns, &ColumnResult{
						TableName: tableName,
						ColName:   colName,
						Ops:       [][]string{opStack},
					})
				} else {
					// exists
					columns[o].Ops = append(columns[o].Ops, opStack)
				}
			}
		}
	}
	visit(nil, cols)
	return
}

func (r *Resolver) BuildPhysicalColumnsTransformations(dbID string, stmt sqlparser.Statement) (cols Columns, err error) {
	var (
		tempCols   Columns
		originCols Columns
	)

	switch n := stmt.(type) {
	case sqlparser.SelectStatement:
		return r.buildSelectStatement(dbID, n, nil)
	case *sqlparser.Insert:
		// in insert, origin cols ia always physical cols
		if originCols, err = r.buildAllColumnInTable(dbID, n.Table); err != nil {
			return
		}
		// insert columns can be projected
		if tempCols, err = r.buildInsertColumns(dbID, n.Columns, originCols); err != nil {
			return
		}
		cols = append(cols, tempCols...)
		// insert values has no relation to the table
		if tempCols, err = r.buildInsertRows(dbID, n.Rows); err != nil {
			return
		}
		cols = append(cols, tempCols...)
		// update expression can be projected, on duplication expression is the same as update expression
		if tempCols, err = r.buildUpdateExprs(dbID, sqlparser.UpdateExprs(n.OnDup), originCols); err != nil {
			return
		}
		cols = append(cols, tempCols...)
	case *sqlparser.Update:
		// in update, origin cols is always physical cols
		if originCols, err = r.buildAllColumnInTableExprs(dbID, n.TableExprs, nil); err != nil {
			return
		}
		// update expression can be projected
		if tempCols, err = r.buildUpdateExprs(dbID, n.Exprs, originCols); err != nil {
			return
		}
		cols = append(cols, tempCols...)
		if tempCols, err = r.buildWhere(dbID, n.Where, originCols); err != nil {
			return
		}
		cols = append(cols, tempCols...)
		if tempCols, err = r.buildOrderBy(dbID, n.OrderBy, originCols); err != nil {
			return
		}
		cols = append(cols, tempCols...)
		if tempCols, err = r.buildLimit(dbID, n.Limit); err != nil {
			return
		}
		cols = append(cols, tempCols...)
	case *sqlparser.Delete:
		if originCols, err = r.buildAllColumnInTableExprs(dbID, n.TableExprs, nil); err != nil {
			return
		}
		cols = append(cols, originCols...)
		if tempCols, err = r.buildWhere(dbID, n.Where, originCols); err != nil {
			return
		}
		cols = append(cols, tempCols...)
		if tempCols, err = r.buildOrderBy(dbID, n.OrderBy, originCols); err != nil {
			return
		}
		cols = append(cols, tempCols...)
		if tempCols, err = r.buildLimit(dbID, n.Limit); err != nil {
			return
		}
		cols = append(cols, tempCols...)
	case *sqlparser.DDL:
		// requires table privilege, if create table statement is passed, requires extra privilege
	case *sqlparser.Show:
		// no physical column rely, but requires extra privilege
	case *sqlparser.Explain:
		// no physical column rely, but requires extra privilege
	default:
		// invalid statement
		err = errors.Wrap(ErrInvalidStatement, "unknown statement")
	}

	return
}
