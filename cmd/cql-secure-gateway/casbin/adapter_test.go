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
	"gopkg.in/yaml.v2"
)

func mustNewField(c C, fieldStr string) (f Field) {
	var (
		err error
		pf  *Field
	)
	pf, err = NewFieldFromString(fieldStr)
	f = *pf
	c.So(err, ShouldBeNil)
	return
}

func TestNewField(t *testing.T) {
	Convey("test new field", t, func() {
		var (
			f   *Field
			err error
		)
		f, err = NewFieldFromString("db1.tbl1.col1")
		So(f, ShouldNotBeNil)
		So(err, ShouldBeNil)
		So(f.Database, ShouldEqual, "db1")
		So(f.Table, ShouldEqual, "tbl1")
		So(f.Column, ShouldEqual, "col1")
		So(f.String(), ShouldEqual, "db1.tbl1.col1")

		// test marshal unmarshal
		var yamlBytes []byte
		yamlBytes, err = yaml.Marshal(f)
		So(err, ShouldBeNil)
		So(yamlBytes, ShouldNotBeEmpty)

		var uField *Field
		err = yaml.Unmarshal(yamlBytes, &uField)
		So(err, ShouldBeNil)
		So(uField, ShouldResemble, f)

		// test load invalid field
		_, err = NewFieldFromString("")
		So(err, ShouldNotBeNil)

		_, err = NewFieldFromString("  ")
		So(err, ShouldNotBeNil)

		_, err = NewFieldFromString("db1")
		So(err, ShouldNotBeNil)

		_, err = NewFieldFromString("db1.tbl1")
		So(err, ShouldNotBeNil)

		_, err = NewFieldFromString("db1..col1")
		So(err, ShouldNotBeNil)

		_, err = NewFieldFromString("db1.col1.")
		So(err, ShouldNotBeNil)

		_, err = NewFieldFromString(".tbl1.col1")
		So(err, ShouldNotBeNil)

		// trivial cases
		f, err = NewFieldFromString("a . b . c")
		So(err, ShouldBeNil)
		So(f.Database, ShouldEqual, "a")
		So(f.Table, ShouldEqual, "b")
		So(f.Column, ShouldEqual, "c")

		// invalid marshal
		yamlBytes, err = yaml.Marshal(map[string]string{"a": "1"})
		So(err, ShouldBeNil)
		So(yamlBytes, ShouldNotBeEmpty)
		err = yaml.Unmarshal(yamlBytes, &uField)
		So(err, ShouldNotBeNil)
	})
}

