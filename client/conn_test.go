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
	"database/sql"
	"sync"
	"testing"

	"github.com/CovenantSQL/CovenantSQL/utils/log"
	. "github.com/smartystreets/goconvey/convey"
)

func TestConn(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	Convey("test connection", t, func() {
		var stopTestService func()
		var err error
		stopTestService, _, err = startTestService()
		So(err, ShouldBeNil)
		defer stopTestService()

		var db *sql.DB
		db, err = sql.Open("covenantsql", "covenantsql://db?update_interval=400ms")
		So(db, ShouldNotBeNil)
		So(err, ShouldBeNil)

		_, err = db.Exec("create table test (test int)")
		So(err, ShouldBeNil)
		_, err = db.Exec("insert into test values (1)")
		So(err, ShouldBeNil)

		// test with query
		var rows *sql.Rows
		var result int
		rows, err = db.Query("select * from test")
		So(err, ShouldBeNil)
		So(rows, ShouldNotBeNil)
		So(rows.Next(), ShouldBeTrue)
		err = rows.Scan(&result)
		So(err, ShouldBeNil)
		So(result, ShouldEqual, 1)
		So(rows.Next(), ShouldBeFalse)
		rows.Close()

		testRowCount := func(expected int) {
			var row *sql.Row
			var err error
			var result int
			row = db.QueryRow("select count(1) as cnt from test")
			So(row, ShouldNotBeNil)
			err = row.Scan(&result)
			So(err, ShouldBeNil)
			So(result, ShouldEqual, expected)
		}

		// test with query and arguments
		_, err = db.Exec("insert into test values(?)", 2)
		So(err, ShouldBeNil)
		testRowCount(2)

		// test with query and multiple arguments
		_, err = db.Exec("insert into test values(?), (?)", 3, 4)
		So(err, ShouldBeNil)
		testRowCount(4)

		// parameter count is more than placeholders
		_, err = db.Exec("insert into test values(?)", 5, 6)
		So(err, ShouldBeNil)
		testRowCount(5)

		// parameter count fewer than placeholders
		_, err = db.Exec("insert into test values(?)")
		So(err, ShouldNotBeNil)

		// select with parameter
		var row *sql.Row
		row = db.QueryRow("select * from test where test > ? limit 1", 1)
		So(row, ShouldNotBeNil)
		err = row.Scan(&result)
		So(err, ShouldBeNil)
		So(result, ShouldEqual, 2)

		// test with named parameters
		row = db.QueryRow("select * from test where test < :higher_bound and test > :lower_bound limit 1",
			sql.Named("lower_bound", 2),
			sql.Named("higher_bound", 4),
		)
		So(row, ShouldNotBeNil)
		err = row.Scan(&result)
		So(err, ShouldBeNil)
		So(result, ShouldResemble, 3)

		// multiple rows
		rows, err = db.Query("select * from test where test < 3")
		So(err, ShouldBeNil)
		So(rows, ShouldNotBeNil)
		var columns []string
		columns, err = rows.Columns()
		So(err, ShouldBeNil)
		So(columns, ShouldResemble, []string{"test"})
		var types []*sql.ColumnType
		types, err = rows.ColumnTypes()
		So(err, ShouldBeNil)
		So(len(types), ShouldEqual, 1)
		So(types[0].Name(), ShouldEqual, "test")
		So(types[0].DatabaseTypeName(), ShouldEqual, "INT") // driver will do auto type uppercase

		for i := 1; i != 3; i++ {
			So(rows.Next(), ShouldBeTrue)
			err = rows.Scan(&result)
			So(err, ShouldBeNil)
			So(result, ShouldEqual, i)
		}

		So(rows.Next(), ShouldBeFalse)
		rows.Close()

		// close the rows during read
		rows, err = db.Query("select * from test where test < 3")
		So(err, ShouldBeNil)
		So(rows, ShouldNotBeNil)
		So(rows.Next(), ShouldBeTrue)
		rows.Close()
		So(rows.Next(), ShouldBeFalse)

		// use of closed connection
		db.Close()

		_, err = db.Exec("insert into test values(5)")
		So(err, ShouldNotBeNil)
	})
}

