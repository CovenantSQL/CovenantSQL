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
	"testing"

	"github.com/CovenantSQL/CovenantSQL/client"
	. "github.com/smartystreets/goconvey/convey"
)

func TestConsole(t *testing.T) {
	Convey("console", t, func() {
		// reset
		commonVarsReset()
		variables = varsFlag{}
		dsn = ""
		outFile = ""
		noRC = false
		singleTransaction = false
		command = ""
		fileName = ""
		adapterAddr = ""
		explorerAddr = ""
		client.UnInit()

		dsn = "covenantsql://c9e8b381aa466a8d9955701967ad5535e7899ab138b8674ab14b31b75c64b656"
		command = "select 1 from test1;"
		configFile = FJ(testWorkingDir, "./bench_testnet/node_c/config.yaml")
		runConsole(CmdConsole, []string{dsn})
	})
}
