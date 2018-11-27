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
	"testing"

	"github.com/CovenantSQL/CovenantSQL/cmd/cql-mysql-adapter/resolver"
	"github.com/CovenantSQL/sqlparser"
	. "github.com/smartystreets/goconvey/convey"
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
			"CREATE TABLE tab2(col0 INTEGER, col1 INTEGER, col2 INTEGER, col3 INTEGER)",
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

		// READ STATEMENTS
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
SELECT tab1.col1 AS col0 FROM tab1 GROUP BY tab1.col1 HAVING COUNT(tab1.col2) >= ( NULL )`))
		So(err, ShouldBeNil)
		So(colResults, ShouldHaveLength, 2)

		colResults, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT pk FROM tab3 WHERE ((col4 > 901.6) AND col0 >= 17 AND (col0 IS NULL) AND col0 < 649) AND (col3 > 960) OR (col3 > 58) ORDER BY 1 DESC`))
		So(err, ShouldBeNil)
		So(colResults, ShouldHaveLength, 4)

		colResults, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT pk FROM tab3 WHERE (col0 >= 93) AND (((col1 >= 55.99 OR (col3 <= 1) OR (((col3 >= 24))) AND
col3 IN (84,26,92,96) OR ((col3 >= 53)) AND (col4 >= 6.22) AND (col0 >= 33 OR (((((col0 >= 6)) AND
((col3 < 27 AND (((col0 > 78))))) OR col1 > 90.47 OR col3 IN (47,79) AND col1 > 57.55 OR ((col1 < 77.85)) OR (col3 = 36 OR col0 < 90) AND col3 <= 44)))
OR col1 IN (30.98,34.53,44.58,62.81,85.85) OR col3 IS NULL) AND (col3 IN (37,6,58,52,90,51) OR (col3 >= 55)))) AND ((col1 = 46.0 OR col3 <= 24))
AND ((((col0 < 75)) AND col4 >= 77.95)) OR (col0 < 40) OR ((((((col3 IS NULL))) OR (col3 > 61) AND col3 > 30)) OR col1 < 11.20) AND col1 >= 17.42
AND col0 >= 96 OR col3 > 47 OR (((col3 BETWEEN 4 AND 79) AND col3 >= 51)))`))
		So(err, ShouldBeNil)
		So(colResults, ShouldHaveLength, 5)

		// join statement
		colResults, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT COUNT( * ) + 30 AS col2 FROM tab0 AS cor0 CROSS JOIN tab0`))
		So(err, ShouldBeNil)
		So(colResults, ShouldHaveLength, 0)

		// join with using
		colResults, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT COUNT(1) FROM tab0 JOIN tab1 USING(col0)`))
		So(err, ShouldBeNil)
		So(colResults, ShouldHaveLength, 2)

		// join with on condition
		colResults, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT + MIN ( cor0.col2 + 36 ) FROM tab2 AS cor0 JOIN tab0 cor1 ON NOT + - 81 >= + CAST ( NULL AS INT ) + - 6`))
		So(err, ShouldBeNil)
		So(colResults, ShouldHaveLength, 1)

		// paren from table statement
		colResults, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT * FROM (tab0, tab2)`))
		So(err, ShouldBeNil)
		So(colResults, ShouldHaveLength, 7)

		// group_concat and cast statement
		colResults, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT substr(1, 2, 3), group_concat(pk), cast(pk as string) FROM tab3 
`))
		So(err, ShouldBeNil)
		So(colResults, ShouldHaveLength, 1)

		// literals
		colResults, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT true, null, current_timestamp, not false, "1"
`))
		So(err, ShouldBeNil)
		So(colResults, ShouldHaveLength, 0)

		// ambiguous column from two tables
		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT col0 FROM tab0, tab1
`))
		So(err, ShouldNotBeNil)

		// invalid column without table reference
		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT col1024 FROM tab0
