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
	"database/sql"
	"flag"
	"fmt"

	_ "github.com/CovenantSQL/CovenantSQL/client"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

func main() {
	log.SetLevel(log.InfoLevel)
	var dsn string

	flag.StringVar(&dsn, "dsn", "", "Database url")
	flag.Parse()

	// If your CovenantSQL config.yaml is not in ~/.cql/config.yaml
	// Uncomment and edit following code
	/*
			config := "/data/myconfig/config.yaml"
			password := "mypassword"
		    err := client.Init(config, []byte(password))
		    if err != nil {
		        log.Fatal(err)
		    }
	*/

	db, err := sql.Open("covenantsql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec("DROP TABLE IF EXISTS testSimple;")
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec("CREATE TABLE testSimple ( indexedColumn, nonIndexedColumn );")
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec("CREATE INDEX testIndexedColumn ON testSimple ( indexedColumn );")
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec("INSERT INTO testSimple VALUES(?, ?);", 4, 400)
	if err != nil {
		log.Fatal(err)
	}

	row := db.QueryRow("SELECT nonIndexedColumn FROM testSimple LIMIT 1;")

	var result int
	err = row.Scan(&result)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("SELECT nonIndexedColumn FROM testSimple LIMIT 1; result %d\n", result)

}
