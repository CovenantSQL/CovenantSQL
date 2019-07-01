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
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	gorp "gopkg.in/gorp.v2"

	"github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy/utils"
)

type Developer struct {
	ID          int64  `db:"id"`
	Name        string `db:"name"`
	Email       string `db:"email"`
	RawExtra    []byte `db:"extra"`
	Created     int64  `db:"created"`
	MainAccount int64  `db:"main_account"`
	LastLogin   int64  `db:"last_login"`
	GithubID    int64  `db:"github_id"`
	Extra       gin.H  `db:"-"`
}

func (d *Developer) PostGet(gorp.SqlExecutor) error {
	return d.LoadExtra()
}

func (d *Developer) PreUpdate(gorp.SqlExecutor) error {
	return d.SaveExtra()
}

func (d *Developer) PreInsert(gorp.SqlExecutor) error {
	return d.SaveExtra()
}

func (d *Developer) SaveExtra() (err error) {
	d.RawExtra, err = json.Marshal(d.Extra)
	return
}

func (d *Developer) LoadExtra() (err error) {
	err = json.Unmarshal(d.RawExtra, &d.Extra)
	return
}

func EnsureDeveloper(db *gorp.DbMap, githubID int64, name string, email string, extra gin.H) (d *Developer, err error) {
	err = db.SelectOne(&d, `SELECT * FROM "developer" WHERE "github_id" = ? LIMIT 1`, githubID)
	exists := true
	now := time.Now().Unix()

	if err != nil {
		// not exists
		d = &Developer{
			Name:      name,
			Email:     email,
			Created:   now,
			LastLogin: now,
			GithubID:  githubID,
			Extra:     extra,
		}

		exists = false
		err = nil
	} else {
		d.LastLogin = now
		d.Name = name
		d.Email = email
		d.Extra = extra
	}

	// update user info
	if exists {
		_, err = db.Update(d)
		if err != nil {
			err = errors.Wrap(err, "update developer user info failed")
		}
	} else {
		err = db.Insert(d)
		if err != nil {
			err = errors.Wrapf(err, "add new developer failed")
		}
	}

	return
}

func SetMainAccount(db *gorp.DbMap, developerID int64, account utils.AccountAddress) (err error) {
	// query account for existence
	ar, err := GetAccount(db, developerID, account)
	if err != nil {
		err = errors.Wrapf(err, "get user account info failed")
		return
	}

	_, err = db.Exec(
		`UPDATE "developer" SET "main_account" = ? WHERE "id" = ?`, ar.ID, developerID)
	if err != nil {
		err = errors.Wrapf(err, "set account as developer main account failed")
	}
	return
}

func SetIfNoMainAccount(db *gorp.DbMap, developerID int64, account utils.AccountAddress) (err error) {
	d, err := GetDeveloper(db, developerID)
	if err != nil {
		err = errors.Wrapf(err, "get developer user info failed")
		return
	}

	if d.MainAccount != 0 {
		_, err = GetAccountByID(db, developerID, d.MainAccount)
		if err == nil {
			return
		}

		err = nil
	}

	err = SetMainAccount(db, developerID, account)
	if err != nil {
		err = errors.Wrapf(err, "set main account failed")
	}

	return
}

func FixDeletedMainAccount(db *gorp.DbMap, developerID int64, account int64) (err error) {
	_, err = db.Exec(
		`UPDATE "developer" SET "main_account" = 0 WHERE "id" = ? AND "main_account" = ?`,
		developerID, account)
	if err != nil {
		err = errors.Wrapf(err, "fix deleted main account failed")
	}
	return
}

func GetDeveloper(db *gorp.DbMap, developerID int64) (d *Developer, err error) {
	err = db.SelectOne(&d,
		`SELECT * FROM "developer" WHERE "id" = ? LIMIT 1`, developerID)
	if err != nil {
		err = errors.Wrapf(err, "get developer info failed")
	}
	return
}

func GetMainAccount(db *gorp.DbMap, developerID int64) (p *DeveloperPrivateKey, err error) {
	d, err := GetDeveloper(db, developerID)
	if err != nil {
		return
	}

	p, err = GetAccountByID(db, developerID, d.MainAccount)
	if err != nil {
		p = nil
		return
	}

	return
}
