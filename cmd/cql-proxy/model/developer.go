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
	"github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy/utils"
	"github.com/gin-gonic/gin"
	"gopkg.in/gorp.v1"
	"time"
)

type Developer struct {
	ID          int64  `db:"id"`
	Name        string `db:"name"`
	Email       string `db:"email"`
	Created     int64  `db:"created"`
	MainAccount int64  `db:"main_account"`
	LastLogin   int64  `db:"last_login"`
	GithubID    int64  `db:"github_id"`
}

func UpdateDeveloper(c *gin.Context, githubID int64, name string, email string) (d *Developer, err error) {
	dbMap := c.MustGet(keyDB).(*gorp.DbMap)
	err = dbMap.SelectOne(&d, `SELECT * FROM "developer" WHERE "github_id" = ? LIMT 1`)
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
		}

		exists = false
	} else {
		d.LastLogin = now
		d.Name = name
		d.Email = email
	}

	// update user info
	if exists {
		_, err = dbMap.Update(d)
	} else {
		err = dbMap.Insert(d)
	}

	return
}

func SetMainAccount(c *gin.Context, developerID int64, account utils.AccountAddress) (err error) {
	// query account for existence
	ar, err := GetAccount(c, developerID, account)
	if err != nil {
		return
	}

	_, err = c.MustGet(keyDB).(*gorp.DbMap).Exec(
		`UPDATE "developer" SET "main_account" = ? WHERE "id" = ?`, ar.ID, developerID)
	return
}

func SetIfNoMainAccount(c *gin.Context, developerID int64, account utils.AccountAddress) (err error) {
	d, err := GetDeveloper(c, developerID)
	if err != nil {
		return
	}

	if d.MainAccount != 0 {
		_, err = GetAccountByID(c, developerID, d.MainAccount)
		if err == nil {
			return
		}

		err = nil
	}

	err = SetMainAccount(c, developerID, account)

	return
}

func FixDeletedMainAccount(c *gin.Context, developerID int64, account int64) (err error) {
	_, err = c.MustGet(keyDB).(*gorp.DbMap).Exec(
		`UPDATE "developer" SET "main_account" = 0 WHERE "id" = ? AND "main_account" = ?`,
		developerID, account)
	return
}

func GetDeveloper(c *gin.Context, developerID int64) (d *Developer, err error) {
	err = c.MustGet(keyDB).(*gorp.DbMap).SelectOne(&d,
		`SELECT * FROM "developer" WHERE "id" = ? LIMIT 1`, developerID)
	return
}

func GetMainAccount(c *gin.Context, developerID int64) (p *DeveloperPrivateKey, err error) {
	d, err := GetDeveloper(c, developerID)
	if err != nil {
		return
	}

	p, err = GetAccountByID(c, developerID, d.MainAccount)
	if err != nil {
		p = nil
		return
	}

	return
}

func init() {
	RegisterModel("developer", Developer{}, "ID", true)
}
