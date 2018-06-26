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

package storage

import (
	"context"
	"fmt"
	"io/ioutil"
	"reflect"
	"testing"
	"time"
)

func TestBadType(t *testing.T) {
	fl, err := ioutil.TempFile("", "sqlite3-")

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	st, err := New(fmt.Sprintf("file:%s", fl.Name()))

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	if err = st.Prepare(context.Background(), struct{}{}); err == nil {
		t.Fatal("Unexpected result: returned nil while expecting an error")
	} else {
		t.Logf("Error occurred as expected: %v", err)
	}

	if err = st.Commit(context.Background(), struct{}{}); err == nil {
		t.Fatal("Unexpected result: returned nil while expecting an error")
	} else {
		t.Logf("Error occurred as expected: %v", err)
	}

	if err = st.Rollback(context.Background(), struct{}{}); err == nil {
		t.Fatal("Unexpected result: returned nil while expecting an error")
	} else {
		t.Logf("Error occurred as expected: %v", err)
	}
}

func TestStorage(t *testing.T) {
	fl, err := ioutil.TempFile("", "sqlite3-")

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	st, err := New(fmt.Sprintf("file:%s", fl.Name()))

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	el1 := &ExecLog{
		ConnectionID: 1,
		SeqNo:        1,
		Timestamp:    time.Now().UnixNano(),
		Queries: []string{
			"CREATE TABLE IF NOT EXISTS `kv` (`key` TEXT PRIMARY KEY, `value` BLOB)",
			"INSERT OR IGNORE INTO `kv` VALUES ('k0', NULL)",
			"INSERT OR IGNORE INTO `kv` VALUES ('k1', 'v1')",
			"INSERT OR IGNORE INTO `kv` VALUES ('k2', 'v2')",
			"INSERT OR IGNORE INTO `kv` VALUES ('k3', 'v3')",
			"INSERT OR REPLACE INTO `kv` VALUES ('k3', 'v3-2')",
			"DELETE FROM `kv` WHERE `key`='k2'",
		},
	}

	el2 := &ExecLog{
		ConnectionID: 1,
		SeqNo:        2,
		Timestamp:    time.Now().UnixNano(),
		Queries: []string{
			"INSERT OR REPLACE INTO `kv` VALUES ('k1', 'v1-2')",
		},
	}

	if err = st.Prepare(context.Background(), el1); err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	if err = st.Prepare(context.Background(), el1); err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	if err = st.Prepare(context.Background(), el2); err == nil {
		t.Fatal("Unexpected result: returned nil while expecting an error")
	} else {
		t.Logf("Error occurred as expected: %v", err)
	}

	if err = st.Commit(context.Background(), el2); err == nil {
		t.Fatal("Unexpected result: returned nil while expecting an error")
	} else {
		t.Logf("Error occurred as expected: %v", err)
	}

	if err = st.Rollback(context.Background(), el2); err == nil {
		t.Fatal("Unexpected result: returned nil while expecting an error")
	} else {
		t.Logf("Error occurred as expected: %v", err)
	}

	if err = st.Commit(context.Background(), el1); err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	// test query
	columns, types, data, err := st.Query(context.Background(), []string{"SELECT * FROM `kv` ORDER BY `key` ASC"})

	if err != nil {
		t.Fatalf("Query failed: %v", err.Error())
	}
	if !reflect.DeepEqual(columns, []string{"key", "value"}) {
		t.Fatalf("Error column result: %v", columns)
	}
	if !reflect.DeepEqual(types, []string{"TEXT", "BLOB"}) {
		t.Fatalf("Error types result: %v", types)
	}
	if len(data) != 3 {
		t.Fatalf("Error result count: %v, should be 3", len(data))
	} else {
		// compare rows
		// FIXME(xq262144), this version of go-sqlite3 driver returns text field in bytes array type
		should1 := []interface{}{[]byte("k0"), nil}
		should2 := []interface{}{[]byte("k1"), []byte("v1")}
		should3 := []interface{}{[]byte("k3"), []byte("v3-2")}
		t.Logf("Rows: %v", data)
		if !reflect.DeepEqual(data[0], should1) {
			t.Fatalf("Error result row: %v, should: %v", data[0], should1)
		}
		if !reflect.DeepEqual(data[1], should2) {
			t.Fatalf("Error result row: %v, should: %v", data[1], should2)
		}
		if !reflect.DeepEqual(data[2], should3) {
			t.Fatalf("Error result row: %v, should: %v", data[2], should2)
		}
	}

	// test query with projection
	columns, types, data, err = st.Query(context.Background(), []string{"SELECT `key` FROM `kv` ORDER BY `key` ASC"})

	if err != nil {
		t.Fatalf("Query failed: %v", err.Error())
	}
	if !reflect.DeepEqual(columns, []string{"key"}) {
		t.Fatalf("Error column result: %v", columns)
	}
	if !reflect.DeepEqual(types, []string{"TEXT"}) {
		t.Fatalf("Error types result: %v", types)
	}
	if len(data) != 3 {
		t.Fatalf("Error result count: %v, should be 3", len(data))
	} else {
		// compare rows
		// FIXME(xq262144), this version of go-sqlite3 driver returns text field in bytes array type
		should1 := []interface{}{[]byte("k0")}
		should2 := []interface{}{[]byte("k1")}
		should3 := []interface{}{[]byte("k3")}
		t.Logf("Rows: %v", data)
		if !reflect.DeepEqual(data[0], should1) {
			t.Fatalf("Error result row: %v, should: %v", data[0], should1)
		}
		if !reflect.DeepEqual(data[1], should2) {
			t.Fatalf("Error result row: %v, should: %v", data[1], should2)
		}
		if !reflect.DeepEqual(data[2], should3) {
			t.Fatalf("Error result row: %v, should: %v", data[2], should2)
		}
	}

	// test query with condition
	columns, types, data, err = st.Query(context.Background(), []string{"SELECT `key` FROM `kv` WHERE `value` IS NOT NULL ORDER BY `key` ASC"})

	if err != nil {
		t.Fatalf("Query failed: %v", err.Error())
	}
	if !reflect.DeepEqual(columns, []string{"key"}) {
		t.Fatalf("Error column result: %v", columns)
	}
	if !reflect.DeepEqual(types, []string{"TEXT"}) {
		t.Fatalf("Error types result: %v", types)
	}
	if len(data) != 2 {
		t.Fatalf("Error result count: %v, should be 3", len(data))
	} else {
		// compare rows
		// FIXME(xq262144), this version of go-sqlite3 driver returns text field in bytes array type
		should1 := []interface{}{[]byte("k1")}
		should2 := []interface{}{[]byte("k3")}
		t.Logf("Rows: %v", data)
		if !reflect.DeepEqual(data[0], should1) {
			t.Fatalf("Error result row: %v, should: %v", data[0], should1)
		}
		if !reflect.DeepEqual(data[1], should2) {
			t.Fatalf("Error result row: %v, should: %v", data[1], should2)
		}
	}

	// test failed query
	columns, types, data, err = st.Query(context.Background(), []string{"SQL???? WHAT!!!!"})

	if err == nil {
		t.Fatalf("Query should failed")
	} else {
		t.Logf("Query failed as expected with: %v", err.Error())
	}

	// test non-read query
	columns, types, data, err = st.Query(context.Background(), []string{"DELETE FROM `kv` WHERE `value` IS NULL"})

	affected, err := st.Exec(context.Background(), []string{"INSERT OR REPLACE INTO `kv` VALUES ('k4', 'v4')"})
	if err != nil || affected != 1 {
		t.Fatalf("Exec INSERT failed: %v", err)
	}
	affected, err = st.Exec(context.Background(), []string{"DELETE FROM `kv` WHERE `key`='k4'"})
	if err != nil || affected != 1 {
		t.Fatalf("Exec DELETE failed: %v", err)
	}
	affected, err = st.Exec(context.Background(), []string{"DELETE FROM `kv` WHERE `key`='noexist'"})
	if err != nil || affected != 0 {
		t.Fatalf("Exec DELETE failed: %v", err)
	}
	// FIXME(xq262144), storage should return err on non-read query

	// test again
	columns, types, data, err = st.Query(context.Background(), []string{"SELECT `key` FROM `kv`"})
	if err != nil {
		t.Fatalf("Query failed: %v", err.Error())
	} else if len(data) != 3 {
		t.Fatalf("Last write query should not take any effect, row count: %v", len(data))
	}

	// test with function
	columns, types, data, err = st.Query(context.Background(), []string{"SELECT COUNT(1) AS `c` FROM `kv`"})
	if err != nil {
		t.Fatalf("Query failed: %v", err.Error())
	} else {
		if len(columns) != 1 {
			t.Fatalf("Query result should contain only one column, now %v", len(columns))
		} else if columns[0] != "c" {
			t.Fatalf("Query result column name is not defined alias, but :%v", columns[0])
		}
		if len(types) != 1 {
			t.Fatalf("Query result should contain only one column, now %v", len(types))
		} else {
			// FIXME(xq262144), dynamic function type is not analyzed sqlite3 itself nor the golang-sqlite3 driver
			t.Logf("Query result type is: %v", types[0])
		}
		if len(data) != 1 || len(data[0]) != 1 {
			t.Fatalf("Query result should contain only one row and one column, now %v", data)
		} else if !reflect.DeepEqual(data[0][0], int64(3)) {
			t.Fatalf("Query result should be table row count 3, but: %v", data[0])
		}
	}
}
