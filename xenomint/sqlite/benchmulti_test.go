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

package sqlite

import (
	"fmt"
	"math/rand"
	"sync/atomic"
	"testing"

	xi "github.com/CovenantSQL/CovenantSQL/xenomint/interfaces"
)

func BenchSqliteMultiDatabase(b *testing.B, databaseCount, tableCount int, dataCount int64) {
	var sts []xi.Storage

	for i := 0; i < databaseCount; i++ {
		dbname := fmt.Sprintf(databaseNamePattern, i, tableCount, dataCount)
		st := createSqliteStorage(dbname)
		sts = append(sts, st)

		for j := 0; j < tableCount; j++ {
			tableName := fmt.Sprintf(tableNamePattern, j)
			//showSqliteTableCount(st, tableName)
			deleteSqliteTableExtraData(st, tableName, dataCount)
			//showSqliteTableCount(st, tableName)
		}
	}

	var i int64
	testCase := fmt.Sprintf("benchmark SQLite %v Database %v table %v data INSERT", databaseCount, tableCount, dataCount)

	b.Run(testCase, func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				ii := atomic.AddInt64(&i, 1)
				index := dataCount + ii
				databaseIndex := index % int64(databaseCount)
				st := sts[databaseIndex]
				tableName := fmt.Sprintf(tableNamePattern, index%int64(tableCount))
				var vals [1024]byte
				rand.Read(vals[:])
				data := string(vals[:])

				_, err := st.Writer().Exec(
					fmt.Sprintf(`INSERT INTO "%s" VALUES (?, ?)`, tableName),
					index, data,
				)
				if err != nil {
					fmt.Printf("Failed to insert bench data in sqlite multi database test: %v %v\n", index, err)
				}
			}
		})
	})
}

func showSqliteTableCount(st xi.Storage, tableName string) {
	tableDesc := fmt.Sprintf(`select count(*) FROM "%s"`, tableName)
	var count int64
	err := st.Writer().QueryRow(tableDesc).Scan(&count)
	if err != nil {
		fmt.Printf("Failed to show test data table count in bench environment: %v\n", err)
	}
	fmt.Printf("Row count after delete extra data:%v %v %v\n", st, tableName, count)
}

func deleteSqliteTableExtraData(st xi.Storage, tableName string, dataCount int64) {
	tableDel := fmt.Sprintf(`DELETE FROM "%s" WHERE "k" > %v`, tableName, dataCount)
	result, err := st.Writer().Exec(tableDel)
	if err != nil {
		fmt.Printf("Failed to delete extra test data in bench environment: %v\n", err)
	}
	rows, _ := result.RowsAffected()
	fmt.Printf("DELETED rows before test:%v\n", rows)
}

func BenchmarkSqliteMultiDatabases(b *testing.B) {
	//2 database, 1 table, 1m data
	BenchSqliteMultiDatabase(b, 2, 1, 1000000)
	//1 database, 2 table, 1m data
	BenchSqliteMultiDatabase(b, 1, 2, 1000000)

	//2 database, 1 table, 10m data
	//BenchSqliteMultiDatabase(b, 2, 1, 10000000)
	//1 database, 2 table, 10m data
	//BenchSqliteMultiDatabase(b, 1, 2, 10000000)

	//2 database, 1 table, 100m data
	//BenchSqliteMultiDatabase(b, 2, 1, 100000000)
	//1 database, 2 table, 100m data
	//BenchSqliteMultiDatabase(b, 1, 2, 100000000)

	//2 database, 1 table, 500m data
	//BenchSqliteMultiDatabase(b, 2, 1, 500000000)
	//1 database, 2 table, 500m data
	//BenchSqliteMultiDatabase(b, 1, 2, 500000000)
}