func TestCasbin(t *testing.T) {
	Convey("test casbin", t, func(c C) {
		testEnforce := func(e *casbin.Enforcer, sub string, obj interface{}, act string, res bool) {
			actual, _ := e.EnforceSafe(sub, obj, act)
			c.So(actual, ShouldEqual, res)
		}

		cfg := &Config{
			Policies: []Policy{
				{
					User:   "alice",
					Field:  mustNewField(c, "db1.tbl1.*"),
					Action: ReadAction,
				},
				{
					User:   "bob",
					Field:  mustNewField(c, "db1.tbl2.col1"),
					Action: WriteAction,
				},
				{
					User:   "data_group_admin",
					Field:  mustNewField(c, "db1.tbl1.col2"),
					Action: WriteAction,
				},
				{
					User:   "data_group_admin",
					Field:  mustNewField(c, "db1.tbl2.col1"),
					Action: WriteAction,
				},
			},
			UserGroup: map[string][]string{
				"data_group_admin": {
					"alice",
				},
			},
		}

		e, err := NewCasbin(cfg)
		So(err, ShouldBeNil)

		testEnforce(e, "alice", "db1.tbl1.col1", ReadAction, true)
		testEnforce(e, "alice", "db1.tbl1.col1", WriteAction, false)
		testEnforce(e, "alice", "db1.tbl2.col1", ReadAction, true)
		testEnforce(e, "alice", "db1.tbl2.col1", WriteAction, true)
		testEnforce(e, "alice", "db1.tbl1.col2", ReadAction, true)
		testEnforce(e, "alice", "db1.tbl1.col2", WriteAction, true)
		testEnforce(e, "alice", "db1.tbl1.col3", ReadAction, true)
		testEnforce(e, "alice", "db1.tbl1.col3", WriteAction, false)
		testEnforce(e, "bob", "db1.tbl1.col1", ReadAction, false)
		testEnforce(e, "bob", "db1.tbl1.col1", WriteAction, false)
		testEnforce(e, "bob", "db1.tbl2.col1", ReadAction, true)
		testEnforce(e, "bob", "db1.tbl2.col1", WriteAction, true)
		testEnforce(e, "bob", "db1", ReadAction, false)
		testEnforce(e, "bob", "db1.tbl1.col1", ReadAction, false)
	})
	Convey("test invalid config", t, func(c C) {
		// nil config
		_, err := NewCasbin(nil)
		So(err, ShouldNotBeNil)
		So(errors.Cause(err), ShouldEqual, ErrEmptyConfigItem)

		// correct control group
		cfg := &Config{
			Policies: []Policy{
				{
					User:   "user1",
					Field:  mustNewField(c, "db1.tbl1.col1"),
					Action: ReadAction,
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
					Field:  mustNewField(c, "db1.tbl1.col1"),
					Action: ReadAction,
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
					Field:  mustNewField(c, "db1.tbl1.col1"),
					Action: "",
				},
			},
		}
		_, err = NewCasbin(cfg)
		So(err, ShouldNotBeNil)
		So(errors.Cause(err), ShouldEqual, ErrEmptyConfigItem)

		// correct control group with user groups
		cfg = &Config{
			Policies: []Policy{
				{
					User:   "a",
					Field:  mustNewField(c, "db1.tbl1.col1"),
					Action: ReadAction,
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
		}
		_, err = NewCasbin(cfg)
		So(err, ShouldBeNil)

		// empty group name in user groups
		cfg = &Config{
			Policies: []Policy{
				{
					User:   "a",
					Field:  mustNewField(c, "db1.tbl1.col1"),
					Action: ReadAction,
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
		}
		_, err = NewCasbin(cfg)
		So(err, ShouldNotBeNil)
		So(errors.Cause(err), ShouldEqual, ErrEmptyConfigItem)

		// empty user name in user groups
		cfg = &Config{
			Policies: []Policy{
				{
					User:   "a",
					Field:  mustNewField(c, "db1.tbl1.col1"),
					Action: ReadAction,
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
		}
		_, err = NewCasbin(cfg)
		So(err, ShouldNotBeNil)
		So(errors.Cause(err), ShouldEqual, ErrEmptyConfigItem)
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

func TestField(t *testing.T) {
	Convey("test field", t, func(c C) {
		// normal field matches wildcard field
		f := mustNewField(c, "db1.tbl1.col1")
		So(f.MatchesString("db1.tbl1.col1"), ShouldBeTrue)
		So(f.MatchesString("db1.tbl1.c*1"), ShouldBeTrue)
		So(f.MatchesString("db*"), ShouldBeFalse)
		So(f.MatchesField(nil), ShouldBeFalse)
		So((*Field)(nil).MatchesField(&f), ShouldBeFalse)
		So(f.MatchesString("db1.tbl2.col1"), ShouldBeFalse)
		// reverse that
		f = mustNewField(c, "db1.tb*.col1")
		So(f.MatchesString("db1.tb1.col1"), ShouldBeTrue)
		So(f.MatchesString("db1.tb2.col1"), ShouldBeTrue)
		So(f.MatchesString("db2.tb1.col1"), ShouldBeFalse)
		So(f.MatchesString("db1.tb*.col1"), ShouldBeTrue)
		So(f.MatchesString("db1.*tb.col1"), ShouldBeFalse)
	})
}
