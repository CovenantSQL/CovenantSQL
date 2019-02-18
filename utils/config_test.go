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
	"path/filepath"
	"runtime"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestDupConf(t *testing.T) {
	Convey("dup config file", t, func() {
		var d string
		var err error
		d, err = ioutil.TempDir("", "utils_test_")
		So(err, ShouldBeNil)
		dupConfFile := filepath.Join(d, "config.yaml")

		_, testFile, _, _ := runtime.Caller(0)
		confFile := filepath.Join(filepath.Dir(testFile), "../test/node_standalone/config.yaml")

		err = DupConf(confFile, dupConfFile)
		So(err, ShouldBeNil)

		err = DupConf("", dupConfFile)
		So(err, ShouldNotBeNil)

		err = DupConf(confFile, "")
		So(err, ShouldNotBeNil)
	})
}
