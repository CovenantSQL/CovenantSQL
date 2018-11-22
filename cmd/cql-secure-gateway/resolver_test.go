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
	"github.com/CovenantSQL/CovenantSQL/cmd/cql-mysql-adapter/resolver"
	"github.com/CovenantSQL/sqlparser"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func newColName(table string, col string) sqlparser.ColName {
	return sqlparser.ColName{
		Qualifier: sqlparser.TableName{
			Name: sqlparser.NewTableIdent(table),
		},
		Name: sqlparser.NewColIdent(col),
	}
}

func mustParseStatement(c C, query string) sqlparser.Statement {
	stmt, err := sqlparser.Parse(query)
	c.So(err, ShouldBeNil)
	return stmt
}

func TestResolverColumns(t *testing.T) {
	Convey("test resolve column names", t, func() {
		cols := NewColumns(
			&Column{
				ColName:    newColName("test", "test"),
				IsPhysical: true,
			},
			&Column{
				ColName:    newColName("test2", "test"),
				IsPhysical: true,
			},
			&Column{
				ColName:    newColName("test", "test3"),
				IsPhysical: true,
			},
		)

		col1 := newColName("test", "test")
		col, err := cols.ResolveColName(&col1)
		So(err, ShouldBeNil)
		So(col, ShouldNotBeNil)
		So(col1.Qualifier.Name.String(), ShouldEqual, "test")
		So(col1.Name.String(), ShouldEqual, "test")

		col2 := newColName("test4", "test")
		_, err = cols.ResolveColName(&col2)
		So(err, ShouldNotBeNil)

		col3 := newColName("test", "test2")
		_, err = cols.ResolveColName(&col3)
		So(err, ShouldNotBeNil)

		col4 := newColName("", "test")
		_, err = cols.ResolveColName(&col4)
		So(err, ShouldNotBeNil)

		col, err = cols.ResolveColName(nil)
		So(col, ShouldBeNil)
		So(err, ShouldBeNil)

		col, err = cols.ResolveColIdent(sqlparser.NewColIdent("test3"))
		So(err, ShouldBeNil)
		So(col, ShouldNotBeNil)

		_, err = cols.ResolveColIdent(sqlparser.NewColIdent("test2"))
		So(err, ShouldNotBeNil)
	})
}

