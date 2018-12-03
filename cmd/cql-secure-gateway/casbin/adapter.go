/*
 * Copyright 2018 The CovenantSQL Authors.
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

package casbin

import (
	"fmt"
	"strings"

	"github.com/casbin/casbin/model"
	"github.com/casbin/casbin/persist"
	"github.com/pkg/errors"
)

var (
	// ErrNotSupported defines error for policy persistence adapter not supported feature.
	ErrNotSupported = errors.New("not supported")
	// ErrInvalidField defines error for invalid field description.
	ErrInvalidField = errors.New("invalid field description")
	// ErrEmptyConfigItem defines empty config item in policies.
	ErrEmptyConfigItem = errors.New("empty config item")
)

// Field defines a single database column.
type Field struct {
	Database string
	Table    string
	Column   string
}

// NewField creates new field object instance.
func NewField(db, table, col string) (f *Field, err error) {
	db = strings.TrimSpace(db)
	table = strings.TrimSpace(table)
	col = strings.TrimSpace(col)

	for _, s := range []string{db, table, col} {
		if s == "" {
			err = errors.Wrapf(ErrInvalidField,
				"invalid field: %s, <db>/<table>/<column> parts should not be blanked", f)
			return
		}
	}

	f = &Field{
		Database: db,
		Table:    table,
		Column:   col,
	}

	return
}

// NewFieldFromString creates new field object instance from string field representation.
func NewFieldFromString(fieldStr string) (f *Field, err error) {
	f = &Field{}
	err = f.loadFromStr(fieldStr)
	return
}

// MatchesString test if string field matches.
func (f *Field) MatchesString(fieldStr string) bool {
	var (
		field *Field
		err   error
	)
	if field, err = NewFieldFromString(fieldStr); err != nil {
		// ignore error
		return false
	}
	return f.MatchesField(field)
}

// MatchesField test if field matches.
func (f *Field) MatchesField(field *Field) bool {
	if f == nil || field == nil {
		return false
	}

	var s1, s2 string
	for i := 0; i != 3; i++ {
		switch i {
		case 0:
			s1 = f.Database
			s2 = field.Database
		case 1:
			s1 = f.Table
			s2 = field.Table
		case 2:
			s1 = f.Column
			s2 = field.Column
		}

		if !tryGlobMatch(s1, s2) {
			return false
		}
	}

	return true
}

// IsWildcard returns whether the field is a wildcard or plain.
func (f *Field) IsWildcard() bool {
	if f == nil {
		return false
	}

	return strings.ContainsRune(f.Database, '*') ||
		strings.ContainsRune(f.Table, '*') ||
		strings.ContainsRune(f.Column, '*')
}

// UnmarshalYAML implements yaml.Unmarshaler interface.
func (f *Field) UnmarshalYAML(unmarshal func(interface{}) error) (err error) {
	var fieldStr string

	if err = unmarshal(&fieldStr); err != nil {
		return
	}

	return f.loadFromStr(fieldStr)
}

func (f *Field) loadFromStr(fieldStr string) (err error) {
	var parts []string

	// split field into parts
	parts = strings.Split(fieldStr, ".")
	if len(parts) != 3 {
		err = errors.Wrapf(ErrInvalidField,
			"invalid field: %s, should be in form \"<db>.<table>.<column>\", wildcard is supported", f)
		return
	}

	for i, p := range parts {
		p = strings.TrimSpace(p)

		if p == "" {
			err = errors.Wrapf(ErrInvalidField,
				"invalid field: %s, <db>/<table>/<column> parts should not be blanked", f)
			return
		}

		switch i {
		case 0:
			f.Database = p
		case 1:
			f.Table = p
		case 2:
			f.Column = p
		}
	}

	return
}

// MarshalYAML implements yaml.Marshaler interface.
func (f Field) MarshalYAML() (interface{}, error) {
	return f.String(), nil
}

// String returns the string representation of field.
func (f *Field) String() string {
	return strings.ToLower(fmt.Sprintf("%s.%s.%s", f.Database, f.Table, f.Column))
}

// Policy defines single resource policy.
type Policy struct {
	User   string `yaml:"User"`
	Field  Field  `yaml:"Field"`
	Action string `yaml:"Action"`
}

// UserGroup defines single user group relation.
type UserGroup map[string][]string

// Config defines the rules config for casbin adapter.
type Config struct {
	Policies  []Policy  `yaml:"Policies"`
	UserGroup UserGroup `yaml:"UserGroups"`
}

// Adapter defines a adapter for casbin to access rules.
type Adapter struct {
	cfg *Config
}

// NewAdapter returns an adapter for casbin enforcer rules.
func NewAdapter(cfg *Config) (a *Adapter, err error) {
	if cfg == nil {
		err = errors.Wrapf(ErrEmptyConfigItem, "nil config")
		return
	}

	// valid config
	a = &Adapter{
		cfg: cfg,
	}

	if err = a.validate(); err != nil {
		a = nil
	}

	return
}

func (a *Adapter) validate() (err error) {
	// validate non-group fields
	for _, p := range a.cfg.Policies {
		if p.User == "" || p.Action == "" {
			err = errors.Wrapf(ErrEmptyConfigItem, "%#v contains empty field", p)
			return
		}
	}

	for userGroup, users := range a.cfg.UserGroup {
		if userGroup == "" {
			err = errors.Wrapf(ErrEmptyConfigItem, "user group name should not be empty")
			return
		}

		for _, u := range users {
			if u == "" {
				err = errors.Wrapf(ErrEmptyConfigItem, "user name should not be empty in group: %s", userGroup)
				return
			}
		}
	}

	return
}

// LoadPolicy implements casbin persist.Adapter interface.
func (a *Adapter) LoadPolicy(model model.Model) (err error) {
	// build casbin csv policy and load

	// load policy
	for _, p := range a.cfg.Policies {
		line := fmt.Sprintf("p, %s, %s, %s", p.User, p.Field.String(), p.Action)
		persist.LoadPolicyLine(line, model)
	}

	// load user groups
	for userGroup, users := range a.cfg.UserGroup {
		for _, user := range users {
			line := fmt.Sprintf("g, %s, %s", user, userGroup)
			persist.LoadPolicyLine(line, model)
		}
	}

	return
}

// SavePolicy mocks casbin persist.Adapter interface.
func (*Adapter) SavePolicy(model model.Model) error {
	return ErrNotSupported
}

// AddPolicy mocks casbin persist.Adapter interface.
func (*Adapter) AddPolicy(sec string, ptype string, rule []string) error {
	return ErrNotSupported
}

// RemovePolicy mocks casbin persist.Adapter interface.
func (*Adapter) RemovePolicy(sec string, ptype string, rule []string) error {
	return ErrNotSupported
}

// RemoveFilteredPolicy mocks casbin persist.Adapter interface.
func (*Adapter) RemoveFilteredPolicy(sec string, ptype string, fieldIndex int, fieldValues ...string) error {
	return ErrNotSupported
}
