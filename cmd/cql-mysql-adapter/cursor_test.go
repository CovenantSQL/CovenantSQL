/*
 * Copyright 2026 The CovenantSQL Authors.
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
	"testing"

	_ "github.com/CovenantSQL/go-sqlite3-encrypt"
	my "github.com/go-mysql-org/go-mysql/mysql"
	. "github.com/smartystreets/goconvey/convey"
)

func TestResultSetNull(t *testing.T) {
	// In a text result row, 0x00 is a zero-length string and 0xfb is NULL.
	Convey("an empty string is sent as an empty string, not NULL", t, func() {
		db, err := sql.Open("sqlite3", ":memory:")
		So(err, ShouldBeNil)
		defer db.Close()

		rows, err := db.Query("SELECT '', NULL, 'x'")
		So(err, ShouldBeNil)
		defer rows.Close()

		r, err := NewCursor(&Server{}).buildResultSet(rows)
		So(err, ShouldBeNil)
		So(r.Resultset.RowDatas, ShouldResemble, []my.RowData{{0x00, 0xfb, 0x01, 'x'}})
	})
	Convey("SELECT DATABASE() is NULL until a database is selected", t, func() {
		c := NewCursor(&Server{})
		r, processed, err := c.handleSpecialQuery("SELECT DATABASE()")
		So(err, ShouldBeNil)
		So(processed, ShouldBeTrue)
		So(r.Resultset.RowDatas, ShouldResemble, []my.RowData{{0xfb}})

		c.curDB = "db"
		r, _, err = c.handleSpecialQuery("SELECT DATABASE()")
		So(err, ShouldBeNil)
		So(r.Resultset.RowDatas, ShouldResemble, []my.RowData{{0x02, 'd', 'b'}})
	})
}