func TestColumnResolver(t *testing.T) {
	Convey("resolve columns", t, func(c C) {
		var (
			r          = &Resolver{Resolver: resolver.NewResolver()}
			dbID       = randomDBID()
			cols       Columns
			err        error
			db         *wrapFakeDB
			colResults []*ColumnResult
		)

		// inject normal database
		db, err = fakeConn([]string{
			"CREATE TABLE t1(a INTEGER, b INTEGER, c INTEGER, d INTEGER, e INTEGER)",
			"CREATE TABLE tab0(col0 INTEGER, col1 INTEGER, col2 INTEGER)",
			"CREATE TABLE tab1(col0 INTEGER, col1 INTEGER, col2 INTEGER)",
			"CREATE TABLE tab2(col0 INTEGER, col1 INTEGER, col2 INTEGER)",
			"CREATE TABLE tab3(pk INTEGER PRIMARY KEY, col0 INTEGER, col1 FLOAT, col2 TEXT, col3 INTEGER, col4 FLOAT, col5 TEXT)",
		})
		So(err, ShouldBeNil)
		r.RegisterDB(dbID, db)
		defer db.Close()

		cols, err = r.BuildPhysicalColumnsTransformations(dbID, mustParseStatement(c, "SELECT ? AS B"))
		So(err, ShouldBeNil)
		So(cols, ShouldHaveLength, 1)

		cols, err = r.BuildPhysicalColumnsTransformations(dbID, mustParseStatement(c, "SELECT ? AS A, ? AS B"))
		So(err, ShouldBeNil)
		So(cols, ShouldHaveLength, 2)

		cols, err = r.BuildPhysicalColumnsTransformations(dbID, mustParseStatement(c, "SELECT test FROM (SELECT 1 AS test)"))
		So(err, ShouldBeNil)
		So(cols, ShouldHaveLength, 1)

		cols, err = r.BuildPhysicalColumnsTransformations(dbID, mustParseStatement(c, "SELECT * FROM (SELECT 1 AS test)"))
		So(err, ShouldBeNil)
		So(cols, ShouldHaveLength, 1)

		cols, err = r.BuildPhysicalColumnsTransformations(dbID, mustParseStatement(c, "SELECT t.* FROM (SELECT 1 AS test) AS t"))
		So(err, ShouldBeNil)
		So(cols, ShouldHaveLength, 1)

		cols, err = r.BuildPhysicalColumnsTransformations(dbID, mustParseStatement(c, "SELECT test FROM (SELECT 1 AS test) AS t"))
		So(err, ShouldBeNil)
		So(cols, ShouldHaveLength, 1)

		cols, err = r.BuildPhysicalColumnsTransformations(dbID, mustParseStatement(c, "SELECT * FROM (SELECT 1 AS test1), (SELECT 1 AS test2)"))
		So(err, ShouldBeNil)
		So(cols, ShouldHaveLength, 2)

		cols, err = r.BuildPhysicalColumnsTransformations(dbID, mustParseStatement(c, "SELECT t1.* FROM (SELECT 1 AS test1) AS t1 JOIN (SELECT 1 AS test2) AS t2"))
		So(err, ShouldBeNil)
		So(cols, ShouldHaveLength, 1)

		cols, err = r.BuildPhysicalColumnsTransformations(dbID, mustParseStatement(c, "SELECT * FROM (SELECT 1 AS test1) AS t1 JOIN (SELECT 1 AS test2) AS t2"))
		So(err, ShouldBeNil)
		So(cols, ShouldHaveLength, 2)

		cols, err = r.BuildPhysicalColumnsTransformations(dbID, mustParseStatement(c, "SELECT 1 AS test UNION SELECT 2 AS test"))
		So(err, ShouldBeNil)
		So(cols, ShouldHaveLength, 1)

		cols, err = r.BuildPhysicalColumnsTransformations(dbID, mustParseStatement(c, `
		SELECT (a+b+c+d+e)/5,
		     a+b*2+c*3+d*4+e*5,
		     d-e,
		     a+b*2,
		     CASE a+1 WHEN b THEN 111 WHEN c THEN 222
		      WHEN d THEN 333  WHEN e THEN 444 ELSE 555 END,
		     c
		FROM t1
		WHERE c BETWEEN b-2 AND d+2
		 AND b>c
		 AND a>b`))
		So(err, ShouldBeNil)
		So(cols, ShouldHaveLength, 6)

		cols, err = r.BuildPhysicalColumnsTransformations(dbID, mustParseStatement(c, `
		SELECT CASE a+1 WHEN b THEN 111 WHEN c THEN 222
		       WHEN d THEN 333  WHEN e THEN 444 ELSE 555 END
		 FROM t1
		WHERE EXISTS(SELECT 1 FROM t1 AS x WHERE x.b<t1.b)`))
		So(err, ShouldBeNil)
		So(cols, ShouldHaveLength, 1)

		cols, err = r.BuildPhysicalColumnsTransformations(dbID, mustParseStatement(c, `
		SELECT c-d,
		      abs(b-c)
		 FROM t1
		WHERE (c<=d-2 OR c>=d+2)
		  AND e+d BETWEEN a+b-10 AND c+130
		  AND b IS NOT NULL`))
		So(err, ShouldBeNil)
		So(cols, ShouldHaveLength, 2)

		colResults, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT c-d,
       abs(b-c)
  FROM t1
 WHERE (c<=d-2 OR c>=d+2)
   AND e+d BETWEEN a+b-10 AND c+130
   AND b IS NOT NULL`))
		So(err, ShouldBeNil)
		So(colResults, ShouldHaveLength, 5)

		colResults, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT a-b,
       (SELECT count(1) FROM t1 AS x WHERE x.b<t1.b),
       a
  FROM t1
`))
		So(err, ShouldBeNil)
		So(colResults, ShouldHaveLength, 2)

		colResults, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT c
  FROM t1
 WHERE (e>a AND e<b)
`))
		So(err, ShouldBeNil)
		So(colResults, ShouldHaveLength, 4)

		colResults, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT a FROM t1 LIMIT (SELECT count(b) FROM t1) 
`))
		So(err, ShouldBeNil)
		So(colResults, ShouldHaveLength, 2)

		colResults, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT DISTINCT col0 * - cor0.col0 + cor0.col0 AS col0 FROM tab0 AS cor0 GROUP BY cor0.col0`))
		So(err, ShouldBeNil)
		So(colResults, ShouldHaveLength, 1)

		colResults, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT pk FROM tab3 WHERE ((col4 > 901.6) AND col0 >= 17 AND (col0 IS NULL) AND col0 < 649) AND (col3 > 960) OR (col3 > 58) ORDER BY 1 DESC`))
		So(err, ShouldBeNil)
		So(colResults, ShouldHaveLength, 4)

		colResults, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT substr(pk, 1, 1), group_concat(pk), cast(pk as string) FROM tab3 
`))
		So(err, ShouldBeNil)
		So(colResults, ShouldHaveLength, 1)

		colResults, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT true, null, current_timestamp, not false
`))
		So(err, ShouldBeNil)
		So(colResults, ShouldHaveLength, 0)

		colResults, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT col0 FROM tab0, tab1
`))
		So(err, ShouldNotBeNil)

		colResults, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT col1024 FROM tab0
`))
		So(err, ShouldNotBeNil)
	})
}
