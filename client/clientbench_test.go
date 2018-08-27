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
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

var (
	baseDir        = utils.GetProjectSrcDir()
	testWorkingDir = FJ(baseDir, "./test/GNTE/conf")
	logDir         = FJ(testWorkingDir, "./log/")
	once           sync.Once
)

var FJ = filepath.Join

func BenchmarkThunderDBDriver(b *testing.B) {
	var err error
	log.SetLevel(log.DebugLevel)
	err = os.Chdir(testWorkingDir)
	if err != nil {
		log.Errorf("change working dir failed: %s", err)
		return
	}

	once.Do(func() {
		log.Debug("benchmarking")
		err = Init(FJ(testWorkingDir, "./node_c/config.yaml"), []byte(""))
		if err != nil {
			b.Fatal(err)
		}
	})

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

	b.Run("benchmark insert", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err = db.Exec("INSERT INTO test VALUES(?)", i)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("benchmark select", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			row := db.QueryRow("SELECT * FROM test LIMIT 1")

			var result int
			err = row.Scan(&result)
			if err != nil || result < 0 {
				b.Fatal(err)
			}
			log.Debugf("result %d", result)
		}
	})
	err = db.Close()
	if err != nil {
		b.Fatal(err)
	}
	err = Drop(dsn)
	if err != nil {
		b.Fatal(err)
	}
}
