// +build !testbinary

/*
 * Copyright 2018-2019 The CovenantSQL Authors.
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

package internal

import (
	"path/filepath"
	"testing"

	"github.com/CovenantSQL/CovenantSQL/utils"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	FJ             = filepath.Join
	baseDir        = utils.GetProjectSrcDir()
	testWorkingDir = FJ(baseDir, "./test/")
)

func commonVarsReset() {
	// common flags
	help = false
	withPassword = false
	password = ""
	consoleLogLevel = "info"

	// config flags
	configFile = "~/.cql/config.yaml"
	waitTxConfirmationMaxDuration = 0

	// wait flag
	waitTxConfirmation = false

	// bg server flags
	tmpPath = ""
	bgLogLevel = "info"
}

func TestBase(t *testing.T) {
	cmd := Command{
		UsageLine: "cql generate",
	}

	Convey("LongName", t, func() {
		longName := cmd.LongName()
		So(longName, ShouldResemble, "generate")
	})

	Convey("Name", t, func() {
		name := cmd.Name()
		So(name, ShouldResemble, "generate")
	})

	//Convey("Usage", t, func() {
	//	go cmd.Usage()
	//	time.Sleep(1 * time.Second)
	//})

	//Convey("Exit", t, func() {
	//	go func() {
	//		SetExitStatus(1)
	//		AtExit(func() {})
	//		ExitIfErrors()
	//	}()
	//	time.Sleep(1 * time.Second)
	//})
}
