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

package client

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/CovenantSQL/CovenantSQL/blockproducer"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

var (
	baseDir        = utils.GetProjectSrcDir()
	testWorkingDir = FJ(baseDir, "./test/GNTE/conf")
	once           sync.Once
)

var FJ = filepath.Join

func BenchmarkCovenantSQLDriver(b *testing.B) {
	var err error
	log.SetLevel(log.DebugLevel)
	err = os.Chdir(testWorkingDir)
	if err != nil {
		log.WithError(err).Error("change working dir failed")
		return
	}

	once.Do(func() {
		log.Debug("benchmarking")
		err = Init(FJ(testWorkingDir, "./node_c/config.yaml"), []byte(""))
		if err != nil {
			b.Fatal(err)
		}
	})

	// wait for chain service
	var ctx1, cancel1 = context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel1()
	err = blockproducer.WaitBPChainService(ctx1, 3*time.Second)
	if err != nil {
		b.Fatalf("wait for chain service failed: %v", err)
	}

	// create
	meta := ResourceMeta{}
	meta.Node = 3
	dsn, err := Create(meta)
	if err != nil {
		b.Fatal(err)
	}

	log.WithField("dsn", dsn).Info("created database")

	db, err := sql.Open("covenantsql", dsn)
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
			log.WithField("result", result).Debug("collected result")
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
