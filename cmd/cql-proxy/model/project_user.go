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
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	gorp "gopkg.in/gorp.v2"
)

type ProjectUserState int16

const (
	ProjectUserStatePreRegistered ProjectUserState = iota
	ProjectUserStateWaitSignedConfirm
	ProjectUserStateEnabled
	ProjectUserStateDisabled
)

var projectStateStrMap = [...]string{
	"PreRegistered",
	"SignedUp",
	"Enabled",
	"Disabled",
}

func (s ProjectUserState) String() string {
	if s >= 0 && int(s) < len(projectStateStrMap) {
		return projectStateStrMap[s]
	}

	return "Unknown"
}

func ParseProjectUserState(s string) (st ProjectUserState, err error) {
	for i, t := range projectStateStrMap {
		if strings.EqualFold(s, t) {
			st = ProjectUserState(i)
			return
		}
	}

	err = errors.New("invalid state")
	return
}

type ProjectUser struct {
	ID          int64            `db:"id"`
	Name        string           `db:"name"`
	Email       string           `db:"email"`
	State       ProjectUserState `db:"state"`
	Provider    string           `db:"provider"`
	ProviderUID string           `db:"provider_uid"`
	RawExtra    []byte           `db:"extra"`
	Extra       gin.H            `db:"-"`
	Created     int64            `db:"created"`
	LastLogin   int64            `db:"last_login"`
}

func (u *ProjectUser) SaveExtra() (err error) {
	u.RawExtra, err = json.Marshal(u.Extra)
	return
}

func (u *ProjectUser) LoadExtra() (err error) {
	err = json.Unmarshal(u.RawExtra, &u.Extra)
	return
}

func PreRegisterUser(db *gorp.DbMap, provider string, name string, email string) (u *ProjectUser, err error) {
	u = &ProjectUser{
		Name:     name,
		Email:    email,
		Provider: provider,
		State:    ProjectUserStatePreRegistered,
		Created:  time.Now().Unix(),
	}

	err = u.SaveExtra()
	if err != nil {
		return
	}

	err = db.Insert(u)

	return
}

func GetProjectUser(db *gorp.DbMap, id int64) (u *ProjectUser, err error) {
	err = db.SelectOne(&u, `SELECT * FROM "____user" WHERE "id" = ? LIMIT 1`,
		id)
	if err != nil {
		return
	}
	err = u.LoadExtra()
	return
}

func GetProjectUsers(db *gorp.DbMap, id []int64) (users []*ProjectUser, err error) {
	if len(id) == 0 {
		return
	}

	var args []interface{}

	for _, v := range id {
		args = append(args, v)
	}

	_, err = db.Select(&users, `SELECT * FROM "____user" WHERE "id" IN (`+
		strings.Repeat("?,", len(id)-1)+`?)`, args...)
	if err != nil {
		return
	}

	for _, u := range users {
		_ = u.LoadExtra()
	}

	return
}

func GetProjectUserList(db *gorp.DbMap, searchTerm string, showOnlyEnabled bool, offset int64, limit int64) (
	users []*ProjectUser, total int64, err error) {
	var (
		sql       = `SELECT * FROM "____user" WHERE 1=1 `
		args      []interface{}
		totalSQL  = `SELECT COUNT(1) AS "cnt" FROM "____user" WHERE 1=1 `
		totalArgs []interface{}
	)

	if showOnlyEnabled {
		sql += ` AND "state" = ? `
		totalSQL += ` AND "state" = ? `
		args = append(args, ProjectUserStateEnabled)
		totalArgs = append(totalArgs, ProjectUserStateEnabled)
	}

	if searchTerm != "" {
		sql += ` AND ("name" LIKE ("%" || ? || "%") OR "email" LIKE ("%" || ? || "%"))`
		args = append(args, searchTerm, searchTerm)
		totalSQL += ` AND ("name" LIKE ("%" || ? || "%") OR "email" LIKE ("%" || ? || "%"))`
		totalArgs = append(totalArgs, searchTerm, searchTerm)
	}

	sql += ` ORDER BY "id" LIMIT ?, ?`
	args = append(args, offset, limit)

	total, err = db.SelectInt(totalSQL, totalArgs...)
	if err != nil {
		return
	}

	_, err = db.Select(&users, sql, args...)
	if err != nil {
		return
	}

	for _, u := range users {
		_ = u.LoadExtra()
	}

	return
}

func EnsureProjectUser(db *gorp.DbMap, provider string,
	uid string, name string, email string, extra gin.H,
	enableSignUp bool, signUpState ProjectUserState) (
	u *ProjectUser, err error) {

	// find by uid
	err = db.SelectOne(&u,
		`SELECT * FROM "____user" WHERE "provider" = ? AND "provider_uid" = ? LIMIT 1`,
		provider, uid)
	exists := true
	now := time.Now().Unix()

	if err != nil {
		// find by provider and email or provider and name
		switch provider {
		case "weibo":
			err = db.SelectOne(&u,
				`SELECT * FROM "____user" WHERE "provider" = ? AND "name" = ? AND "state" = ? LIMIT 1`,
				provider, name, ProjectUserStatePreRegistered)
		default:
			// google, facebook, twitter, etc.
			err = db.SelectOne(&u,
				`SELECT * FROM "____user" WHERE "provider" = ? AND "email" = ? AND "state" = ? LIMIT 1`,
				provider, email, ProjectUserStatePreRegistered)
		}
	}

	if err != nil && !enableSignUp {
		// new user, not even pre-registered
		return
	}

	if err != nil {
		u = &ProjectUser{
			Name:        name,
			Email:       email,
			State:       signUpState,
			Provider:    provider,
			ProviderUID: uid,
			Extra:       extra,
		}

		exists = false
		err = nil
	} else {
		if u.State == ProjectUserStateDisabled {
			// disabled
			err = errors.New("account is disabled")
			return
		}

		u.LastLogin = now
		u.Name = name
		u.Email = email
		u.Extra = extra
	}

	if u.State == ProjectUserStatePreRegistered {
		// for pre-registered user, set to newly signed up state
		u.State = signUpState
		u.ProviderUID = uid
	}

	// encode extra
	err = u.SaveExtra()
	if err != nil {
		return
	}

	// update user info
	if exists {
		_, err = db.Update(u)
	} else {
		err = db.Insert(u)
	}

	if err != nil {
		// return database operation error
		return
	}

	if u.State == ProjectUserStateWaitSignedConfirm {
		// need admin validation
		err = errors.New("account need admin verification")
	}

	return
}

func UpdateProjectUser(db *gorp.DbMap, u *ProjectUser) (err error) {
	if err = u.SaveExtra(); err != nil {
		return
	}

	_, err = db.Update(u)
	return
}
