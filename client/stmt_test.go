/*
 * Copyright 2018 The ThunderDB Authors.
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
	"database/sql"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestStmt(t *testing.T) {
	Convey("test statement", t, func() {
		var stopTestService func()
		var err error
		stopTestService, err = startTestService()
		So(err, ShouldBeNil)
		defer stopTestService()

		var db *sql.DB
		db, err = sql.Open("thunderdb", "thunderdb://db")
		So(db, ShouldNotBeNil)
		So(err, ShouldBeNil)

		_, err = db.Exec("create table test (test int)")
		So(err, ShouldBeNil)
		_, err = db.Exec("insert into test values (1)")
		So(err, ShouldBeNil)

		var stmt *sql.Stmt
		stmt, err = db.Prepare("select count(1) as cnt from test where test = ?")
		So(err, ShouldBeNil)
		So(stmt, ShouldNotBeNil)

		var row *sql.Row
		var result int

		// query with argument 1
		row = stmt.QueryRow(1)
		So(row, ShouldNotBeNil)
		err = row.Scan(&result)
		So(err, ShouldBeNil)
		So(result, ShouldEqual, 1)

		// query with argument 2
		row = stmt.QueryRow(2)
		So(row, ShouldNotBeNil)
		err = row.Scan(&result)
		So(err, ShouldBeNil)
		So(result, ShouldEqual, 0)

		// query when statement closed
		stmt.Close()
		row = stmt.QueryRow(1)
		So(row, ShouldNotBeNil)
		err = row.Scan(&result)
		So(err, ShouldNotBeNil)

		// exec statement
		stmt, err = db.Prepare("insert into test values(?)")

		// parameter count equals to placeholders
		_, err = stmt.Exec(2)
		So(err, ShouldBeNil)
		row = db.QueryRow("select count(1) as cnt from test")
		So(row, ShouldNotBeNil)
		err = row.Scan(&result)
		So(err, ShouldBeNil)
		So(result, ShouldEqual, 2) // test insert success

		// parameter count is more than placeholders
		_, err = stmt.Exec(3, 4, 5, 6)
		So(err, ShouldBeNil)
		row = db.QueryRow("select count(1) as cnt from test")
		So(row, ShouldNotBeNil)
		err = row.Scan(&result)
		So(err, ShouldBeNil)
		So(result, ShouldEqual, 3)

		// not enough placeholders
		_, err = stmt.Exec()
		So(err, ShouldNotBeNil)

		db.Close()

		// prepare on closed
		_, err = db.Prepare("select * from test")
		So(err, ShouldNotBeNil)
	})
}
