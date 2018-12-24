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

package storage

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"reflect"
	"testing"
	"time"
)

func newQuery(query string, args ...interface{}) (q Query) {
	q.Pattern = query

	// convert args
	q.Args = make([]sql.NamedArg, len(args))
	for i, v := range args {
		q.Args[i] = sql.Named("", v)
	}

	return
}

func newNamedQuery(query string, args map[string]interface{}) (q Query) {
	q.Pattern = query
	q.Args = make([]sql.NamedArg, len(args))
	i := 0

	// convert args
	for n, v := range args {
		q.Args[i] = sql.Named(n, v)
		i++
	}

	return
}

func TestBadType(t *testing.T) {
	fl, err := ioutil.TempFile("", "sqlite3-")

	if err != nil {
		t.Fatalf("error occurred: %v", err)
	}

	st, err := New(fmt.Sprintf("file:%s", fl.Name()))

	if err != nil {
		t.Fatalf("error occurred: %v", err)
	}

	if err = st.Prepare(context.Background(), struct{}{}); err == nil {
		t.Fatal("unexpected result: returned nil while expecting an error")
	} else {
		t.Logf("Error occurred as expected: %v", err)
	}

	if _, err = st.Commit(context.Background(), struct{}{}); err == nil {
		t.Fatal("unexpected result: returned nil while expecting an error")
	} else {
		t.Logf("Error occurred as expected: %v", err)
	}

	if err = st.Rollback(context.Background(), struct{}{}); err == nil {
		t.Fatal("unexpected result: returned nil while expecting an error")
	} else {
		t.Logf("Error occurred as expected: %v", err)
	}
}

