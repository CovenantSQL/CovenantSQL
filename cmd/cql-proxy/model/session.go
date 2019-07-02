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
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	gorp "gopkg.in/gorp.v2"
)

// Session defines the session object for user/admin.
type Session struct {
	ID       string `db:"id"`
	RawStore []byte `db:"store"`
	Store    gin.H  `db:"-"`
	Created  int64  `db:"created"`
	Expire   int64  `db:"expire"`
}

// Get returns the object with specified key stored in session object.
func (s *Session) Get(key string) (value interface{}, exists bool) {
	value, exists = s.Store[key]
	return
}

// GetInt returns the int value with specified key stored in session object.
func (s *Session) GetInt(key string) (value int64, exists bool) {
	rv, exists := s.Get(key)
	if !exists {
		return
	}

	switch v := rv.(type) {
	case int:
		value = int64(v)
	case int8:
		value = int64(v)
	case int16:
		value = int64(v)
	case int32:
		value = int64(v)
	case int64:
		value = v
	case uint:
		value = int64(v)
	case uint8:
		value = int64(v)
	case uint16:
		value = int64(v)
	case uint32:
		value = int64(v)
	case uint64:
		value = int64(v)
	case float32:
		value = int64(v)
	case float64:
		value = int64(v)
	case bool:
		if v {
			value = 1
		} else {
			value = 0
		}
	case string:
		var err error
		value, err = strconv.ParseInt(v, 10, 64)
		if err != nil {
			exists = false
		}
		return
	default:
		exists = false
	}

	return
}

// GetUint returns the unsigned int value with specified key stored in session object.
func (s *Session) GetUint(key string) (value uint64, exists bool) {
	rv, exists := s.Get(key)
	if !exists {
		return
	}

	switch v := rv.(type) {
	case int:
		value = uint64(v)
	case int8:
		value = uint64(v)
	case int16:
		value = uint64(v)
	case int32:
		value = uint64(v)
	case int64:
		value = uint64(v)
	case uint:
		value = uint64(v)
	case uint8:
		value = uint64(v)
	case uint16:
		value = uint64(v)
	case uint32:
		value = uint64(v)
	case uint64:
		value = uint64(v)
	case float32:
		value = uint64(v)
	case float64:
		value = uint64(v)
	case bool:
		if v {
			value = 1
		} else {
			value = 0
		}
	case string:
		var err error
		value, err = strconv.ParseUint(v, 10, 64)
		if err != nil {
			exists = false
		}
	default:
		exists = false
	}

	return
}

// GetString returns the string value with specified key store in session object.
func (s *Session) GetString(key string) (value string, exists bool) {
	rv, exists := s.Get(key)
	if !exists {
		return
	}

	value = fmt.Sprint(rv)
	return
}

// GetBool returns the bool value with specified key in session object.
func (s *Session) GetBool(key string) (value bool, exists bool) {
	rv, exists := s.Get(key)
	if !exists {
		return
	}

	switch v := rv.(type) {
	case int:
		value = v != 0
	case int8:
		value = v != 0
	case int16:
		value = v != 0
	case int32:
		value = v != 0
	case int64:
		value = v != 0
	case uint:
		value = v != 0
	case uint8:
		value = v != 0
	case uint16:
		value = v != 0
	case uint32:
		value = v != 0
	case uint64:
		value = v != 0
	case float32:
		value = math.Abs(float64(v)) > 0
	case float64:
		value = math.Abs(v) > 0
	case bool:
		value = v
	case string:
		value = len(v) > 0
	default:
		rrv := reflect.ValueOf(rv)

		switch rrv.Kind() {
		case reflect.Array, reflect.Slice, reflect.Map, reflect.Chan:
			value = rrv.Len() > 0
		default:
			exists = false
		}
	}

	return
}

// MustGet returns the value with specified key in session object.
func (s *Session) MustGet(key string) (value interface{}) {
	value, _ = s.Get(key)
	return
}

// MustGetInt returns the int value with specified key in session object.
func (s *Session) MustGetInt(key string) (value int64) {
	value, _ = s.GetInt(key)
	return
}

// MustGetUint returns the unsigned int value with specified key in session object.
func (s *Session) MustGetUint(key string) (value uint64) {
	value, _ = s.GetUint(key)
	return
}

// MustGetString returns the string value with specified key in session object.
func (s *Session) MustGetString(key string) (value string) {
	value, _ = s.GetString(key)
	return
}

// MustGetBool returns the bool value with specified key in session object.
func (s *Session) MustGetBool(key string) (value bool) {
	value, _ = s.GetBool(key)
	return
}

// Set stores value to session object.
func (s *Session) Set(key string, value interface{}) {
	s.Store[key] = value
}

// Delete removes value from session object.
func (s *Session) Delete(key string) {
	delete(s.Store, key)
}

// Serialize marshal session object storage to bytes form.
func (s *Session) Serialize() (err error) {
	s.RawStore, err = json.Marshal(s.Store)
	return
}

// Deserialize unmarshal session object from bytes form.
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

// NewEmptySession returns new session object with empty storage.
func NewEmptySession(c *gin.Context) (s *Session) {
	s = &Session{
		Store: gin.H{},
	}

	c.Set("session", s)

	return
}

// NewSession created and save new session to database.
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
		err = errors.Wrapf(err, "new session failed")
		return
	}

	c.Set("session", s)

	return
}

// GetSession returns session object of specified session id.
func GetSession(c *gin.Context, id string) (s *Session, err error) {
	err = GetDB(c).SelectOne(&s,
		`SELECT * FROM "session" WHERE "id" = ? LIMIT 1`, id)
	if err != nil {
		err = errors.Wrapf(err, "get session failed")
		return
	}
	err = s.Deserialize()
	if err != nil {
		err = errors.Wrapf(err, "decode session failed")
		return
	}

	c.Set("session", s)

	return
}

// SaveSession saves session object to database.
func SaveSession(c *gin.Context, s *Session, expire int64) (r *Session, err error) {
	if s == nil {
		return NewSession(c, expire)
	}

	r = s
	err = r.Serialize()
	if err != nil {
		err = errors.Wrapf(err, "encode session failed")
		return
	}

	_, err = GetDB(c).Update(r)
	if err != nil {
		err = errors.Wrapf(err, "update session failed")
	}
	return
}

// ExpireSessions expires session object in database.
func ExpireSessions(db *gorp.DbMap) (expireCount int64, err error) {
	var sessions []*Session
	_, err = db.Select(&sessions, `SELECT * FROM "session" WHERE "expire" <= ?`,
		time.Now().Unix())
	if err != nil {
		err = errors.Wrapf(err, "get sessions to expire failed")
		return
	}

	var args []interface{}

	for _, s := range sessions {
		args = append(args, s)
	}

	expireCount, err = db.Delete(args...)
	if err != nil {
		err = errors.Wrapf(err, "remove expired sessions failed")
	}

	return
}
