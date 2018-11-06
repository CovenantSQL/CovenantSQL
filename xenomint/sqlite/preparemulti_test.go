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
	"path"
	"testing"

	xi "github.com/CovenantSQL/CovenantSQL/xenomint/interfaces"
)

const (
	databaseNamePattern = "sqlitedb%v_%v_%v"
	tableNamePattern    = "table%v"
)

func prepareSqliteStorages(databaseCount, tableCount int, dataCount int64) {
	for i := 0; i < databaseCount; i++ {
		dbname := fmt.Sprintf(databaseNamePattern, i, tableCount, dataCount)
		st := createSqliteStorage(dbname)

		for j := 0; j < tableCount; j++ {
			tableName := fmt.Sprintf(tableNamePattern, j)
			createSqliteTestTable(st, tableName)

			var index int64
			for index = 0; index < dataCount; {
				start := index
				end := index + 10000
				insertSqliteTableData(st, tableName, start, end)
				index = end
				fmt.Printf("Now insert index %v\n", index)
			}
		}
	}
}

func createSqliteStorage(dbname string) xi.Storage {
	fl := path.Join("./", dbname)

	st, err := NewSqlite(fmt.Sprint("file:", fl))
	if err != nil {
		fmt.Printf("Failed to create sqlite database bench environment: %v\n", err)
		return nil
	}

	return st
}

func createSqliteTestTable(st xi.Storage, tableName string) {
	tableDesc := fmt.Sprintf(`CREATE TABLE "%s" ("k" INT, "v1" TEXT, PRIMARY KEY("k"))`, tableName)
	if _, err := st.Writer().Exec(tableDesc); err != nil {
		fmt.Printf("Failed to create table in bench environment: %v\n", err)
	}
}

func insertSqliteTableData(st xi.Storage, tableName string, start, end int64) {
	tx, err := st.Writer().Begin()
	if err != nil {
		fmt.Printf("Failed to create transaction in bench environment: %v\n", err)
	}

	stmt, err := tx.Prepare(
		fmt.Sprintf(`INSERT INTO "%s" VALUES (?, ?)`, tableName),
	)
	if err != nil {
		fmt.Printf("Failed to prepare insert data in bench environment: %v\n", err)
	}

	var i int64
	for i = start; i < end; i++ {
		var (
			vals [333]byte
			args [2]interface{}
		)
		args[0] = i
		rand.Read(vals[:])
		args[1] = string(vals[:])

		if _, err = stmt.Exec(args[0], args[1]); err != nil {
			fmt.Printf("Failed to insert data in bench environment: %v %v\n", args[0], err)
		}
	}

	if err = tx.Commit(); err != nil {
		fmt.Printf("Failed to commit data in bench environment: %v\n", err)
	}
}

func TestPrepareSqliteStorages(t *testing.T) {
	//2 database, 1 table, 1m data
	prepareSqliteStorages(2, 1, 1000000)
	//1 database, 2 table, 1m data
	prepareSqliteStorages(1, 2, 1000000)

	//2 database, 1 table, 10m data
	//prepareSqliteStorages(2, 1, 10000000)
	//1 database, 2 table, 10m data
	//prepareSqliteStorages(1, 2, 10000000)

	//2 database, 1 table, 100m data
	//prepareSqliteStorages(2, 1, 100000000)
	//1 database, 2 table, 100m data
	//prepareSqliteStorages(1, 2, 100000000)

	//2 database, 1 table, 500m data
	//prepareSqliteStorages(2, 1, 500000000)
	//1 database, 2 table, 100m data
	//prepareSqliteStorages(1, 2, 500000000)
}
