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
	"testing"

	"github.com/CovenantSQL/sqlparser"
	"github.com/pkg/errors"
	. "github.com/smartystreets/goconvey/convey"
)

func runTest(c C, r *Resolver, dbID string, query string, paramCount int, resultColumns []string) {
	var (
		st  sqlparser.Statement
		q   *Query
		err error
	)
	st, err = sqlparser.Parse(query)
	c.So(err, ShouldBeNil)
	q, err = r.Resolve(dbID, st)
	c.So(err, ShouldBeNil)
	c.So(q.ParamCount, ShouldEqual, paramCount)
	c.So(q.ResultColumns, ShouldResemble, resultColumns)
}

func runErrorTest(c C, r *Resolver, dbID string, query string, expectedErr error) {
	var (
		st  sqlparser.Statement
		err error
	)
	st, err = sqlparser.Parse(query)
	c.So(err, ShouldBeNil)
	_, err = r.Resolve(dbID, st)
	c.So(errors.Cause(err), ShouldEqual, expectedErr)
}

func TestResolver(t *testing.T) {
	Convey("resolver", t, func(c C) {
		var (
			r        *Resolver
			testDBID = randomDBID()
			db       *wrapFakeDB
			err      error
		)
		r = NewResolver()
		defer r.Close()

		// inject real tables
		db, err = fakeConn([]string{
			"CREATE TABLE test1 (test2 string)",
			"CREATE TABLE test3 (test4 string, test5 string, test6 string)",
		})
		So(err, ShouldBeNil)
		r.RegisterDB(testDBID, db)
		defer db.Close()

		// single parameter
		runTest(c, r, testDBID, "SELECT ? AS B", 1, []string{"B"})
		// multiple parameter
		runTest(c, r, testDBID, "SELECT ? AS A, ? AS B", 2, []string{"A", "B"})
		// single named parameter referenced multiple times
		runTest(c, r, testDBID, "SELECT :var AS A, :var AS B", 1, []string{"A", "B"})
		// select from sub query
		runTest(c, r, testDBID, "SELECT test FROM (SELECT 1 AS test)", 0, []string{"test"})
		// star select from sub query
		runTest(c, r, testDBID, "SELECT * FROM (SELECT 1 AS test)", 0, []string{"test"})
		// named star select from named sub query
		runTest(c, r, testDBID, "SELECT t.* FROM (SELECT 1 AS test) AS t", 0, []string{"test"})
		// select from named sub query
		runTest(c, r, testDBID, "SELECT test FROM (SELECT 1 AS test) AS t", 0, []string{"test"})
		// select from multiple tables
		runTest(c, r, testDBID, "SELECT * FROM (SELECT 1 AS test1), (SELECT 1 AS test2)", 0, []string{"test1", "test2"})
		// select from paren tables
		runTest(c, r, testDBID, "SELECT * FROM ((SELECT 1 AS test1), (SELECT 1 AS test2))", 0, []string{"test1", "test2"})
		// named star select from multiple tables
		runTest(c, r, testDBID, "SELECT t1.* FROM (SELECT 1 AS test1) AS t1, (SELECT 1 AS test2) AS t2", 0, []string{"test1"})
		// test join
		runTest(c, r, testDBID, "SELECT t1.* FROM (SELECT 1 AS test1) AS t1 JOIN (SELECT 1 AS test2) AS t2", 0, []string{"test1"})
		// test union
		runTest(c, r, testDBID, "SELECT 1 AS test UNION SELECT 2 AS test", 0, []string{"test"})
		// test special queries
		runTest(c, r, testDBID, "SHOW TABLES", 0, columnsShowTables)
		runTest(c, r, testDBID, "SHOW INDEX FROM TABLE test", 0, columnsShowIndex)
		runTest(c, r, testDBID, "SHOW CREATE TABLE test", 0, columnsShowCreateTable)
		runTest(c, r, testDBID, "DESC test", 0, columnsShowTableInfo)
		runTest(c, r, testDBID, "EXPLAIN SELECT 1", 0, columnsExplain)
		// test no result column query
		runTest(c, r, testDBID, "INSERT INTO test VALUES(?)", 1, nil)
		// test with real tables
		runTest(c, r, testDBID, "SELECT * FROM test1", 0, []string{"test2"})
		// test multiple real tables
		runTest(c, r, testDBID, "SELECT * FROM test1, test3", 0, []string{"test2", "test4", "test5", "test6"})
		// projection table not exists in from clause
		runErrorTest(c, r, testDBID, "SELECT test3.* FROM test1", ErrQueryLogicError)
		// select from non-existence table
		runErrorTest(c, r, testDBID, "SELECT * FROM test4", ErrTableNotExists)

		// wrapped single query resolver
		var q *Query
		q, err = r.ResolveSingleQuery(testDBID, "SELECT 1")
		So(err, ShouldBeNil)
		So(q, ShouldNotBeNil)
		So(q.ParamCount, ShouldEqual, 0)
		So(q.ResultColumns, ShouldResemble, []string{"1"})

		// wrapped multiple query resolver
		var queries []*Query
		queries, err = r.ResolveQuery(testDBID, "SELECT 1 AS a; SELECT 2 AS b;")
		So(err, ShouldBeNil)
		So(queries, ShouldHaveLength, 2)
		So(queries[0].ParamCount, ShouldEqual, 0)
		So(queries[0].ResultColumns, ShouldResemble, []string{"a"})
		So(queries[1].ParamCount, ShouldEqual, 0)
		So(queries[1].ResultColumns, ShouldResemble, []string{"b"})

		// test edge cases
		runTest(c, r, testDBID, "SELECT * FROM (SELECT 1 AS test), (SELECT 1 AS test)", 0, []string{"test", "test"})
	})
}

