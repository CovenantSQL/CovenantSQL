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

// ProjectUserState defines the state for project user.
type ProjectUserState int16

const (
	// ProjectUserStatePreRegistered represents user just being pre-registered.
	ProjectUserStatePreRegistered ProjectUserState = iota
	// ProjectUserStateWaitSignedConfirm represents user signed up and need developer confirmation.
	ProjectUserStateWaitSignedConfirm
	// ProjectUserStateEnabled represents user enabled for activities.
	ProjectUserStateEnabled
	// ProjectUserStateDisabled represents user being disabled.
	ProjectUserStateDisabled
)

var projectStateStrMap = [...]string{
	"PreRegistered",
	"SignedUp",
	"Enabled",
	"Disabled",
}

// String implements Stringer for user state stringify.
func (s ProjectUserState) String() string {
	if s >= 0 && int(s) < len(projectStateStrMap) {
		return projectStateStrMap[s]
	}

	return "Unknown"
}

// ParseProjectUserState parse user state string to enum.
func ParseProjectUserState(s string) (st ProjectUserState, err error) {
	for i, t := range projectStateStrMap {
		if strings.EqualFold(s, t) {
			st = ProjectUserState(i)
			return
		}
	}

	err = errors.New("invalid user state")
	return
}

// ProjectUser defines project user info object.
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

// PostGet implements gorp.HasPostGet interface.
func (u *ProjectUser) PostGet(gorp.SqlExecutor) error {
	return u.LoadExtra()
}

// PreUpdate implements gorp.HasPreUpdate interface.
func (u *ProjectUser) PreUpdate(gorp.SqlExecutor) error {
	return u.SaveExtra()
}

// PreInsert implements gorp.HasPreInsert interface.
func (u *ProjectUser) PreInsert(gorp.SqlExecutor) error {
	return u.SaveExtra()
}

// SaveExtra marshal extra user info to bytes form.
func (u *ProjectUser) SaveExtra() (err error) {
	u.RawExtra, err = json.Marshal(u.Extra)
	return
}

// LoadExtra unmarshal extra user info object from database.
func (u *ProjectUser) LoadExtra() (err error) {
	err = json.Unmarshal(u.RawExtra, &u.Extra)
	return
}

// PreRegisterUser manually adds new user to database.
func PreRegisterUser(db *gorp.DbMap, provider string, name string, email string) (u *ProjectUser, err error) {
	u = &ProjectUser{
		Name:     name,
		Email:    email,
		Provider: provider,
		State:    ProjectUserStatePreRegistered,
		Created:  time.Now().Unix(),
	}

	err = db.Insert(u)
	if err != nil {
		err = errors.Wrapf(err, "add pre-registered user failed")
	}

	return
}

// GetProjectUser get project user info of specified uid.
func GetProjectUser(db *gorp.DbMap, id int64) (u *ProjectUser, err error) {
	err = db.SelectOne(&u, `SELECT * FROM "____user" WHERE "id" = ? LIMIT 1`,
		id)
	if err != nil {
		err = errors.Wrapf(err, "get project user failed")
		return
	}
	return
}

// GetProjectUsers batch fetch user info by ids.
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
		err = errors.Wrapf(err, "get project users failed")
	}

	return
}

// GetProjectUserList provide search and paging of project user list.
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
		err = errors.Wrapf(err, "get user list total count failed")
		return
	}

	_, err = db.Select(&users, sql, args...)
	if err != nil {
		err = errors.Wrapf(err, "get user list failed")
		return
	}

	return
}

// EnsureProjectUser add/update user info to project database.s
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
		err = errors.New("user does not exists, sign up disabled")
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

	// update user info
	if exists {
		_, err = db.Update(u)
		if err != nil {
			err = errors.Wrapf(err, "update user info failed")
		}
	} else {
		err = db.Insert(u)
		if err != nil {
			err = errors.Wrapf(err, "add new user failed")
		}
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

// UpdateProjectUser populate project user object to database.
func UpdateProjectUser(db *gorp.DbMap, u *ProjectUser) (err error) {
	_, err = db.Update(u)
	if err != nil {
		err = errors.Wrapf(err, "update project user failed")
	}
	return
}
