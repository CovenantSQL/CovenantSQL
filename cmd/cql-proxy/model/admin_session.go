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
	"github.com/gin-gonic/gin"
	"github.com/satori/go.uuid"
	"gopkg.in/gorp.v1"
	"time"
)

type AdminSession struct {
	ID       string                 `db:"id"`
	RawStore []byte                 `db:"store"`
	Store    map[string]interface{} `db:"-"`
	Created  int64                  `db:"created"`
	Expire   int64                  `db:"expire"`
}

func (s *AdminSession) Get(key string) (value interface{}, exists bool) {
	value, exists = s.Store[key]
	return
}

func (s *AdminSession) MustGet(key string) (value interface{}) {
	value, _ = s.Get(key)
	return
}

func (s *AdminSession) Set(key string, value interface{}) {
	s.Store[key] = value
}

func (s *AdminSession) Delete(key string) {
	delete(s.Store, key)
}

func (s *AdminSession) Serialize() (err error) {
	s.RawStore, err = json.Marshal(s.Store)
	return
}

func (s *AdminSession) Deserialize() (err error) {
	err = json.Unmarshal(s.RawStore, &s.Store)
	if err != nil {
		return
	}
	if s.Store == nil {
		s.Store = make(map[string]interface{})
	}
	return
}

func NewEmptySession(c *gin.Context) (s *AdminSession) {
	s = &AdminSession{
		Store: make(map[string]interface{}),
	}

	c.Set("session", s)

	return
}

func NewAdminSession(c *gin.Context, expire int64) (s *AdminSession, err error) {
	id := uuid.Must(uuid.NewV4()).String()
	now := time.Now().Unix()
	s = &AdminSession{
		ID:      id,
		Created: now,
		Expire:  now + expire,
		Store:   make(map[string]interface{}),
	}
	err = c.MustGet(keyDB).(*gorp.DbMap).Insert(s)
	if err != nil {
		return
	}

	c.Set("session", s)

	return
}

func GetAdminSession(c *gin.Context, id string) (s *AdminSession, err error) {
	err = c.MustGet(keyDB).(*gorp.DbMap).SelectOne(&s,
		`SELECT * FROM "admin_session" WHERE "id" = ? LIMIT 1`, id)
	if err != nil {
		return
	}
	err = s.Deserialize()
	if err != nil {
		return
	}

	c.Set("session", s)

	return
}

func SaveAdminSession(c *gin.Context, s *AdminSession, expire int64) (r *AdminSession, err error) {
	if s == nil {
		return NewAdminSession(c, expire)
	}

	r = s
	err = r.Serialize()
	if err != nil {
		return
	}

	r.Expire = time.Now().Unix() + expire

	_, err = c.MustGet(keyDB).(*gorp.DbMap).Update(r)
	return
}

func init() {
	RegisterModel("admin_session", AdminSession{}, "ID", false)
}
