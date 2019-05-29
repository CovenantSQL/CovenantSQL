/*
 * Copyright 2019 The CovenantSQL Authors.
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
	"database/sql"
	"github.com/CovenantSQL/CovenantSQL/client"
	"github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy/config"
	"gopkg.in/gorp.v1"
)

func NewStorage(cfg *config.StorageConfig) (storage *gorp.DbMap, err error) {
	var db *sql.DB

	if cfg == nil {
		// using test database
		db, err = sql.Open("sqlite3", "file::memory:?mode=memory&cache=shared")
	} else if cfg.UseLocalDatabase {
		db, err = sql.Open("sqlite3", cfg.DatabaseID)
	} else {
		dsnCfg := client.NewConfig()
		dsnCfg.DatabaseID = cfg.DatabaseID
		db, err = sql.Open("covenantsql", dsnCfg.FormatDSN())
	}

	if err != nil {
		return
	}

	storage = &gorp.DbMap{Db: db, Dialect: gorp.SqliteDialect{}}
	return
}