`))
		So(err, ShouldNotBeNil)

		// invalid column with table reference
		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT tab0.col1024 FROM tab0`))
		So(err, ShouldNotBeNil)

		// and left expression failure
		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT (tab0.col1024) AND (tab0.col0) FROM tab0`))
		So(err, ShouldNotBeNil)

		// and right expression failure
		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT (tab0.col0) AND (tab0.col1024) FROM tab0`))
		So(err, ShouldNotBeNil)

		// or left expression failure
		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT (tab0.col1024) OR (tab0.col0) FROM tab0`))
		So(err, ShouldNotBeNil)

		// or right expression failure
		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT (tab0.col0) OR (tab0.col1024) FROM tab0`))
		So(err, ShouldNotBeNil)

		// not expression failure
		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT NOT tab0.col1024 FROM tab0`))
		So(err, ShouldNotBeNil)

		// value tuple failure
		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT tab0.col0 IN (tab0.col1024) FROM tab0`))
		So(err, ShouldNotBeNil)

		// invalid binary left expression
		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT tab0.col1024 + tab0.col0 FROM tab0`))
		So(err, ShouldNotBeNil)

		// invalid binary right expression
		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT tab0.col0 + tab0.col1024 FROM tab0`))
		So(err, ShouldNotBeNil)

		// invalid case condition expression
		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT CASE tab0.col1024 WHEN 1 THEN 1 ELSE 0 END FROM tab0`))
		So(err, ShouldNotBeNil)

		// invalid case result expression
		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT CASE tab0.col0 WHEN 1 THEN tab0.col1024 ELSE 2 END FROM tab0`))
		So(err, ShouldNotBeNil)

		// invalid expression to cast
		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT CAST(tab0.col1024 AS string) FROM tab0`))
		So(err, ShouldNotBeNil)

		// invalid sub query of exists
		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT COUNT(1) FROM tab0 WHERE EXISTS (SELECT x.col1024 FROM tab0 AS x WHERE tab0.col0 = x.col1024)`))
		So(err, ShouldNotBeNil)

		// invalid function argument
		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT MAX(tab0.col1024) FROM tab0`))
		So(err, ShouldNotBeNil)

		// invalid group concat argument
		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT GROUP_CONCAT(tab0.col1024) FROM tab0`))
		So(err, ShouldNotBeNil)

		// interval expression is not supported
		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT INTERVAL 3 MONTH`))
		So(err, ShouldNotBeNil)

		// invalid is expression argument
		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT tab0.col1024 IS NULL FROM tab0
`))
		So(err, ShouldNotBeNil)

		// invalid expression in between
		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT tab0.col1024 BETWEEN 1 AND 2 FROM tab0`))
		So(err, ShouldNotBeNil)

		// invalid unary expression
		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT ~tab0.col1024 FROM tab0`))
		So(err, ShouldNotBeNil)

		// invalid sub query expression
		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT (SELECT tab0.col1024 FROM tab0)`))
		So(err, ShouldNotBeNil)

		// invalid sub query column count
		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT (SELECT tab0.col0, tab0.col1 FROM tab0)`))
		So(err, ShouldNotBeNil)

		// invalid order by statement
		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT * FROM tab0 ORDER BY tab0.col1024 ASC`))
		So(err, ShouldNotBeNil)

		// invalid order by using two column expression
		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT * FROM tab0 ORDER BY (1, 2)`))
		So(err, ShouldNotBeNil)

		// invalid group by column expression
		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT COUNT(1) FROM tab0 GROUP BY tab0.col1024`))
		So(err, ShouldNotBeNil)

		// unknown table
		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT * FROM tab1000`))
		So(err, ShouldNotBeNil)

		// invalid sub query in from statement
		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT * FROM (SELECT tab0.col1024 FROM tab0)`))
		So(err, ShouldNotBeNil)

		// invalid join left table statement
		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT * FROM (SELECT tab0.col1024 FROM tab0) JOIN tab1 ON col1024 = tab1.col1`))
		So(err, ShouldNotBeNil)

		// invalid join right table statement
		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT * FROM tab1 JOIN (SELECT tab0.col1024 FROM tab0) ON col1024 = tab1.col1`))
		So(err, ShouldNotBeNil)

		// invalid join using statement
		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT * FROM tab0 JOIN tab2 USING (col3)`))
		So(err, ShouldNotBeNil)

		// invalid join on statement
		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT * FROM tab0 JOIN tab2 ON tab0.col3 = tab2.col3`))
		So(err, ShouldNotBeNil)

		// invalid paren table statement
		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT * FROM (tab0, tab1024)`))
		So(err, ShouldNotBeNil)

		// invalid limit statement
		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT * FROM tab0 LIMIT (SELECT COUNT(1) FROM tab1024)`))
		So(err, ShouldNotBeNil)

		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT * FROM tab0 LIMIT (SELECT COUNT(1) FROM tab1024), 1`))
		So(err, ShouldNotBeNil)

		// invalid select column alias
		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT (1, 2) as A`))
		So(err, ShouldNotBeNil)

		// invalid star select
		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT t.* FROM tab0`))
		So(err, ShouldNotBeNil)

		// invalid having statement
		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT * FROM tab0 GROUP BY col0 HAVING count(col1024) > 0`))
		So(err, ShouldNotBeNil)

		// invalid union left statement
		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT * FROM tab1024 UNION SELECT * FROM tab0`))
		So(err, ShouldNotBeNil)

		// invalid union right statement
		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT * FROM tab0 UNION SELECT * FROM tab1024`))
		So(err, ShouldNotBeNil)

		// invalid union order by statement
		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT * FROM tab0 UNION SELECT * FROM tab1 ORDER BY col1024`))
		So(err, ShouldNotBeNil)

		// invalid union limit statement
		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
