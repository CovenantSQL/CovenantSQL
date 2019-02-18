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

package utils

import (
	"io/ioutil"
	"os"
	"os/user"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestCopyFile(t *testing.T) {
	Convey("copy file", t, func() {
		bytes := []byte("abc")
		defer os.Remove("testcopy")
		defer os.Remove("testcopy2")
		ioutil.WriteFile("testcopy", bytes, 0600)
		CopyFile("testcopy", "testcopy2")
		bytes2, _ := ioutil.ReadFile("testcopy2")
		So(bytes2, ShouldResemble, bytes)

		n, err := CopyFile("testcopy", "testcopy")
		So(err, ShouldBeNil)
		So(n, ShouldBeZeroValue)

		n, err = CopyFile("/path/not/exist", "testcopy")
		So(err, ShouldNotBeNil)
		So(n, ShouldBeZeroValue)

		n, err = CopyFile("testcopy", "/path/not/exist")
		So(err, ShouldNotBeNil)
		So(n, ShouldBeZeroValue)
	})
}

func TestHomeDirExpand(t *testing.T) {
	Convey("expand ~ dir", t, func() {
		usr, err := user.Current()
		So(err, ShouldBeNil)

		homeDir := HomeDirExpand("~")
		So(homeDir, ShouldEqual, usr.HomeDir)

		fullFilepathWithHome := HomeDirExpand("~/.local")
		So(fullFilepathWithHome, ShouldEqual, usr.HomeDir+"/.local")

		fullFilepathRaw := HomeDirExpand("/dev/null")
		So(fullFilepathRaw, ShouldEqual, "/dev/null")

		emptyPath := HomeDirExpand("")
		So(emptyPath, ShouldEqual, "")
	})
}

func TestExist(t *testing.T) {
	Convey("path exist or not", t, func() {
		So(Exist("/tmp/anemptypathshouldnotexist"), ShouldEqual, false)
		So(Exist("/"), ShouldEqual, true)
		So(Exist("/dev/null"), ShouldEqual, true)
	})
}
