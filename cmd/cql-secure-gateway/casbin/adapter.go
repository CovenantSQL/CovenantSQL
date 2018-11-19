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

// Policy defines single resource policy.
type Policy struct {
	User   string `yaml:"User"`
	Field  string `yaml:"Field"`
	Action string `yaml:"Action"`
}

// UserGroup defines single user group relation.
type UserGroup map[string][]string

// FieldGroup defines single field group relation.
type FieldGroup map[string][]string

// Config defines the rules config for casbin adapter.
type Config struct {
	Policies   []Policy   `yaml:"Policies"`
	UserGroup  UserGroup  `yaml:"UserGroups"`
	FieldGroup FieldGroup `yaml:"FieldGroups"`
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
		if p.User == "" || p.Field == "" || p.Action == "" {
			err = errors.Wrapf(ErrEmptyConfigItem, "%#v contains empty field", p)
			return
		}

		if _, exists := a.cfg.FieldGroup[p.Field]; !exists {
			// must be a valid field
			if err = a.validateField(p.Field); err != nil {
				return
			}
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

	// validate fields
	for fieldGroup, fields := range a.cfg.FieldGroup {
		if fieldGroup == "" {
			err = errors.Wrapf(ErrEmptyConfigItem, "field group name should not be empty")
			return
		}

		// should all be valid field
		for _, f := range fields {
			if f == "" {
				err = errors.Wrapf(ErrEmptyConfigItem, "field name should not be empty")
				return
			}

			if err = a.validateField(f); err != nil {
				return
			}
		}
	}

	return
}

func (a *Adapter) validateField(f string) (err error) {
	parts := strings.Split(f, ".")

	if len(parts) != 3 {
		err = errors.Wrapf(ErrInvalidField,
			"invalid field: %s, should be in form \"<db>.<table>.<column>\", wildcard is supported", f)
		return
	}

	for _, p := range parts {
		p = strings.TrimSpace(p)

		if p == "" {
			err = errors.Wrapf(ErrInvalidField,
				"invalid field: %s, <db>/<table>/<column> parts should not be blanked", f)
			return
		}
	}

	return
}

// LoadPolicy implements casbin persist.Adapter interface.
func (a *Adapter) LoadPolicy(model model.Model) (err error) {
	// build casbin csv policy and load

	// load policy
	for _, p := range a.cfg.Policies {
		line := fmt.Sprintf("p, %s, %s, %s", p.User, p.Field, p.Action)
		persist.LoadPolicyLine(line, model)
	}

	// load user groups
	for userGroup, users := range a.cfg.UserGroup {
		for _, user := range users {
			line := fmt.Sprintf("g, %s, %s", user, userGroup)
			persist.LoadPolicyLine(line, model)
		}
	}

	// load field groups
	for fieldGroup, fields := range a.cfg.FieldGroup {
		for _, field := range fields {
			line := fmt.Sprintf("g2, %s, %s", field, fieldGroup)
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
