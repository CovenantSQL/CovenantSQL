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
	"testing"

	"github.com/CovenantSQL/CovenantSQL/client"
	. "github.com/smartystreets/goconvey/convey"
)

func testTransferReset() {
	toUser = ""
	toDSN = ""
	amount = 0
	tokenType = ""
}

func TestTransfer(t *testing.T) {
	Convey("transfer", t, func() {
		// reset
		commonVarsReset()
		testTransferReset()
		client.UnInit()

		toUser = "43602c17adcc96acf2f68964830bb6ebfbca6834961c0eca0915fcc5270e0b40"
		toDSN = ""
		amount = 100
		tokenType = "Particle"
		// waitTxConfirmation = true
		configFile = FJ(testWorkingDir, "./bench_testnet/node_c/config.yaml")
		runTransfer(CmdTransfer, []string{})
	})

	Convey("transfer", t, func() {
		// reset
		commonVarsReset()
		testTransferReset()
		client.UnInit()

		toDSN = "covenantsql://02a8ad1419fb2033cef8cf6f97ec16a784d90e654380eac7ce76b965e27c9e5c"
		toUser = ""
		amount = 100
		tokenType = "Particle"
		// waitTxConfirmation = true
		configFile = FJ(testWorkingDir, "./bench_testnet/node_c/config.yaml")
		runTransfer(CmdTransfer, []string{})
	})
}
