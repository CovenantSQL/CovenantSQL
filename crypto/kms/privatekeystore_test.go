/*
 * Copyright 2018 The ThunderDB Authors.
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

package kms

import (
	"testing"

	"os"

	. "github.com/smartystreets/goconvey/convey"
)

const masterKeyPath = "./.testmasterkey"

func TestLoadMasterKey(t *testing.T) {
	Convey("save and load", t, func() {
		mk, err := GenerateMasterKey()
		So(mk, ShouldNotBeNil)
		So(err, ShouldBeNil)
		err = SaveMasterKey(masterKeyPath, mk)
		So(err, ShouldBeNil)

		lk, err := LoadMasterKey(masterKeyPath)
		So(err, ShouldBeNil)
		So(string(mk.Serialize()), ShouldEqual, string(lk.Serialize()))
		os.Remove(masterKeyPath)
	})
	Convey("load error", t, func() {
		lk, err := LoadMasterKey("/path/not/exist")
		So(err, ShouldNotBeNil)
		So(lk, ShouldBeNil)
	})
	Convey("empty key file", t, func() {
		os.Create("./.empty")
		lk, err := LoadMasterKey("./.empty")
		So(err, ShouldEqual, ErrNotKeyFile)
		So(lk, ShouldBeNil)
		os.Remove("./.empty")
	})
	Convey("not key file", t, func() {
		lk, err := LoadMasterKey("doc.go")
		So(err, ShouldEqual, ErrNotKeyFile)
		So(lk, ShouldBeNil)
	})
}
