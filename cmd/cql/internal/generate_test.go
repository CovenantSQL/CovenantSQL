// +build !testbinary

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

package internal

import (
	"os"
	"testing"

	"github.com/CovenantSQL/CovenantSQL/utils"
	. "github.com/smartystreets/goconvey/convey"
)

func testGenerateReset() {
	// reset
	commonVarsReset()
	privateKeyParam = ""
	source = ""
	minerListenAddr = ""
	difficulty = 10
}

func TestGenerate(t *testing.T) {
	Convey("generate", t, func(c C) {
		testGenerateReset()
		os.RemoveAll(utils.HomeDirExpand("~/.cql"))
		privateKeyParam = FJ(testWorkingDir, "./integration/node_c/private.key")
		source = FJ(testWorkingDir, "./integration/node_c/config.yaml")
		minerListenAddr = "127.0.0.1:12313"
		runGenerate(CmdGenerate, []string{""})
	})

	Convey("clean generate", t, func(c C) {
		testGenerateReset()
		os.RemoveAll(utils.HomeDirExpand("~/.cql"))
		runGenerate(CmdGenerate, []string{""})
	})
}
