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

func TestExplorer(t *testing.T) {
	// reset
	commonVarsReset()
	explorerAddr = ""
	explorerService = nil
	explorerHTTPServer = nil

	Convey("explorer", t, func() {
		explorerAddr = "127.0.0.1:9002"
		configFile = FJ(testWorkingDir, "./bench_testnet/node_c/config.yaml")
		client.UnInit()
		configInit()
		bgServerInit()
		cancelFunc := startExplorerServer(explorerAddr)
		cancelFunc()
	})
}
