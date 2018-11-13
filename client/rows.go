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

package client

import (
	"database/sql/driver"
	"io"
	"strings"

	"github.com/CovenantSQL/CovenantSQL/types"
)

type rows struct {
	columns []string
	types   []string
	data    []types.ResponseRow
}

func newRows(res *types.Response) *rows {
	return &rows{
		columns: res.Payload.Columns,
		types:   res.Payload.DeclTypes,
		data:    res.Payload.Rows,
	}
}

// Columns implements driver.Rows.Columns method.
func (r *rows) Columns() []string {
	return r.columns[:]
}

// Close implements driver.Rows.Close method.
func (r *rows) Close() error {
	r.data = nil
	return nil
}

// Next implements driver.Rows.Next method.
func (r *rows) Next(dest []driver.Value) error {
	if len(r.data) == 0 {
		return io.EOF
	}

	for i, d := range r.data[0].Values {
		dest[i] = d
	}

	// unshift data
	r.data = r.data[1:]

	return nil
}

// ColumnTypeDatabaseTypeName implements driver.RowsColumnTypeDatabaseTypeName.ColumnTypeDatabaseTypeName method.
func (r *rows) ColumnTypeDatabaseTypeName(index int) string {
	return strings.ToUpper(r.types[index])
}
