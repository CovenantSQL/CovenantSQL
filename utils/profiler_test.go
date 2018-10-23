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
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

const (
	cpuProfile = ".cpuProfile"
	memProfile = ".memProfile"
)

func TestStartStopProfile(t *testing.T) {
	Convey("start normally", t, func() {
		defer os.Remove(cpuProfile)
		defer os.Remove(memProfile)
		err := StartProfile(cpuProfile, memProfile)
		So(err, ShouldBeNil)

		StopProfile()
		cpuFileInfo, err := os.Stat(cpuProfile)
		So(cpuFileInfo.Size(), ShouldBeGreaterThan, 0)
		So(err, ShouldBeNil)

		memFileInfo, err := os.Stat(memProfile)
		So(memFileInfo.Size(), ShouldBeGreaterThan, 0)
		So(err, ShouldBeNil)
	})
	Convey("start empty", t, func() {
		err := StartProfile("", "")
		So(err, ShouldBeNil)

		StopProfile()
	})
	Convey("start no such path", t, func() {
		err := StartProfile("/not/exist/path", "")
		So(err, ShouldNotBeNil)

		StopProfile()

		err = StartProfile("", "/not/exist/path")
		So(err, ShouldNotBeNil)

		StopProfile()
	})
}
