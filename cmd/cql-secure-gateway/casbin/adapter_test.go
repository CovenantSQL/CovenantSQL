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
	"testing"

	"github.com/casbin/casbin"
	"github.com/pkg/errors"
	. "github.com/smartystreets/goconvey/convey"
)

func TestCasbin(t *testing.T) {
	Convey("test casbin", t, func(c C) {
		testEnforce := func(e *casbin.Enforcer, sub string, obj interface{}, act string, res bool) {
			c.So(e.Enforce(sub, obj, act), ShouldEqual, res)
		}

		cfg := &Config{
			Policies: []Policy{
				{
					User:   "alice",
					Field:  "db1.tbl1.col1",
					Action: "read",
				},
				{
					User:   "bob",
					Field:  "db1.tbl2.col1",
					Action: "write",
				},
				{
					User:   "data_group_admin",
					Field:  "field_group",
					Action: "write",
				},
			},
			UserGroup: map[string][]string{
				"data_group_admin": {
					"alice",
				},
			},
			FieldGroup: map[string][]string{
				"field_group": {
					"db1.tbl1.col1",
					"db1.tbl2.col1",
				},
			},
		}

		e, err := NewCasbin(cfg)
		So(err, ShouldBeNil)

		testEnforce(e, "alice", "db1.tbl1.col1", "read", true)
		testEnforce(e, "alice", "db1.tbl1.col1", "write", true)
		testEnforce(e, "alice", "db1.tbl2.col1", "read", false)
		testEnforce(e, "alice", "db1.tbl2.col1", "write", true)
		testEnforce(e, "bob", "db1.tbl1.col1", "read", false)
		testEnforce(e, "bob", "db1.tbl1.col1", "write", false)
		testEnforce(e, "bob", "db1.tbl2.col1", "read", false)
		testEnforce(e, "bob", "db1.tbl2.col1", "write", true)
	})
	Convey("test invalid field config", t, func() {
		// nil config
		_, err := NewCasbin(nil)
		So(err, ShouldNotBeNil)
		So(errors.Cause(err), ShouldEqual, ErrEmptyConfigItem)

		// correct control group
		cfg := &Config{
			Policies: []Policy{
				{
					User:   "user1",
					Field:  "db1.tbl1.col1",
					Action: "read",
				},
			},
		}
		_, err = NewCasbin(cfg)
		So(err, ShouldBeNil)

		// empty user field
		cfg = &Config{
			Policies: []Policy{
				{
					User:   "",
					Field:  "db1.tbl1.col1",
					Action: "read",
				},
			},
		}

		_, err = NewCasbin(cfg)
		So(err, ShouldNotBeNil)
		So(errors.Cause(err), ShouldEqual, ErrEmptyConfigItem)

		// empty field
		cfg = &Config{
			Policies: []Policy{
				{
					User:   "user1",
					Field:  "",
					Action: "read",
				},
			},
		}
		_, err = NewCasbin(cfg)
		So(err, ShouldNotBeNil)
		So(errors.Cause(err), ShouldEqual, ErrEmptyConfigItem)

		// empty action
		cfg = &Config{
			Policies: []Policy{
				{
					User:   "user1",
					Field:  "db1.tbl1.col1",
					Action: "",
				},
			},
		}
		_, err = NewCasbin(cfg)
		So(err, ShouldNotBeNil)
		So(errors.Cause(err), ShouldEqual, ErrEmptyConfigItem)

		// invalid field, not enough field parts
		cfg = &Config{
			Policies: []Policy{
				{
					User:   "user1",
					Field:  "db1",
					Action: "read",
				},
			},
		}
		_, err = NewCasbin(cfg)
		So(err, ShouldNotBeNil)
		So(errors.Cause(err), ShouldEqual, ErrInvalidField)

		// invalid field, more than required field parts
		cfg = &Config{
			Policies: []Policy{
				{
					User:   "user1",
					Field:  "db1.tbl1.col1.what?",
					Action: "read",
				},
			},
		}
		_, err = NewCasbin(cfg)
		So(err, ShouldNotBeNil)
		So(errors.Cause(err), ShouldEqual, ErrInvalidField)

		// invalid field, contains empty field part
		cfg = &Config{
			Policies: []Policy{
				{
					User:   "user1",
					Field:  "db1..col1",
					Action: "read",
				},
			},
		}
		_, err = NewCasbin(cfg)
		So(err, ShouldNotBeNil)
		So(errors.Cause(err), ShouldEqual, ErrInvalidField)

		// correct control group with user groups
		cfg = &Config{
			Policies: []Policy{
				{
					User:   "a",
					Field:  "fg1",
					Action: "read",
				},
			},
			UserGroup: map[string][]string{
				"a": {
					"user1",
				},
				"b": {
					"user1",
				},
			},
			FieldGroup: map[string][]string{
				"fg1": {
					"db1.tbl1.col1",
				},
				"fg2": {
					"db1.tbl1.col1",
				},
			},
		}
		_, err = NewCasbin(cfg)
		So(err, ShouldBeNil)

		// empty group name in user groups
		cfg = &Config{
			Policies: []Policy{
				{
					User:   "a",
					Field:  "fg1",
					Action: "read",
				},
			},
			UserGroup: map[string][]string{
				"a": {
					"user1",
				},
				"": {
					"user1",
				},
			},
			FieldGroup: map[string][]string{
				"fg1": {
					"db1.tbl1.col1",
				},
				"fg2": {
					"db1.tbl1.col1",
				},
			},
		}
		_, err = NewCasbin(cfg)
		So(err, ShouldNotBeNil)
		So(errors.Cause(err), ShouldEqual, ErrEmptyConfigItem)

		// empty group name in field groups
		cfg = &Config{
			Policies: []Policy{
				{
					User:   "a",
					Field:  "fg1",
					Action: "read",
				},
			},
			UserGroup: map[string][]string{
				"a": {
					"user1",
				},
				"b": {
					"user1",
				},
			},
			FieldGroup: map[string][]string{
				"fg1": {
					"db1.tbl1.col1",
				},
				"": {
					"db1.tbl1.col1",
				},
			},
		}
		_, err = NewCasbin(cfg)
		So(err, ShouldNotBeNil)
		So(errors.Cause(err), ShouldEqual, ErrEmptyConfigItem)

		// empty user name in user groups
		cfg = &Config{
			Policies: []Policy{
				{
					User:   "a",
					Field:  "fg1",
					Action: "read",
				},
			},
			UserGroup: map[string][]string{
				"a": {
					"user1",
					"",
				},
				"b": {
					"user1",
				},
			},
			FieldGroup: map[string][]string{
				"fg1": {
					"db1.tbl1.col1",
				},
				"fg2": {
					"db1.tbl1.col1",
				},
			},
		}
		_, err = NewCasbin(cfg)
		So(err, ShouldNotBeNil)
		So(errors.Cause(err), ShouldEqual, ErrEmptyConfigItem)

		// empty field name in field groups
		cfg = &Config{
			Policies: []Policy{
				{
					User:   "a",
					Field:  "fg1",
					Action: "read",
				},
			},
			UserGroup: map[string][]string{
				"a": {
					"user1",
				},
				"b": {
					"user1",
				},
			},
			FieldGroup: map[string][]string{
				"fg1": {
					"db1.tbl1.col1",
					"",
				},
				"fg2": {
					"db1.tbl1.col1",
				},
			},
		}
		_, err = NewCasbin(cfg)
		So(err, ShouldNotBeNil)
		So(errors.Cause(err), ShouldEqual, ErrEmptyConfigItem)

		// unknown field group
		cfg = &Config{
			Policies: []Policy{
				{
					User:   "a",
					Field:  "fg3",
					Action: "read",
				},
			},
			UserGroup: map[string][]string{
				"a": {
					"user1",
				},
				"b": {
					"user1",
				},
			},
			FieldGroup: map[string][]string{
				"fg1": {
					"db1.tbl1.col1",
				},
				"fg2": {
					"db1.tbl1.col1",
				},
			},
		}
		_, err = NewCasbin(cfg)
		So(err, ShouldNotBeNil)
		So(errors.Cause(err), ShouldEqual, ErrInvalidField)

		// invalid field in field group
		cfg = &Config{
			Policies: []Policy{
				{
					User:   "a",
					Field:  "fg1",
					Action: "read",
				},
			},
			UserGroup: map[string][]string{
				"a": {
					"user1",
				},
				"b": {
					"user1",
				},
			},
			FieldGroup: map[string][]string{
				"fg1": {
					"db1.tbl1.col1",
					"db1",
				},
				"fg2": {
					"db1.tbl1.col1",
				},
			},
		}
		_, err = NewCasbin(cfg)
		So(err, ShouldNotBeNil)
		So(errors.Cause(err), ShouldEqual, ErrInvalidField)
	})
	Convey("unsupported features", t, func() {
		a, err := NewAdapter(&Config{})
		So(err, ShouldBeNil)
		So(a, ShouldNotBeNil)
		// call unsupported fields
		err = a.SavePolicy(nil)
		So(err, ShouldEqual, ErrNotSupported)
		err = a.AddPolicy("", "", []string{})
		So(err, ShouldEqual, ErrNotSupported)
		err = a.RemovePolicy("", "", []string{})
		So(err, ShouldEqual, ErrNotSupported)
		err = a.RemoveFilteredPolicy("", "", 0)
		So(err, ShouldEqual, ErrNotSupported)
	})
}
