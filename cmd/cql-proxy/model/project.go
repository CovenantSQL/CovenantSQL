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

package model

import (
	gorp "gopkg.in/gorp.v2"

	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/proto"
)

type Project struct {
	ID        int64            `db:"id"`
	DB        proto.DatabaseID `db:"database_id"`
	Alias     string           `db:"alias"`
	Developer int64            `db:"developer_id"`
}

func AddProject(db *gorp.DbMap, dbID proto.DatabaseID, developer int64) (p *Project, err error) {
	p = &Project{
		DB:        dbID,
		Developer: developer,
		Alias:     string(dbID),
	}
	err = db.Insert(p)
	return
}

func GetProject(db *gorp.DbMap, name string) (p *Project, err error) {
	// if the alias fits to a hash, query using database id
	var h hash.Hash
	err = hash.Decode(&h, name)
	if err == nil {
		err = db.SelectOne(&p, `SELECT * FROM "project" WHERE "database_id" = ? LIMIT 1`, name)
	}

	if err == nil {
		return
	}

	err = db.SelectOne(&p, `SELECT * FROM "project" WHERE "alias" = ? LIMIT 1`, name)
	return
}

func GetProjects(db *gorp.DbMap, developer int64) (p []*Project, err error) {
	_, err = db.Select(&p, `SELECT * FROM "project" WHERE "developer_id" = ?`, developer)
	return
}

func HasPrivilege(db *gorp.DbMap, dbID proto.DatabaseID, developer int64) bool {
	recordCnt, err := db.SelectInt(
		`SELECT COUNT(1) AS "cnt" FROM "project" WHERE "database_id" = ? AND "developer_id" = ?`,
		dbID, developer)
	return err == nil && recordCnt > 0
}

func DeleteProject(db *gorp.DbMap, dbID proto.DatabaseID, developer int64) (err error) {
	_, err = db.Exec(
		`DELETE FROM "project" WHERE "database_id" = ? AND "developer_id" = ?`,
		dbID, developer)
	return
}

func SetProjectAlias(db *gorp.DbMap, dbID proto.DatabaseID, developer int64, alias string) (err error) {
	if alias == "" {
		alias = string(dbID)
	}
	_, err = db.Exec(
		`UPDATE "project" SET "alias" = ? WHERE "database_id" = ? AND "developer_id" = ?`,
		alias, dbID, developer)
	return
}
