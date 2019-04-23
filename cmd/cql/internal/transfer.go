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
	"encoding/json"
	"regexp"
	"strconv"

	"github.com/CovenantSQL/CovenantSQL/client"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/types"
)

// CmdTransfer is cql transfer command entity.
var CmdTransfer = &Command{
	UsageLine: "cql transfer [common params] [-wait-tx-confirm] meta_json",
	Short:     "transfer token to target account",
	Long: `
Transfer transfers your token to the target account.
The command argument is a token transaction in JSON format.
e.g.
    cql transfer '{"addr": "43602c17adcc96acf2f68964830bb6ebfbca6834961c0eca0915fcc5270e0b40", "amount": "100 Particle"}'

Since CovenantSQL is built on top of blockchains, you may want to wait for the transaction
confirmation before the transfer takes effect.
e.g.
    cql transfer -wait-tx-confirm '{"addr": "43602c17adcc96acf2f68964830bb6ebfbca6834961c0eca0915fcc5270e0b40", "amount": "100 Particle"}'
`,
}

func init() {
	CmdTransfer.Run = runTransfer

	addCommonFlags(CmdTransfer)
	addWaitFlag(CmdTransfer)
}

type tranToken struct {
	TargetUser proto.AccountAddress `json:"addr"`
	Amount     string               `json:"amount"`
}

func runTransfer(cmd *Command, args []string) {
	configInit(cmd)

	if len(args) != 1 {
		ConsoleLog.Error("Transfer command need target user and token amount in json string as param")
		SetExitStatus(1)
		return
	}

	transferStr := args[0]

	// transfer token
	var tran tranToken
	if err := json.Unmarshal([]byte(transferStr), &tran); err != nil {
		ConsoleLog.WithError(err).Errorf("transfer token failed: invalid transfer description")
		SetExitStatus(1)
		return
	}

	var validAmount = regexp.MustCompile(`^([0-9]+) *([a-zA-Z]+)$`)
	if !validAmount.MatchString(tran.Amount) {
		ConsoleLog.Error("transfer token failed: invalid transfer description")
		SetExitStatus(1)
		return
	}
	amountUnit := validAmount.FindStringSubmatch(tran.Amount)
	if len(amountUnit) != 3 {
		ConsoleLog.Error("transfer token failed: invalid transfer description")
		for _, v := range amountUnit {
			ConsoleLog.Error(v)
		}
		SetExitStatus(1)
		return
	}
	amount, err := strconv.ParseUint(amountUnit[1], 10, 64)
	if err != nil {
		ConsoleLog.Error("transfer token failed: invalid token amount")
		SetExitStatus(1)
		return
	}
	unit := types.FromString(amountUnit[2])
	if !unit.Listed() {
		ConsoleLog.Error("transfer token failed: invalid token type")
		SetExitStatus(1)
		return
	}

	var txHash hash.Hash
	txHash, err = client.TransferToken(tran.TargetUser, amount, unit)
	if err != nil {
		ConsoleLog.WithError(err).Error("transfer token failed")
		SetExitStatus(1)
		return
	}

	if waitTxConfirmation {
		err = wait(txHash)
		if err != nil {
			SetExitStatus(1)
			return
		}
	}

	ConsoleLog.Info("succeed in sending transaction to CovenantSQL")
}
