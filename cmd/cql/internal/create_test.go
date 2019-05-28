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

func TestCreate(t *testing.T) {
	// reset
	commonVarsReset()
	targetMiners = List{}
	node32 = 0
	meta = client.ResourceMeta{}

	Convey("create", t, func() {
		client.UnInit()
		//targetMiners = List{[]string{"000005aa62048f85da4ae9698ed59c14ec0d48a88a07c15a32265634e7e64ade", "000005f4f22c06f76c43c4f48d5a7ec1309cc94030cbf9ebae814172884ac8b5"}}
		node32 = 1
		// waitTxConfirmation = true
		configFile = FJ(testWorkingDir, "./bench_testnet/node_c/config.yaml")
		jsonStr := `
			{
					"loadavgpercpu": 0,
					"encryptionkey": "",
					"useeventualconsistency": false,
					"consistencylevel": 1,
					"isolationlevel": 1,
					"gasprice": 1,
					"advancepayment": 20000000
			}
		`
		runCreate(CmdCreate, []string{jsonStr})
	})
}