func TestStorage(t *testing.T) {
	fl, err := ioutil.TempFile("", "sqlite3-")

	if err != nil {
		t.Fatalf("error occurred: %v", err)
	}

	st, err := New(fmt.Sprintf("file:%s", fl.Name()))

	if err != nil {
		t.Fatalf("error occurred: %v", err)
	}

	el1 := &ExecLog{
		ConnectionID: 1,
		SeqNo:        1,
		Timestamp:    time.Now().UnixNano(),
		Queries: []Query{
			newQuery("CREATE TABLE IF NOT EXISTS `kv` (`key` TEXT PRIMARY KEY, `value` BLOB)"),
			newQuery("INSERT OR IGNORE INTO `kv` VALUES ('k0', NULL)"),
			newQuery("INSERT OR IGNORE INTO `kv` VALUES ('k1', 'v1')"),
			newQuery("INSERT OR IGNORE INTO `kv` VALUES ('k2', 'v2')"),
			newQuery("INSERT OR IGNORE INTO `kv` VALUES ('k3', 'v3')"),
			newQuery("INSERT OR REPLACE INTO `kv` VALUES ('k3', 'v3-2')"),
			newQuery("DELETE FROM `kv` WHERE `key`='k2'"),
		},
	}

	el2 := &ExecLog{
		ConnectionID: 1,
		SeqNo:        2,
		Timestamp:    time.Now().UnixNano(),
		Queries: []Query{
			newQuery("INSERT OR REPLACE INTO `kv` VALUES ('k1', 'v1-2')"),
		},
	}

	if err = st.Prepare(context.Background(), el1); err != nil {
		t.Fatalf("error occurred: %v", err)
	}

	if err = st.Prepare(context.Background(), el1); err != nil {
		t.Fatalf("error occurred: %v", err)
	}

	if err = st.Prepare(context.Background(), el2); err == nil {
		t.Fatal("unexpected result: returned nil while expecting an error")
	} else {
		t.Logf("Error occurred as expected: %v", err)
	}

	if _, err = st.Commit(context.Background(), el2); err == nil {
		t.Fatal("unexpected result: returned nil while expecting an error")
	} else {
		t.Logf("Error occurred as expected: %v", err)
	}

	if err = st.Rollback(context.Background(), el2); err == nil {
		t.Fatal("unexpected result: returned nil while expecting an error")
	} else {
		t.Logf("Error occurred as expected: %v", err)
	}

	var res interface{}
	if res, err = st.Commit(context.Background(), el1); err != nil {
		t.Fatalf("error occurred: %v", err)
	} else {
		result := res.(ExecResult)
		t.Logf("Result: %v", result)
	}

	// test query
	columns, types, data, err := st.Query(context.Background(),
		[]Query{newQuery("SELECT * FROM `kv` ORDER BY `key` ASC")})

	if err != nil {
		t.Fatalf("query failed: %v", err.Error())
	}
	if !reflect.DeepEqual(columns, []string{"key", "value"}) {
		t.Fatalf("error column result: %v", columns)
	}
	if !reflect.DeepEqual(types, []string{"TEXT", "BLOB"}) {
		t.Fatalf("error types result: %v", types)
	}
	if len(data) != 3 {
		t.Fatalf("error result count: %v, should be 3", len(data))
	} else {
		// compare rows
		should1 := []interface{}{[]byte("k0"), nil}
		should2 := []interface{}{[]byte("k1"), []byte("v1")}
		should3 := []interface{}{[]byte("k3"), []byte("v3-2")}
		t.Logf("Rows: %v", data)
		if !reflect.DeepEqual(data[0], should1) {
			t.Fatalf("error result row: %v, should: %v", data[0], should1)
		}
		if !reflect.DeepEqual(data[1], should2) {
			t.Fatalf("error result row: %v, should: %v", data[1], should2)
		}
		if !reflect.DeepEqual(data[2], should3) {
			t.Fatalf("error result row: %v, should: %v", data[2], should2)
		}
	}

	// test query with projection
	columns, types, data, err = st.Query(context.Background(),
		[]Query{newQuery("SELECT `key` FROM `kv` ORDER BY `key` ASC")})

	if err != nil {
		t.Fatalf("query failed: %v", err.Error())
	}
	if !reflect.DeepEqual(columns, []string{"key"}) {
		t.Fatalf("error column result: %v", columns)
	}
	if !reflect.DeepEqual(types, []string{"TEXT"}) {
		t.Fatalf("error types result: %v", types)
	}
	if len(data) != 3 {
		t.Fatalf("error result count: %v, should be 3", len(data))
	} else {
		// compare rows
		should1 := []interface{}{[]byte("k0")}
		should2 := []interface{}{[]byte("k1")}
		should3 := []interface{}{[]byte("k3")}
		t.Logf("Rows: %v", data)
		if !reflect.DeepEqual(data[0], should1) {
			t.Fatalf("error result row: %v, should: %v", data[0], should1)
		}
		if !reflect.DeepEqual(data[1], should2) {
			t.Fatalf("error result row: %v, should: %v", data[1], should2)
		}
		if !reflect.DeepEqual(data[2], should3) {
			t.Fatalf("error result row: %v, should: %v", data[2], should2)
		}
	}

	// test query with condition
	columns, types, data, err = st.Query(context.Background(),
		[]Query{newQuery("SELECT `key` FROM `kv` WHERE `value` IS NOT NULL ORDER BY `key` ASC")})

	if err != nil {
		t.Fatalf("query failed: %v", err.Error())
	}
	if !reflect.DeepEqual(columns, []string{"key"}) {
		t.Fatalf("error column result: %v", columns)
	}
	if !reflect.DeepEqual(types, []string{"TEXT"}) {
		t.Fatalf("error types result: %v", types)
	}
	if len(data) != 2 {
		t.Fatalf("error result count: %v, should be 3", len(data))
	} else {
		// compare rows
		should1 := []interface{}{[]byte("k1")}
		should2 := []interface{}{[]byte("k3")}
		t.Logf("Rows: %v", data)
		if !reflect.DeepEqual(data[0], should1) {
			t.Fatalf("error result row: %v, should: %v", data[0], should1)
		}
		if !reflect.DeepEqual(data[1], should2) {
			t.Fatalf("error result row: %v, should: %v", data[1], should2)
		}
	}

	// test failed query
	columns, types, data, err = st.Query(context.Background(), []Query{newQuery("SQL???? WHAT!!!!")})

	if err == nil {
		t.Fatal("query should failed")
	} else {
		t.Logf("Query failed as expected with: %v", err.Error())
	}

	// test non-read query
	columns, types, data, err = st.Query(context.Background(),
		[]Query{newQuery("DELETE FROM `kv` WHERE `value` IS NULL")})

	execResult, err := st.Exec(context.Background(),
		[]Query{newQuery("INSERT OR REPLACE INTO `kv` VALUES ('k4', 'v4')")})
	if err != nil || execResult.RowsAffected != 1 {
		t.Fatalf("exec INSERT failed: %v", err)
	}
	// test with arguments
	execResult, err = st.Exec(context.Background(), []Query{newQuery("DELETE FROM `kv` WHERE `key`='k4'")})
	if err != nil || execResult.RowsAffected != 1 {
		t.Fatalf("exec DELETE failed: %v", err)
	}
	execResult, err = st.Exec(context.Background(),
		[]Query{newQuery("DELETE FROM `kv` WHERE `key`=?", "not_exist")})
	if err != nil || execResult.RowsAffected != 0 {
		t.Fatalf("exec DELETE failed: %v", err)
	}

	// test again
	columns, types, data, err = st.Query(context.Background(), []Query{newQuery("SELECT `key` FROM `kv`")})
	if err != nil {
		t.Fatalf("query failed: %v", err.Error())
	} else if len(data) != 3 {
		t.Fatalf("last write query should not take any effect, row count: %v", len(data))
	} else {
		t.Logf("Rows: %v", data)
	}

	// test with select
	columns, types, data, err = st.Query(context.Background(),
		[]Query{newQuery("SELECT `key` FROM `kv` WHERE `key` IN (?)", "k1")})
	if err != nil {
		t.Fatalf("query failed: %v", err.Error())
	} else if len(data) != 1 {
		t.Fatalf("should only have one record, but actually %v", len(data))
	} else {
		t.Logf("Rows: %v", data)
	}

	// test with select with named arguments
	columns, types, data, err = st.Query(context.Background(),
		[]Query{newNamedQuery("SELECT `key` FROM `kv` WHERE `key` IN (:test2, :test1)", map[string]interface{}{
			"test1": "k1",
			"test2": "k3",
		})})
	if err != nil {
		t.Fatalf("query failed: %v", err.Error())
	} else if len(data) != 2 {
		t.Fatalf("should only have two records, but actually %v", len(data))
	} else {
		t.Logf("Rows: %v", data)
	}

	// test with function
	columns, types, data, err = st.Query(context.Background(),
		[]Query{newQuery("SELECT COUNT(1) AS `c` FROM `kv`")})
	if err != nil {
		t.Fatalf("query failed: %v", err.Error())
	} else {
		if len(columns) != 1 {
			t.Fatalf("query result should contain only one column, now %v", len(columns))
		} else if columns[0] != "c" {
			t.Fatalf("query result column name is not defined alias, but :%v", columns[0])
		}
		if len(types) != 1 {
			t.Fatalf("query result should contain only one column, now %v", len(types))
		} else {
			t.Logf("Query result type is: %v", types[0])
		}
		if len(data) != 1 || len(data[0]) != 1 {
			t.Fatalf("query result should contain only one row and one column, now %v", data)
		} else if !reflect.DeepEqual(data[0][0], int64(3)) {
			t.Fatalf("query result should be table row count 3, but: %v", data[0])
		}
	}

	// test with timestamp fields
	_, err = st.Exec(context.Background(), []Query{
		newQuery("CREATE TABLE `tm` (tm TIMESTAMP)"),
		newQuery("INSERT INTO `tm` VALUES(DATE('NOW'))"),
	})
	if err != nil {
		t.Fatalf("query failed: %v", err.Error())
	} else {
		// query for values
		_, _, data, err = st.Query(context.Background(), []Query{newQuery("SELECT `tm` FROM `tm`")})
		if len(data) != 1 || len(data[0]) != 1 {
			t.Fatalf("query result should contain only one row and one column, now %v", data)
		} else if !reflect.TypeOf(data[0][0]).AssignableTo(reflect.TypeOf(time.Time{})) {
			t.Fatalf("query result should be time.Time type, but: %v", reflect.TypeOf(data[0][0]).String())
		}
	}
}
