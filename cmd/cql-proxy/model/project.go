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
	gorp "gopkg.in/gorp.v1"

	"github.com/CovenantSQL/CovenantSQL/proto"
)

type Project struct {
	ID        int64            `db:"id"`
	DB        proto.DatabaseID `db:"database_id"`
	Developer int64            `db:"developer_id"`
}

func BindProject(db *gorp.DbMap, dbID proto.DatabaseID, developer int64) (p *Project, err error) {
	p = &Project{
		DB:        dbID,
		Developer: developer,
	}
	err = db.Insert(p)
	return
}

func HasPrivilege(db *gorp.DbMap, dbID proto.DatabaseID, developer int64) bool {
	recordCnt, err := db.SelectInt(
		`SELECT COUNT(1) AS "cnt" FROM "project" WHERE "database_id" = ? AND "developer_id" = ?`,
		dbID, developer)
	return err == nil && recordCnt > 0
}

func GetProject(db *gorp.DbMap, dbID proto.DatabaseID) (p *Project, err error) {
	err = db.SelectOne(&p,
		`SELECT * FROM "project" WHERE "database_id" = ? AND "developer_id" = ? LIMIT 1`)
	return
}

func UnbindProject(db *gorp.DbMap, dbID proto.DatabaseID, developer int64) (err error) {
	_, err = db.Exec(
		`DELETE FROM "project" WHERE "database_id" = ? AND "developer_id" = ?`,
		dbID, developer)
	return
}

func DeleteProject(db *gorp.DbMap, dbID proto.DatabaseID) (err error) {
	_, err = db.Exec(
		`DELETE FROM "project" WHERE "database_id" = ?`, dbID)
	return
}

func (p *Project) GetDB() (dbMap *gorp.DbMap, err error) {
	// impersonate user key to request
	return
}
