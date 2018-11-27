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
	"testing"
	"time"

	"github.com/CovenantSQL/CovenantSQL/utils/log"
	. "github.com/smartystreets/goconvey/convey"
)

func TestShardingDriver(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	Convey("", t, func() {
		os.Remove("./foo.db")
		//defer os.Remove("./foo.db")

		db, err := sql.Open(DBSchemeAlias, "./foo.db")
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()
		err = executeSQL(true, db)
		if err != nil {
			log.Fatal(err)
		}
	})
}

func TestSQLite3Driver(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	Convey("", t, func() {
		os.Remove("./foo.db")
		//defer os.Remove("./foo.db")

		db, err := sql.Open("sqlite3", "./foo.db")
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()
		err = executeSQL(false, db)
		if err != nil {
			log.Fatal(err)
		}
	})
}

func executeSQL(isSharding bool, db *sql.DB) (err error) {
	var sqlStmt = fmt.Sprintf(`
	create table if not exists foo%s (id integer not null primary key, name text, time timestamp );
	create index if not exists fooindex%s on foo%s ( time );
	`, ShardSchemaToken, ShardSchemaToken, ShardSchemaToken)
	if isSharding {
		sqlStmt = fmt.Sprintf("SHARDCONFIG foo time %d %d ",
			864000, 1536000000) + sqlStmt
	} else {
		//sqlStmt = fmt.Sprintf(sqlStmt, "", "")
	}

	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
		return
	}

	_, err = db.Exec(`insert into foo(id, name, time) values(?, ?, ?),(?, ?, ?);
							insert into foo(id, name, time) values(61, 'foo', '2018-09-11');
							insert into foo(id, name, time) values(?, ?, ?);`,
		6, "xx", 1536699999,
		7, "xxx", time.Now(),
		8, "xxx", 1536699999.11)
	if err != nil {
		log.Fatal(err)
	}

	//tx, err := db.Begin()
	//if err != nil {
	//	log.Fatal(err)
	//}
	stmt, err := db.Prepare("insert into foo(id, name, time) values(?, ?, ?);")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	for i := 0; i < 2; i++ {
		_, err = stmt.Exec(i, fmt.Sprintf("こんにちわ世界%03d", i), time.Now())
		if err != nil {
			log.Fatal(err)
		}
	}
	//tx.Commit()

	rows, err := db.Query("select id, name, time from foo")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	if rows.Next() {
		log.Fatal("should be empty in table foo")
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec("update foo_ts_0000000000 set name = 'auxten' where id = 1;")
	if err != nil {
		log.Fatal(err)
	}

	rows, err = db.Query("select id, name, time from foo_ts_0000000000")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var name string
		var time interface{}
		err = rows.Scan(&id, &name, &time)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(id, name, time)
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	stmt, err = db.Prepare("select name from foo where id = ?")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	var name string
	err = stmt.QueryRow("1").Scan(&name)
	if err == nil {
		log.Fatal(err)
	}
	err = nil
	fmt.Println(name)

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
	//rows, err = db.Query("select id, name, time from foo")
	//if err != nil {
	//	log.Fatal(err)
	//}
	//defer rows.Close()
	//for rows.Next() {
	//	var id int
	//	var name string
	//	var time time.Time
	//	err = rows.Scan(&id, &name, &time)
	//	if err != nil {
	//		log.Fatal(err)
	//	}
	//	fmt.Println(id, name, time)
	//}
	//err = rows.Err()
	//if err != nil {
	//	log.Fatal(err)
	//}
	return
}
