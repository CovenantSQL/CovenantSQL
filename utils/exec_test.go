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
	"bytes"
	"io/ioutil"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

var (
	baseDir        = GetProjectSrcDir()
	testWorkingDir = FJ(baseDir, "./test/")
	logDir         = FJ(testWorkingDir, "./log/")
)

func TestRunServer(t *testing.T) {
	Convey("build", t, func() {
		log.SetLevel(log.DebugLevel)
		RunCommand(
			"/bin/ls",
			[]string{},
			"ls", testWorkingDir, logDir, true,
		)
		lsOut, _ := ioutil.ReadFile(FJ(logDir, "ls.log"))
		So(bytes.ContainsAny(lsOut, "node_c"), ShouldBeTrue)

		err := RunCommand(
			"/bin/xxxxx",
			[]string{},
			"ls", testWorkingDir, logDir, false,
		)
		So(err, ShouldNotBeNil)

		err = RunCommand(
			"/bin/ls",
			[]string{},
			"ls", testWorkingDir+"noexist", logDir, false,
		)
		So(err, ShouldNotBeNil)

		err = RunCommand(
			"/bin/ls",
			[]string{},
			"ls", testWorkingDir, logDir+"noexist", true,
		)
		So(err, ShouldBeNil)
	})
}
