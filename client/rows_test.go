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
	"testing"

	"github.com/CovenantSQL/CovenantSQL/types"
	. "github.com/smartystreets/goconvey/convey"
)

func TestRowsStructure(t *testing.T) {
	Convey("test rows", t, func() {
		r := newRows(&types.Response{
			Payload: types.ResponsePayload{
				Columns: []string{
					"a",
				},
				DeclTypes: []string{
					"int",
				},
				Rows: []types.ResponseRow{
					{
						Values: []interface{}{1},
					},
				},
			},
		})
		columns := r.Columns()
		So(columns, ShouldResemble, []string{"a"})
		So(r.ColumnTypeDatabaseTypeName(0), ShouldEqual, "INT")

		dest := make([]driver.Value, 1)
		err := r.Next(dest)
		So(err, ShouldBeNil)
		So(dest[0], ShouldEqual, 1)
		err = r.Next(dest)
		So(err, ShouldEqual, io.EOF)
		err = r.Close()
		So(err, ShouldBeNil)
		So(r.data, ShouldBeNil)
	})
}
