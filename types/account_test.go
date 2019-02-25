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

package types

import (
	"encoding/json"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestUserPermissionFromRole(t *testing.T) {
	Convey("test marshal/unmarshal json", t, func() {
		jsonBytes, err := json.Marshal(Read)
		So(err, ShouldBeNil)
		So(jsonBytes, ShouldResemble, []byte(`"Read"`))
		var r UserPermissionRole
		So(r, ShouldEqual, Void)
		err = json.Unmarshal([]byte(`"Write"`), &r)
		So(err, ShouldBeNil)
		So(r, ShouldEqual, Write)
		err = r.UnmarshalJSON([]byte(`"Read,Write"`))
		So(err, ShouldBeNil)
		So(r, ShouldEqual, ReadWrite)
	})
	Convey("test string/from string", t, func() {
		var r UserPermissionRole
		So(r, ShouldEqual, Void)
		r.FromString(Read.String())
		So(r, ShouldEqual, Read)
		r.FromString(Void.String())
		So(r, ShouldEqual, Void)
		r.FromString(ReadWrite.String())
		So(r, ShouldEqual, ReadWrite)
		r.FromString(Admin.String())
		So(r, ShouldEqual, Admin)

		// tricky case
		trickyStr := "Write,Super"
		r.FromString(trickyStr)
		So(r, ShouldEqual, Write|Super)
		So(r.String(), ShouldEqual, trickyStr)
	})
}

func TestUserPermission(t *testing.T) {
	Convey("nil protect", t, func() {
		p := (*UserPermission)(nil)
		So(p.HasReadPermission(), ShouldBeFalse)
		So(p.HasWritePermission(), ShouldBeFalse)
		So(p.HasSuperPermission(), ShouldBeFalse)
		So(p.IsValid(), ShouldBeFalse)
		_, state := p.HasDisallowedQueryPatterns([]Query{})
		So(state, ShouldBeTrue)
	})
	Convey("has read permission", t, func() {
		So(UserPermissionFromRole(Void).HasReadPermission(), ShouldBeFalse)
		So(UserPermissionFromRole(Read).HasReadPermission(), ShouldBeTrue)
		So(UserPermissionFromRole(Write).HasReadPermission(), ShouldBeFalse)
		So(UserPermissionFromRole(ReadWrite).HasReadPermission(), ShouldBeTrue)
		So(UserPermissionFromRole(Admin).HasReadPermission(), ShouldBeTrue)
	})
	Convey("has write permission", t, func() {
		So(UserPermissionFromRole(Void).HasWritePermission(), ShouldBeFalse)
		So(UserPermissionFromRole(Read).HasWritePermission(), ShouldBeFalse)
		So(UserPermissionFromRole(Write).HasWritePermission(), ShouldBeTrue)
		So(UserPermissionFromRole(ReadWrite).HasWritePermission(), ShouldBeTrue)
		So(UserPermissionFromRole(Admin).HasWritePermission(), ShouldBeTrue)
	})
	Convey("has admin permission", t, func() {
		So(UserPermissionFromRole(Void).HasSuperPermission(), ShouldBeFalse)
		So(UserPermissionFromRole(Read).HasSuperPermission(), ShouldBeFalse)
		So(UserPermissionFromRole(Write).HasSuperPermission(), ShouldBeFalse)
		So(UserPermissionFromRole(ReadWrite).HasSuperPermission(), ShouldBeFalse)
		So(UserPermissionFromRole(Admin).HasSuperPermission(), ShouldBeTrue)
	})
	Convey("is valid", t, func() {
		So(UserPermissionFromRole(Void).IsValid(), ShouldBeFalse)
		So(UserPermissionFromRole(Read).IsValid(), ShouldBeTrue)
		So(UserPermissionFromRole(Write).IsValid(), ShouldBeTrue)
		So(UserPermissionFromRole(ReadWrite).IsValid(), ShouldBeTrue)
		So(UserPermissionFromRole(Admin).IsValid(), ShouldBeTrue)
	})
	Convey("query patterns", t, func() {
		// empty patterns limitation
		_, state := UserPermissionFromRole(Read).HasDisallowedQueryPatterns([]Query{
			{
				Pattern: "select 1",
			},
			{
				Pattern: "insert into test values(1)",
			},
		})
		So(state, ShouldBeFalse)

		// test patterns limit
		up := UserPermissionFromRole(Read)
		up.Patterns = append(up.Patterns, "select 1")
		up.Patterns = append(up.Patterns, "select 2")
		// has patterns more than limit
		_, state = up.HasDisallowedQueryPatterns([]Query{
			{
				Pattern: "select 1",
			},
			{
				Pattern: "select 3",
			},
		})
		So(state, ShouldBeTrue)
		// only has limit patterns
		_, state = up.HasDisallowedQueryPatterns([]Query{
			{
				Pattern: "select 1",
			},
		})
		So(state, ShouldBeFalse)
	})
}