func TestTransaction(t *testing.T) {
	Convey("test transaction", t, func() {
		var stopTestService func()
		var err error
		stopTestService, _, err = startTestService()
		So(err, ShouldBeNil)
		defer stopTestService()

		var db *sql.DB
		db, err = sql.Open("covenantsql", "covenantsql://db")
		So(db, ShouldNotBeNil)
		So(err, ShouldBeNil)

		var execResult sql.Result
		var lastInsertID, affectedRows int64

		_, err = db.Exec("create table test (test int)")
		So(err, ShouldBeNil)
		execResult, err = db.Exec("insert into test values (1)")
		So(err, ShouldBeNil)
		lastInsertID, err = execResult.LastInsertId()
		So(err, ShouldBeNil)
		So(lastInsertID, ShouldEqual, 1)
		affectedRows, err = execResult.RowsAffected()
		So(affectedRows, ShouldEqual, 1)

		// test start transaction
		var tx *sql.Tx
		tx, err = db.Begin()
		So(tx, ShouldNotBeNil)
		So(err, ShouldBeNil)

		// test fill invalid read query in transaction
		_, err = tx.Query("select * from test")
		So(err, ShouldNotBeNil)

		// test query
		_, err = tx.Exec("insert into test values(2)")
		So(err, ShouldBeNil)

		// test rollback
		err = tx.Rollback()
		So(err, ShouldBeNil)

		testRowCount := func(expected int) {
			var row *sql.Row
			var err error
			var result int
			row = db.QueryRow("select count(1) as cnt from test")
			So(row, ShouldNotBeNil)
			err = row.Scan(&result)
			So(err, ShouldBeNil)
			So(result, ShouldEqual, expected)
		}

		// test row count on rollback
		testRowCount(1)

		// test commit this time
		err = tx.Commit()
		So(err, ShouldNotBeNil) // already rollback

		// test commits
		tx, err = db.Begin()
		So(tx, ShouldNotBeNil)
		So(err, ShouldBeNil)

		_, err = tx.Exec("insert into test values(2)")
		So(err, ShouldBeNil)
		_, err = tx.Exec("insert into test values(3)")
		So(err, ShouldBeNil)

		err = tx.Commit()
		So(err, ShouldBeNil)
		testRowCount(3)
		err = tx.Rollback()
		So(err, ShouldNotBeNil)

		// test failures during transaction batch
		tx, err = db.Begin()
		So(tx, ShouldNotBeNil)
		So(err, ShouldBeNil)

		_, err = tx.Exec("insert into test values(4)")
		So(err, ShouldBeNil)
		_, err = tx.Exec("THIS IS NOT A SQL!!!!")
		So(err, ShouldBeNil) // should be nil since query is not committed
		err = tx.Commit()
		So(err, ShouldNotBeNil) // should failed
		testRowCount(3)         // should still be 3 rows

		// test rollback empty transaction
		tx, err = db.Begin()
		So(tx, ShouldNotBeNil)
		So(err, ShouldBeNil)
		err = tx.Rollback()
		So(err, ShouldNotBeNil)

		// test commit empty transaction, should silently success
		tx, err = db.Begin()
		So(tx, ShouldNotBeNil)
		So(err, ShouldBeNil)
		err = tx.Commit()
		So(err, ShouldBeNil)

		db.Close()

		// test starting transaction after connection closed
		tx, err = db.Begin()
		So(tx, ShouldBeNil)
		So(err, ShouldNotBeNil)
	})
}

func TestConnAndSeqAllocation(t *testing.T) {
	Convey("conn id and seq no allocation test", t, func() {
		var wg sync.WaitGroup
		var seqMap sync.Map
		testFunc := func(c, s uint64) {
			v, l := seqMap.LoadOrStore(c, s)
			if l {
				vi := v.(uint64)
				if vi >= s {
					// invalid
					t.Fatalf(ErrInvalidRequestSeq.Error())
				}
			} else {
				seqMap.Store(c, s)
			}
		}
		f := func() {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := 0; i != 100; i++ {
					connID, seqNo := allocateConnAndSeq()
					t.Logf("connID: %v, seqNo: %v", connID, seqNo)
					testFunc(connID, seqNo)
					putBackConn(connID)
				}
			}()
		}

		f()
		f()
		f()
		f()
		wg.Wait()
	})
}
