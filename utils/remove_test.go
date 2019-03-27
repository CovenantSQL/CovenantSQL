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

package utils

import (
	"io/ioutil"
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestRemoveAll(t *testing.T) {
	Convey("test remove files", t, func() {
		var names []string
		tempPattern := "_tempfile_test_never_duplicate_*"
		f, err := ioutil.TempFile(".", tempPattern)
		So(err, ShouldBeNil)
		names = append(names, f.Name())
		_ = f.Close()
		f, err = ioutil.TempFile(".", tempPattern)
		So(err, ShouldBeNil)
		names = append(names, f.Name())
		_ = f.Close()

		RemoveAll(tempPattern)

		for _, name := range names {
			_, err := os.Stat(name)
			So(err, ShouldNotBeNil)
			So(os.IsNotExist(err), ShouldBeTrue)
		}
	})
}
