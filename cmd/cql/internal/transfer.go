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
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/types"
)

var (
	addr      string
	amount    uint64
	tokenType string
)

// CmdTransfer is cql transfer command entity.
var CmdTransfer = &Command{
	UsageLine: "cql transfer [common params] [-wait-tx-confirm] [-address wallet] [-amount count] [-type token_type]",
	Short:     "transfer token to target account",
	Long: `
Transfer transfers your token to the target account.
The command arguments are target wallet address, amount of token, and token type.
e.g.
    cql transfer -addr=43602c17adcc96acf2f68964830bb6ebfbca6834961c0eca0915fcc5270e0b40 -amount=100 -type=Particle

Since CovenantSQL is built on top of blockchains, you may want to wait for the transaction
confirmation before the transfer takes effect.
e.g.
    cql transfer -wait-tx-confirm -addr=43602c17adcc96acf2f68964830bb6ebfbca6834961c0eca0915fcc5270e0b40 -amount=100 -type=Particle
`,
}

func init() {
	CmdTransfer.Run = runTransfer

	addCommonFlags(CmdTransfer)
	addWaitFlag(CmdTransfer)
	CmdTransfer.Flag.StringVar(&addr, "address", "", "Address of an account to transfer token.")
	CmdTransfer.Flag.Uint64Var(&amount, "amount", 0, "Token account to transfer.")
	CmdTransfer.Flag.StringVar(&tokenType, "type", "", "Token type to transfer.")
}

func runTransfer(cmd *Command, args []string) {
	if len(args) > 0 || addr == "" || tokenType == "" {
		ConsoleLog.Error("transfer command need target account address and token type as param")
		SetExitStatus(1)
		help = true
	}

	configInit(cmd)

	unit := types.FromString(tokenType)
	if !unit.Listed() {
		ConsoleLog.Error("transfer token failed: invalid token type")
		SetExitStatus(1)
		return
	}

	targetAccountHash, err := hash.NewHashFromStr(addr)
	if err != nil {
		ConsoleLog.WithError(err).Error("target account address is not valid")
		SetExitStatus(1)
		return
	}
	targetAccount := proto.AccountAddress(*targetAccountHash)

	txHash, err := client.TransferToken(targetAccount, amount, unit)
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
