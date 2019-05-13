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

package main

import (
	"crypto/rand"
	"database/sql"
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"sync/atomic"
	"time"

	"github.com/CovenantSQL/CovenantSQL/client"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/ivpusic/grpool"
)

const (
	tableNamePattern = "insert_table%v"
)

func createSqliteTestTable(db *sql.DB, tableName string) {
	tableDesc := fmt.Sprintf("CREATE TABLE `%s` (`k` INT, `v1` TEXT, PRIMARY KEY(`k`))", tableName)
	if _, err := db.Exec(tableDesc); err != nil {
		log.Fatal(err)
	}
}

func insertData(db *sql.DB, tableName string, dataCount int64, pool *grpool.Pool) {
	var i int64
	start := time.Now()
	log.Infof("LOG: %v start time %v\n", tableName, start.String())
	var errCount int32

	for i = 0; i < dataCount; i++ {
		var vals [1024]byte
		rand.Read(vals[:])
		data := string(vals[:])
		index := i
		pool.WaitCount(1)
		pool.JobQueue <- func() {
			defer pool.JobDone()
			_, err := db.Exec(
				fmt.Sprintf("INSERT INTO `%s` VALUES (?, ?)", tableName),
				index, data,
			)
			if err != nil {
				log.Errorf("Failed to insert data in database: %v %v\n", index, err)
				atomic.AddInt32(&errCount, 1)
			} else {
				atomic.StoreInt32(&errCount, 0)
			}
		}
		if i%1000 == 0 {
			log.Infof("%v Inserted: %v %v\n", time.Now().Sub(start), tableName, i)
		}
		if errCount > 10000 {
			log.Errorf("Error count reach max limit\n")
			break
		}
	}
	log.Infof("LOG: %v end time %v\n", tableName, time.Now().String())
}

func main() {
	log.SetLevel(log.InfoLevel)
	var config, password, dsn string

	flag.StringVar(&config, "config", "./conf/config.yaml", "config file path")
	flag.StringVar(&dsn, "dsn", "", "database url")
	flag.StringVar(&password, "password", "", "master key password for covenantsql")
	flag.Parse()

	go func() {
		http.ListenAndServe("0.0.0.0:6061", nil)
	}()

	err := client.Init(config, []byte(password))
	if err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open("covenantsql", dsn)
	if err != nil {
		log.Fatal(err)
	}

	tableName := fmt.Sprintf(tableNamePattern, 0)
	_, err = db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s;", tableName))
	if err != nil {
		log.Fatal(err)
	}

	createSqliteTestTable(db, tableName)

	pool := grpool.NewPool(16, 32)
	defer pool.Release()
	insertData(db, tableName, 500000000, pool)
	pool.WaitAll()

	err = db.Close()
	if err != nil {
		log.Fatal(err)
	}
}
