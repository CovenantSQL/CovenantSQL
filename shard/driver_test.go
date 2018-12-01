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
	"testing"
	"time"

	"github.com/CovenantSQL/CovenantSQL/utils/log"
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

func checkQuery(t *testing.T, dbs *sql.DB, db *sql.DB, query string, args ...interface{}) {
	var dberr, dbserr error
	rows, dberr := db.Query(query, args...)
	srows, dbserr := dbs.Query(query, args...)
	if dberr != nil {
		if reflect.DeepEqual(dberr, dbserr) {
			return
		} else {
			log.Errorf("\ndb: %v\ndbs: %v\nquery: %s\nargs: %#v\n", dberr, dbserr, query, args)
			t.FailNow()
		}
	}

	dbcol, dberr := rows.Columns()
	dbscol, dbserr := srows.Columns()
	if dberr != nil {
		if reflect.DeepEqual(dberr, dbserr) {
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
			if reflect.DeepEqual(dberr, dbserr) {
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
		if reflect.DeepEqual(dberr, dbserr) {
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
		if reflect.DeepEqual(dberr, dbserr) {
			return
		} else {
			log.Errorf("\ndb: %v\ndbs: %v\nargs: %#v\n", dberr, dbserr, args)
			t.FailNow()
		}
	}

	dbcol, dberr := rows.Columns()
	dbscol, dbserr := srows.Columns()
	if dberr != nil {
		if reflect.DeepEqual(dberr, dbserr) {
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
			if reflect.DeepEqual(dberr, dbserr) {
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
		if reflect.DeepEqual(dberr, dbserr) {
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
		if reflect.DeepEqual(dberr, dbserr) {
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
		if reflect.DeepEqual(dberr, dbserr) {
			return
		} else {
			log.Errorf("\ndb: %v\ndbs: %v\nargs: %#v\n", dberr, dbserr, args)
			t.FailNow()
		}
	}
}

func checkPrepare(t *testing.T, dbs *sql.DB, db *sql.DB, query string) (dbsstmt, dbstmt *sql.Stmt) {
	var dberr, dbserr error
	dbstmt, dberr = db.Prepare(query)
	dbsstmt, dbserr = dbs.Prepare(query)
	if dberr != nil {
		if reflect.DeepEqual(dberr, dbserr) {
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
		if reflect.DeepEqual(dberr, dbserr) {
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
			if reflect.DeepEqual(dberr, dbserr) {
				return
			} else {
				log.Errorf("\ndb: %v\ndbs: %v\nquery: %s\nargs: %#v\n", dberr, dbserr, query, args)
				t.FailNow()
			}
		}
	}
}

func testCases(t *testing.T, dbs *sql.DB, db *sql.DB) (err error) {

	checkExec(t, dbs, db,
		`insert into foo(id, name, time) values(?, ?, ?),(?, ?, ?);
				insert into foo(id, name, time) values(61, 'foo', '2018-09-11');
				insert into foo(id, name, time) values(?, ?, ?);`,
		6, "xx", 1536699999,
		7, "xxx", time.Now(),
		8, "xxx", 1536699999.11)

	checkExec(t, dbs, db, `insert into foo(name, time) values(:vv1, :vv2);`,
		sql.Named("vv1", "sss"), sql.Named("vv2", 1536699988.11))

	checkExec(t, dbs, db, `insert into foo(id, name, time) values(?, :vv1, :vv2);`,
		9, sql.Named("vv1", "sss"), sql.Named("vv2", 1536699911.11))

	checkExec(t, dbs, db, `insert into foo(id, name, time) values(:id, :name, :time);`,
		sql.Named("id", 10),
		sql.Named("name", "sss"),
		sql.Named("time", 1536111111.11))

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

	//checkExec(t, dbs, db, `insert into foo_ts_0000000000(id, name, time) values(61, 'sss', '2018-09-11');`)
	//
	//checkQuery(t, dbs, db, `select id, id, name, time from foo_ts_0000000000 limit 1;`+
	//	"select count(1), max(id), id, foo_ts_0000000000.name, time from foo_ts_0000000000 where id < ? and name = :ll limit 10;",
	//	100, sql.Named("ll", "sss"))

	checkQuery(t, dbs, db, "select id, name, time from foo")

	//checkExec(t, dbs, db, "update foo_ts_0000000000 set name = 'auxten' where id = 1;")
	//
	//checkQuery(t, dbs, db, "select id, name, time from foo_ts_0000000000")
	//fmt.Println("")

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

	//_, err = db.Exec("delete from foo")
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	//_, err = db.Exec("insert into foo(id, name, time) values(1, 'foo', '2018-09-11'), (2, 'bar', '2018-09-12'), (3, 'baz', '2018-09-13')")
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	////_, err = db.Exec("insert into foo(id, name) values(4, 'foo');insert into foo(id, name) values(5, 'bar');")
	////if err != nil {
	////	log.Fatal(err)
	////}
	////
	////_, err = db.Exec("insert into foo(id, name) values(?, ?);insert into foo(id, name) values(?, ?);", 6, "xx", 7, "xxx")
	////if err != nil {
	////	log.Fatal(err)
	////}
	//

	return
}