func TestColumnMappingUtil(t *testing.T) {
	Convey("test column mapping util", t, func() {
		cm := NewTableColumnMapping()
		cm.Add("a", []string{"b", "c"})
		cm.Add("b", []string{"c", "d"})
		cols, exists := cm.Get("a")
		So(cols, ShouldResemble, []string{"b", "c"})
		So(exists, ShouldBeTrue)
		_, exists = cm.Get("c")
		So(exists, ShouldBeFalse)

		cm.Add("a", []string{"c", "d"})
		cols, exists = cm.Get("a")
		So(cols, ShouldResemble, []string{"c", "d"})
		So(exists, ShouldBeTrue)

		cm.AppendColumn("a", []string{"f", "g"})
		cols, exists = cm.Get("a")
		So(cols, ShouldResemble, []string{"c", "d", "f", "g"})
		So(exists, ShouldBeTrue)

		cm2 := NewTableColumnMapping()
		cm2.Add("a", []string{"d", "e"})
		cm2.Add("f", []string{"d"})

		cm.Union(nil)
		_, exists = cm.Get("f")
		So(exists, ShouldBeFalse)

		cm.Union(cm2)
		cols, exists = cm.Get("a")
		So(cols, ShouldResemble, []string{"c", "d", "f", "g"})
		So(exists, ShouldBeTrue)
		cols, exists = cm.Get("f")
		So(cols, ShouldResemble, []string{"d"})
		So(exists, ShouldBeTrue)

		cm.Update(nil)
		cols, exists = cm.Get("a")
		So(cols, ShouldResemble, []string{"c", "d", "f", "g"})
		So(exists, ShouldBeTrue)

		cm.Update(cm2)
		cols, exists = cm.Get("a")
		So(cols, ShouldResemble, []string{"d", "e"})
		So(exists, ShouldBeTrue)

		cm.Merge(nil)
		cols, exists = cm.Get("a")
		So(cols, ShouldResemble, []string{"d", "e"})
		So(exists, ShouldBeTrue)

		cm.Merge(cm2)
		cols, exists = cm.Get("a")
		So(cols, ShouldResemble, []string{"d", "e", "d", "e"})
		So(exists, ShouldBeTrue)

		cm.Clear()
		_, exists = cm.Get("a")
		So(exists, ShouldBeFalse)
	})
}
