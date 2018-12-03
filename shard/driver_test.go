/*
 * Copyright 2018 The CovenantSQL Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package shard

import (
	"database/sql"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"testing"
	"time"

	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/pkg/errors"
	. "github.com/smartystreets/goconvey/convey"
)

func TestShardingDriverCorrectness(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	os.Remove("./foo_s.db")
	//defer os.Remove("./foo_s.db")
	os.Remove("./foo.db")
	//defer os.Remove("./foo.db")

	dbs, err := sql.Open(DBSchemeAlias, "./foo_s.db")
	if err != nil {
		log.Fatal(err)
	}
	defer dbs.Close()

	db, err := sql.Open("sqlite3", "./foo.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create table and sharding tables
	const tableSchemaTpl = `
	create table if not exists foo%s (id integer not null primary key, name text, time timestamp );
	create index if not exists fooindex%s on foo%s ( time );
	`
	var sqlStmt = fmt.Sprintf(tableSchemaTpl, ShardSchemaToken, ShardSchemaToken, ShardSchemaToken)
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
		return
	}

	sqlStmt = fmt.Sprintf("SHARDCONFIG foo time %d %d ",
		864000, 1536000000) + sqlStmt
	_, err = dbs.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
		return
	}

	Convey("", t, func() {
		testCases(t, dbs, db)
	})

}

func testCases(t *testing.T, dbs *sql.DB, db *sql.DB) (err error) {

	checkExec(t, dbs, db, `create table bar`)
	checkExec(t, dbs, db, `create table bar (test int)`)
	checkExec(t, dbs, db, `insert into bar(test) values(1)`)
	checkQuery(t, dbs, db, `select * from bar where test = 1`)

	checkExec(t, dbs, db,
		`insert into foo(id, name, time) values(?, ?, ?),(?, ?, ?);
				insert into foo(id, name, time) values(161, 'foo', '2018-09-11');
				insert into foo(id, name, time) values(?, ?, ?);`,
		6, "xx", time.Now().AddDate(0, 0, 6).Unix(),
		7, "xxx", time.Now().AddDate(0, 0, 7),
		8, "xxx", float64(time.Now().AddDate(0, 0, 8).Unix())+0.11)

	// half success in one insert
	checkExec(t, dbs, db,
		`insert into foo(id, name, time) values(?, ?, ?),(?, ?, ?)`,
		6, "xx", time.Now().AddDate(0, 0, 6).Unix(),
		17, "xxx", time.Now().AddDate(0, 0, 17))

	checkExec(t, dbs, db, `insert into foo(xxx, time) values(:vv1, :vv2);`,
		sql.Named("vv1", "sss"), sql.Named("vv2", 1536000001.11))

	checkExec(t, dbs, db, `insert into foo(id, name, time) values(162, 'foo', strftime('%s','now'));`)

	checkExec(t, dbs, db, `insert into foo(name, time) values(:vv1, :vv2);`,
		sql.Named("vv1", "sss"), sql.Named("vv2", 1536000001.11))

	checkExec(t, dbs, db, `insert into foo(id, name, time) values(?, :vv1, :vv2);`,
		9,
		sql.Named("vv1", "sss"),
		sql.Named("vv2", float64(time.Now().AddDate(0, 0, 9).Unix())+0.12))

	checkExec(t, dbs, db, `insert into foo(id, name, time) values(:id, :name, :time);`,
		sql.Named("id", 10),
		sql.Named("name", "sss"),
		sql.Named("time", float64(time.Now().AddDate(0, 0, 10).Unix())+0.13))

	dbsstmt, dbstmt := checkPrepare(t, dbs, db, "insert into foo(id, name, time) values(?, ?, ?);")

	for i := 0; i < 2; i++ {
		checkStmtExec(t, dbsstmt, dbstmt, i, fmt.Sprintf("こんにちわ世界%03d", i), time.Now())
	}
	dbstmt.Close()
	dbsstmt.Close()

	dbsstmt, dbstmt = checkPrepare(t, dbs, db, "insert into foo(id, name, time) values(?, ?, ?);")

	for i := 0; i < 100; i++ {
		checkStmtExec(t, dbsstmt, dbstmt,
			i, fmt.Sprintf("こんにちわ世界%03d", i), time.Now().AddDate(0, 0, i))
	}
	dbstmt.Close()
	dbsstmt.Close()

	checkQuery(t, dbs, db, "select id, name, time, xx from foo")

	checkQuery(t, dbs, db, "select id, name, time from foo")

	dbsstmt, dbstmt = checkPrepare(t, dbs, db, "select name from foo where id = ?")
	checkStmtQuery(t, dbsstmt, dbstmt, "1")
	fmt.Println("")
	dbstmt.Close()
	dbsstmt.Close()

	dbsstmt, dbstmt = checkPrepare(t, dbs, db, "select name from foo where id = ?")
	checkStmtQuery(t, dbsstmt, dbstmt, []interface{}{1})
	fmt.Println("")
	dbstmt.Close()
	dbsstmt.Close()

	return
}

func checkQuery(t *testing.T, dbs *sql.DB, db *sql.DB, query string, args ...interface{}) {
	var dberr, dbserr error
	rows, dberr := db.Query(query, args...)
	srows, dbserr := dbs.Query(query, args...)
	if dberr != nil {
		if isSameTypeError(t, dberr, dbserr) {
			return
		} else {
			log.Errorf("\ndb: %v\ndbs: %v\nquery: %s\nargs: %#v\n", dberr, dbserr, query, args)
			t.FailNow()
		}
	}

	dbcol, dberr := rows.Columns()
	dbscol, dbserr := srows.Columns()
	if dberr != nil {
		if isSameTypeError(t, dberr, dbserr) {
			return
		} else {
			log.Errorf("\ndb: %v\ndbs: %v\nquery: %s\nargs: %#v\n", dberr, dbserr, query, args)
			t.FailNow()
		}
	}

	if !reflect.DeepEqual(dbcol, dbscol) {
		log.Errorf("\ndb: %v\ndbs: %v\nquery: %s\nargs: %#v\n", dbcol, dbscol, query, args)
		t.FailNow()
	}

	defer rows.Close()
	defer srows.Close()
	for rows.Next() {
		srows.Next()
		//if !srows.Next() {
		//	log.Errorf("row count mismatch\nquery: %s\nargs: %#v\n", query, args)
		//	t.FailNow()
		//}

		dest := make([]interface{}, len(dbcol))
		destr := make([]interface{}, len(dbcol))
		for i, _ := range dest {
			destr[i] = &dest[i]
		}
		dberr = rows.Scan(destr...)

		sdest := make([]interface{}, len(dbcol))
		sdestr := make([]interface{}, len(dbcol))
		for i, _ := range dest {
			sdestr[i] = &sdest[i]
		}
		dbserr = rows.Scan(sdestr...)
		if dberr != nil {
			if isSameTypeError(t, dberr, dbserr) {
				return
			} else {
				log.Errorf("\ndb: %v\ndbs: %v\nquery: %s\nargs: %#v\n", dberr, dbserr, query, args)
				t.FailNow()
			}
		}

		if !reflect.DeepEqual(destr, sdestr) {
			log.Errorf("\ndb: %v\ndbs: %v\nquery: %s\nargs: %#v\n", destr, sdestr, query, args)
			t.FailNow()
		}

		log.Debugf("query results: %#v", sdest)
	}
	dberr = rows.Err()
	dbserr = srows.Err()
	if dberr != nil {
		if isSameTypeError(t, dberr, dbserr) {
			return
		} else {
			log.Errorf("\ndb: %v\ndbs: %v\nquery: %s\nargs: %#v\n", dberr, dbserr, query, args)
			t.FailNow()
		}
	}

}

func checkStmtQuery(t *testing.T, dbs *sql.Stmt, db *sql.Stmt, args ...interface{}) {
	var dberr, dbserr error
	rows, dberr := db.Query(args...)
	srows, dbserr := dbs.Query(args...)
	if dberr != nil {
		if isSameTypeError(t, dberr, dbserr) {
			return
		} else {
			log.Errorf("\ndb: %v\ndbs: %v\nargs: %#v\n", dberr, dbserr, args)
			t.FailNow()
		}
	}

	dbcol, dberr := rows.Columns()
	dbscol, dbserr := srows.Columns()
	if dberr != nil {
		if isSameTypeError(t, dberr, dbserr) {
			return
		} else {
			log.Errorf("\ndb: %v\ndbs: %v\nargs: %#v\n", dberr, dbserr, args)
			t.FailNow()
		}
	}

	if !reflect.DeepEqual(dbcol, dbscol) {
		log.Errorf("\ndb: %v\ndbs: %v\nargs: %#v\n", dbcol, dbscol, args)
		t.FailNow()
	}

	defer rows.Close()
	defer srows.Close()
	for rows.Next() {
		srows.Next()
		//if !srows.Next() {
		//	log.Errorf("row count mismatch\nquery: %s\nargs: %#v\n", query, args)
		//	t.FailNow()
		//}

		dest := make([]interface{}, len(dbcol))
		destr := make([]interface{}, len(dbcol))
		for i, _ := range dest {
			destr[i] = &dest[i]
		}
		dberr = rows.Scan(destr...)

		sdest := make([]interface{}, len(dbcol))
		sdestr := make([]interface{}, len(dbcol))
		for i, _ := range dest {
			sdestr[i] = &sdest[i]
		}
		dbserr = rows.Scan(sdestr...)
		if dberr != nil {
			if isSameTypeError(t, dberr, dbserr) {
				return
			} else {
				log.Errorf("\ndb: %v\ndbs: %v\nargs: %#v\n", dberr, dbserr, args)
				t.FailNow()
			}
		}

		if !reflect.DeepEqual(destr, sdestr) {
			log.Errorf("\ndb: %v\ndbs: %v\nargs: %#v\n", destr, sdestr, args)
			t.FailNow()
		}

		log.Debugf("query results: %#v", sdest)
	}
	dberr = rows.Err()
	dbserr = srows.Err()
	if dberr != nil {
		if isSameTypeError(t, dberr, dbserr) {
			return
		} else {
			log.Errorf("\ndb: %v\ndbs: %v\nargs: %#v\n", dberr, dbserr, args)
			t.FailNow()
		}
	}
}

func checkExec(t *testing.T, dbs *sql.DB, db *sql.DB, query string, args ...interface{}) {
	var dberr, dbserr error
	_, dberr = db.Exec(query, args...)
	_, dbserr = dbs.Exec(query, args...)
	if dberr != nil {
		if isSameTypeError(t, dberr, dbserr) {
			return
		} else {
			log.Errorf("\ndb: %v\ndbs: %v\nquery: %s\nargs: %#v\n", dberr, dbserr, query, args)
			t.FailNow()
		}
	}
}

func checkStmtExec(t *testing.T, dbs *sql.Stmt, db *sql.Stmt, args ...interface{}) {
	var dberr, dbserr error
	_, dberr = db.Exec(args...)
	_, dbserr = dbs.Exec(args...)
	if dberr != nil {
		if isSameTypeError(t, dberr, dbserr) {
			return
		} else {
			log.Errorf("\ndb: %v\ndbs: %v\nargs: %#v\n", dberr, errors.Cause(dbserr), args)
			t.FailNow()
		}
	}
}

func isSameTypeError(t *testing.T, dberr error, dbserr error) bool {
	root1 := errors.Cause(dberr)
	root2 := errors.Cause(dbserr)
	if root1 == nil || root2 == nil {
		return root1 == root2
	}
	var re = regexp.MustCompile(`_ts_\d{10}`)
	r1 := re.ReplaceAllLiteralString(root1.Error(), "")
	r2 := re.ReplaceAllLiteralString(root2.Error(), "")
	return r1 == r2
}

func checkPrepare(t *testing.T, dbs *sql.DB, db *sql.DB, query string) (dbsstmt, dbstmt *sql.Stmt) {
	var dberr, dbserr error
	dbstmt, dberr = db.Prepare(query)
	dbsstmt, dbserr = dbs.Prepare(query)
	if dberr != nil {
		if isSameTypeError(t, dberr, dbserr) {
			return
		} else {
			log.Errorf("\ndb: %v\ndbs: %v\nquery: %s\n", dberr, dbserr, query)
			t.FailNow()
		}
	}

	return
}

func checkPrepareExec(t *testing.T, dbs *sql.DB, db *sql.DB, query string, args ...[]interface{}) {
	var dberr, dbserr error

	dbstmt, dberr := db.Prepare(query)
	dbsstmt, dbserr := dbs.Prepare(query)
	if dberr != nil {
		if isSameTypeError(t, dberr, dbserr) {
			return
		} else {
			log.Errorf("\ndb: %v\ndbs: %v\nquery: %s\nargs: %#v\n", dberr, dbserr, query, args)
			t.FailNow()
		}
	}

	defer dbstmt.Close()
	for _, arg := range args {
		_, dberr = dbstmt.Exec(arg...)
		_, dbserr = dbsstmt.Exec(arg...)
		if dberr != nil {
			if isSameTypeError(t, dberr, dbserr) {
				return
			} else {
				log.Errorf("\ndb: %v\ndbs: %v\nquery: %s\nargs: %#v\n", dberr, dbserr, query, args)
				t.FailNow()
			}
		}
	}
}
