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

package client

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"
	"os"

	"gitlab.com/thunderdb/ThunderDB/utils"
	"gitlab.com/thunderdb/ThunderDB/utils/log"
)

var (
	baseDir        = utils.GetProjectSrcDir()
	testWorkingDir = FJ(baseDir, "./test/GNTE/conf")
	logDir         = FJ(testWorkingDir, "./log/")
)

var FJ = filepath.Join

func BenchmarkThunderDBDriver(b *testing.B) {
	var err error
	err = os.Chdir(testWorkingDir)
	if err != nil {
		log.Errorf("change working dir failed: %s", err)
		return
	}


	err = Init(FJ(testWorkingDir, "./node_c/config.yaml"), []byte(""))
	if err != nil {
		b.Fatal(err)
	}

	// create
	dsn, err := Create(ResourceMeta{Node: 3})
	if err != nil {
		b.Fatal(err)
	}

	log.Infof("the created database dsn is %v", dsn)

	db, err := sql.Open("thunderdb", dsn)
	if err != nil {
		b.Fatal(err)
	}
	_, err = db.Exec("CREATE TABLE test (test int)")
	if err != nil {
		b.Fatal(err)
	}
	_, err = db.Exec("INSERT INTO test VALUES(?)", 4)
	if err != nil {
		b.Fatal(err)
	}

	row := db.QueryRow("SELECT * FROM test LIMIT 1")

	var result int
	err = row.Scan(&result)

	// test timestamp fields
	_, err = db.Exec("CREATE TABLE test_time (test timestamp)")

	_, err = db.Exec("INSERT INTO test_time VALUES(DATE('NOW'))")

	row = db.QueryRow("SELECT * FROM test_time LIMIT 1")

	var tmResult time.Time
	err = row.Scan(&tmResult)

	err = db.Close()

	err = Drop(dsn)

	//b.ResetTimer()
	//for i := 0; i < b.N; i++ {
	//}
}

