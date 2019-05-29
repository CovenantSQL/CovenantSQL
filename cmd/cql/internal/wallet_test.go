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

func testWalletReset() {
	// reset
	commonVarsReset()
	tokenName = ""
	databaseID = ""
}

func TestWallet(t *testing.T) {
	Convey("wallet", t, func() {
		testWalletReset()
		client.UnInit()

		databaseID = "covenantsql://658f90f678a90207ab7f442e259f915192188c2f4a4efe7bbf0d69311841766c"
		tokenName = ""
		configFile = FJ(testWorkingDir, "./bench_testnet/node_c/config.yaml")
		runWallet(CmdWallet, []string{""})
	})

	Convey("wallet", t, func() {
		testWalletReset()
		client.UnInit()

		configFile = FJ(testWorkingDir, "./bench_testnet/node_c/config.yaml")
		databaseID = ""
		tokenName = ""
		runWallet(CmdWallet, []string{""})
	})

	Convey("wallet", t, func() {
		testWalletReset()
		client.UnInit()

		tokenName = "Particle"
		databaseID = ""
		configFile = FJ(testWorkingDir, "./bench_testnet/node_c/config.yaml")
		runWallet(CmdWallet, []string{""})
	})
}
