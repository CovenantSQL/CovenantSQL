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
	UsageLine:   "cql transfer [-wait-tx-confirm] [meta_json]",
	Description: "Transfer token to target account",
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
	configInit()

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
