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
	"github.com/CovenantSQL/CovenantSQL/client"
	"github.com/CovenantSQL/CovenantSQL/utils"
	. "github.com/smartystreets/goconvey/convey"
	"path/filepath"
	"testing"
)

func TestRpc(t *testing.T) {
	FJ := filepath.Join
	baseDir := utils.GetProjectSrcDir()
	testWorkingDir := FJ(baseDir, "./test/")

	Convey("rpc", t, func() {
		client.UnInit()
		rpcName = "MCC.QuerySQLChainProfile"
		rpcEndpoint = "000000fd2c8f68d54d55d97d0ad06c6c0d91104e4e51a7247f3629cc2a0127cf"
		rpcReq = "{\"DBID\": \"c8328272ba9377acdf1ee8e73b17f2b0f7430c798141080d0282195507eb94e7\"}"
		configFile = FJ(testWorkingDir, "./bench_testnet/node_c/config.yaml")
		SetExitStatus(0)
		runRPC(CmdRPC, []string{})
	})
}