SELECT * FROM tab0 UNION SELECT * FROM tab1 LIMIT 1, (SELECT * FROM tab1024)`))
		So(err, ShouldNotBeNil)

		// WRITE STATEMENTS
		colResults, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
INSERT INTO tab0 VALUES(1, 2, 3)`))
		So(err, ShouldBeNil)
		So(colResults, ShouldHaveLength, 3)

		colResults, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
INSERT INTO tab0 (col0) VALUES(1)`))
		So(err, ShouldBeNil)
		So(colResults, ShouldHaveLength, 1)

		colResults, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
INSERT INTO tab1 (col0) VALUES((SELECT COUNT(col1) FROM tab2))`))
		So(err, ShouldBeNil)
		So(colResults, ShouldHaveLength, 2)

		colResults, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
INSERT INTO tab1 (col0) SELECT col1 FROM tab2`))
		So(err, ShouldBeNil)
		So(colResults, ShouldHaveLength, 2)

		// invalid insert table
		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
INSERT INTO tab1024 VALUES(1, 1, 1, 1, 1)`))
		So(err, ShouldNotBeNil)

		// column mismatch in columns
		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
INSERT INTO tab0 (col0, col1, col2, col0) VALUES(1, 1, 1, 1)`))
		So(err, ShouldNotBeNil)

		// non-existent column
		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
INSERT INTO tab0 (col1024) VALUES(1)`))
		So(err, ShouldNotBeNil)

		// column count not matched in rows
		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
INSERT INTO tab0 (col0) VALUES(1, 1)`))
		So(err, ShouldNotBeNil)

		// column count not matched in sub query
		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
INSERT INTO tab0 (col0) SELECT * FROM tab1`))
		So(err, ShouldNotBeNil)

		// invalid insert rows expression
		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
INSERT INTO tab0 (col0) VALUES(col1)`))
		So(err, ShouldNotBeNil)

		// invalid insert sub query expression
		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
INSERT INTO tab0 (col0) SELECT * FROM tab1024`))
		So(err, ShouldNotBeNil)

		// update expression
		colResults, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
UPDATE tab0 SET col0 = 1 WHERE col1 = 2 ORDER BY col2 DESC LIMIT 10`))
		So(err, ShouldBeNil)
		So(colResults, ShouldHaveLength, 3)

		colResults, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
UPDATE tab0 SET col0 = col0 + 1`))
		So(err, ShouldBeNil)
		So(colResults, ShouldHaveLength, 1)

		colResults, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
UPDATE tab0 SET col0 = col0 + col1`))
		So(err, ShouldBeNil)
		So(colResults, ShouldHaveLength, 2)

		// invalid table expression
		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
UPDATE tab1024 SET col0 = 1`))
		So(err, ShouldNotBeNil)

		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
UPDATE tab0 SET col1024 = 1`))
		So(err, ShouldNotBeNil)

		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
UPDATE tab0 SET col0 = (SELECT col1024 FROM tab0)`))
		So(err, ShouldNotBeNil)

		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
UPDATE tab0 SET col1 = 1 WHERE col1024 = 1`))
		So(err, ShouldNotBeNil)

		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
UPDATE tab0 SET col1 = 1 ORDER BY col1024`))
		So(err, ShouldNotBeNil)

		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
UPDATE tab0 SET col1 = 1 LIMIT (SELECT col1024 FROM tab0)`))
		So(err, ShouldNotBeNil)

		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
UPDATE tab0 SET col1 = (1, 2)`))
		So(err, ShouldNotBeNil)

		// delete expression
		colResults, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
DELETE FROM tab0 WHERE col0 = 1 ORDER BY col0 LIMIT 10`))
		So(err, ShouldBeNil)
		So(colResults, ShouldHaveLength, 3)

		colResults, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
DELETE FROM tab0 WHERE EXISTS (SELECT col1 FROM tab1)`))
		So(err, ShouldBeNil)
		So(colResults, ShouldHaveLength, 4)

		// invalid table to delete
		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
DELETE FROM tab1024`))
		So(err, ShouldNotBeNil)

		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
DELETE FROM tab0 WHERE EXISTS (SELECT col0 FROM tab1024)`))
		So(err, ShouldNotBeNil)

		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
DELETE FROM tab0 ORDER BY col1024 LIMIT 100`))
		So(err, ShouldNotBeNil)

		_, err = r.GetDependentPhysicalColumns(dbID, mustParseStatement(c, `
DELETE FROM tab0 LIMIT (SELECT col1024 FROM tab0)`))
		So(err, ShouldNotBeNil)

		cols, err = r.BuildPhysicalColumnsTransformations(dbID, mustParseStatement(c, `INSERT INTO tab0 (col0) VALUES(?)`))
		So(err, ShouldBeNil)
	})
}
