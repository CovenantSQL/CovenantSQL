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
	"strings"

	"github.com/CovenantSQL/CovenantSQL/client"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/types"
)

var (
	toUser    string
	toDSN     string
	amount    uint64
	tokenType string
)

// CmdTransfer is cql transfer command entity.
var CmdTransfer = &Command{
	UsageLine: "cql transfer [common params] [-wait-tx-confirm] [-to-user wallet | -to-dsn dsn] [-amount count] [-type token_type]",
	Short:     "transfer token to target account",
	Long: `
Transfer transfers your token to the target account or database.
The command arguments are target wallet address(or dsn), amount of token, and token type.
e.g.
    cql transfer -to-user=43602c17adcc96acf2f68964830bb6ebfbca6834961c0eca0915fcc5270e0b40 -amount=100 -type=Particle

Since CovenantSQL is built on top of blockchains, you may want to wait for the transaction
confirmation before the transfer takes effect.
e.g.
    cql transfer -wait-tx-confirm -to-dsn=43602c17adcc96acf2f68964830bb6ebfbca6834961c0eca0915fcc5270e0b40 -amount=100 -type=Particle
`,
}

func init() {
	CmdTransfer.Run = runTransfer

	addCommonFlags(CmdTransfer)
	addConfigFlag(CmdTransfer)
	addWaitFlag(CmdTransfer)
	CmdTransfer.Flag.StringVar(&toUser, "to-user", "", "Target address of an user account to transfer token.")
	CmdTransfer.Flag.StringVar(&toDSN, "to-dsn", "", "Target database dsn to transfer token.")
	CmdTransfer.Flag.Uint64Var(&amount, "amount", 0, "Token account to transfer.")
	CmdTransfer.Flag.StringVar(&tokenType, "type", "", "Token type to transfer.")
}

func runTransfer(cmd *Command, args []string) {
	if len(args) > 0 || (toUser == "" && toDSN == "") || tokenType == "" {
		ConsoleLog.Error("transfer command need to-user(or to-dsn) address and token type as param")
		SetExitStatus(1)
		help = true
	}
	if toUser != "" && toDSN != "" {
		ConsoleLog.Error("transfer command accepts either to-user or to-dsn as param")
		SetExitStatus(1)
		help = true
	}

	commonFlagsInit(cmd)

	unit := types.FromString(tokenType)
	if !unit.Listed() {
		ConsoleLog.Error("transfer token failed: invalid token type")
		SetExitStatus(1)
		return
	}

	var addr string
	if toUser != "" {
		addr = toUser
	} else {
		if !strings.HasPrefix(toDSN, client.DBScheme) && !strings.HasPrefix(toDSN, client.DBSchemeAlias) {
			ConsoleLog.Error("transfer token failed: invalid dsn provided, use address start with 'covenantsql://'")
			SetExitStatus(1)
			return
		}
		toDSN = strings.TrimLeft(toDSN, client.DBScheme+"://")
		addr = strings.TrimLeft(toDSN, client.DBSchemeAlias+"://")
	}

	targetAccountHash, err := hash.NewHashFromStr(addr)
	if err != nil {
		ConsoleLog.WithError(err).Error("target account address is not valid")
		SetExitStatus(1)
		return
	}
	targetAccount := proto.AccountAddress(*targetAccountHash)

	configInit()

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
