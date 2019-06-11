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
	"fmt"
	"math"
	"reflect"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	uuid "github.com/satori/go.uuid"
)

type Session struct {
	ID       string `db:"id"`
	RawStore []byte `db:"store"`
	Store    gin.H  `db:"-"`
	Created  int64  `db:"created"`
	Expire   int64  `db:"expire"`
}

func (s *Session) Get(key string) (value interface{}, exists bool) {
	value, exists = s.Store[key]
	return
}

func (s *Session) GetInt(key string) (value int64, exists bool) {
	rv, exists := s.Get(key)
	if !exists {
		return
	}

	rrv := reflect.ValueOf(rv)

	switch rrv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value = rrv.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		value = int64(rrv.Uint())
	case reflect.Float32, reflect.Float64:
		value = int64(rrv.Float())
	case reflect.Bool:
		if rrv.Bool() {
			value = 1
		} else {
			value = 0
		}
	case reflect.String:
		var err error
		value, err = strconv.ParseInt(rrv.String(), 10, 64)
		if err != nil {
			exists = false
		}
		return
	default:
		exists = false
	}

	return
}

func (s *Session) GetUint(key string) (value uint64, exists bool) {
	rv, exists := s.Get(key)
	if !exists {
		return
	}

	rrv := reflect.ValueOf(rv)

	switch rrv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value = uint64(rrv.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		value = rrv.Uint()
	case reflect.Float32, reflect.Float64:
		value = uint64(rrv.Float())
	case reflect.Bool:
		if rrv.Bool() {
			value = 1
		} else {
			value = 0
		}
	case reflect.String:
		var err error
		value, err = strconv.ParseUint(rrv.String(), 10, 64)
		if err != nil {
			exists = false
		}
	default:
		exists = false
	}

	return
}

func (s *Session) GetString(key string) (value string, exists bool) {
	rv, exists := s.Get(key)
	if !exists {
		return
	}

	value = fmt.Sprint(rv)
	return
}

func (s *Session) GetBool(key string) (value bool, exists bool) {
	rv, exists := s.Get(key)
	if !exists {
		return
	}

	rrv := reflect.ValueOf(rv)

	switch rrv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value = rrv.Int() != 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		value = rrv.Uint() != 0
	case reflect.Float32, reflect.Float64:
		value = math.Abs(rrv.Float()) > 0
	case reflect.Bool:
		value = rrv.Bool()
	case reflect.Array, reflect.Slice, reflect.Map, reflect.Chan, reflect.String:
		value = rrv.Len() > 0
	default:
		exists = false
	}

	return
}

func (s *Session) MustGet(key string) (value interface{}) {
	value, _ = s.Get(key)
	return
}

func (s *Session) MustGetInt(key string) (value int64) {
	value, _ = s.GetInt(key)
	return
}

func (s *Session) MustGetUint(key string) (value uint64) {
	value, _ = s.GetUint(key)
	return
}

func (s *Session) MustGetString(key string) (value string) {
	value, _ = s.GetString(key)
	return
}

func (s *Session) MustGetBool(key string) (value bool) {
	value, _ = s.GetBool(key)
	return
}

func (s *Session) Set(key string, value interface{}) {
	s.Store[key] = value
}

func (s *Session) Delete(key string) {
	delete(s.Store, key)
}

func (s *Session) Serialize() (err error) {
	s.RawStore, err = json.Marshal(s.Store)
	return
}

func (s *Session) Deserialize() (err error) {
	err = json.Unmarshal(s.RawStore, &s.Store)
	if err != nil {
		return
	}
	if s.Store == nil {
		s.Store = gin.H{}
	}
	return
}

func NewEmptySession(c *gin.Context) (s *Session) {
	s = &Session{
		Store: gin.H{},
	}

	c.Set("session", s)

	return
}

func NewSession(c *gin.Context, expire int64) (s *Session, err error) {
	id := uuid.Must(uuid.NewV4()).String()
	now := time.Now().Unix()
	s = &Session{
		ID:      id,
		Created: now,
		Expire:  now + expire,
		Store:   gin.H{},
	}

	err = GetDB(c).Insert(s)
	if err != nil {
		return
	}

	c.Set("session", s)

	return
}

func GetSession(c *gin.Context, id string) (s *Session, err error) {
	err = GetDB(c).SelectOne(&s,
		`SELECT * FROM "session" WHERE "id" = ? LIMIT 1`, id)
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

func SaveSession(c *gin.Context, s *Session, expire int64) (r *Session, err error) {
	if s == nil {
		return NewSession(c, expire)
	}

	r = s
	err = r.Serialize()
	if err != nil {
		return
	}

	r.Expire = time.Now().Unix() + expire

	_, err = GetDB(c).Update(r)
	return
}
