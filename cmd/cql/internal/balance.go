package internal

import (
	"strings"

	"github.com/CovenantSQL/CovenantSQL/client"
	"github.com/CovenantSQL/CovenantSQL/types"
)

var (
	tokenName string // get specific token's balance of current account
)

var CmdBalance = &Command{
	UsageLine:   "cql balance [-token token_name]",
	Description: "Get CovenantSQL balance of current account",
}

func init() {
	CmdBalance.Run = runBalance

	addCommonFlags(CmdBalance)
	CmdBalance.Flag.StringVar(&tokenName, "token", "", "Get specific token's balance of current account, e.g. Particle, Wave, and etc.")
}

func runBalance(cmd *Command, args []string) {
	configInit()

	var err error
	if tokenName == "" {
		var stableCoinBalance, covenantCoinBalance uint64

		if stableCoinBalance, err = client.GetTokenBalance(types.Particle); err != nil {
			ConsoleLog.WithError(err).Error("get Particle balance failed")
			SetExitStatus(1)
			return
		}
		if covenantCoinBalance, err = client.GetTokenBalance(types.Wave); err != nil {
			ConsoleLog.WithError(err).Error("get Wave balance failed")
			SetExitStatus(1)
			return
		}

		ConsoleLog.Infof("Particle balance is: %d", stableCoinBalance)
		ConsoleLog.Infof("Wave balance is: %d", covenantCoinBalance)
	} else {
		var tokenBalance uint64
		tokenType := types.FromString(tokenName)
		if !tokenType.Listed() {
			values := make([]string, len(types.TokenList))
			for i := types.Particle; i < types.SupportTokenNumber; i++ {
				values[i] = types.TokenList[i]
			}
			ConsoleLog.Errorf("no such token supporting in CovenantSQL (what we support: %s)",
				strings.Join(values, ", "))
			SetExitStatus(1)
			return
		}
		if tokenBalance, err = client.GetTokenBalance(tokenType); err != nil {
			ConsoleLog.WithError(err).Error("get token balance failed")
			SetExitStatus(1)
			return
		}
		ConsoleLog.Infof("%s balance is: %d", tokenType.String(), tokenBalance)
	}
}
