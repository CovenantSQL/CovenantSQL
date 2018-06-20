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
		Timestamp:    uint64(time.Now().Unix()),
		Queries: []string{
			"CREATE TABLE IF NOT EXISTS `kv` (`key` TEXT PRIMARY KEY, `value` BLOB)",
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
		Timestamp:    uint64(time.Now().Unix()),
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
}
