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
	"flag"
	"fmt"
	"strings"

	"github.com/CovenantSQL/CovenantSQL/client"
	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/rpc/mux"
	"github.com/CovenantSQL/CovenantSQL/types"
)

var (
	tokenName  string // get specific token's balance of current account
	databaseID string
)

// CmdWallet is cql wallet command entity.
var CmdWallet = &Command{
	UsageLine: "cql wallet [common params] [-token type] [-dsn dsn]",
	Short:     "get the wallet address and the balance of current account",
	Long: `
Wallet gets the CovenantSQL wallet address and the token balances of the current account.
e.g.
    cql wallet

    cql wallet -token Particle

    cql wallet -dsn "covenantsql://4119ef997dedc585bfbcfae00ab6b87b8486fab323a8e107ea1fd4fc4f7eba5c"
`,
	Flag:       flag.NewFlagSet("Wallet params", flag.ExitOnError),
	CommonFlag: flag.NewFlagSet("Common params", flag.ExitOnError),
	DebugFlag:  flag.NewFlagSet("Debug params", flag.ExitOnError),
}

func init() {
	CmdWallet.Run = runWallet

	addCommonFlags(CmdWallet)
	addConfigFlag(CmdWallet)

	CmdWallet.Flag.StringVar(&tokenName, "token", "", "Get specific token balance of current account, e.g. Particle, Wave, All")
	CmdWallet.Flag.StringVar(&databaseID, "dsn", "", "Show specified database deposit")
}

func showTokenBalance(tokenName string) {
	var (
		tokenBalance uint64
		err          error
	)

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
		if strings.Contains(err.Error(), "no such token balance") {
			fmt.Println("Your account is not created in the TestNet, please apply tokens from our faucet first.")
		} else {
			ConsoleLog.WithError(err).Error("get token balance failed")
			SetExitStatus(1)
			return
		}
	}

	fmt.Printf("%s balance is: %d\n", tokenType, tokenBalance)
}

func showAllTokenBalance() {
	var (
		stableCoinBalance   uint64
		covenantCoinBalance uint64
		err                 error
	)

	if stableCoinBalance, err = client.GetTokenBalance(types.Particle); err != nil {
		if strings.Contains(err.Error(), "no such token balance") {
			fmt.Println("Your account is not created in the TestNet, please apply tokens from our faucet first.")
		} else {
			ConsoleLog.WithError(err).Error("get Particle balance failed")
		}

		SetExitStatus(1)
		return
	}

	if covenantCoinBalance, err = client.GetTokenBalance(types.Wave); err != nil {
		if strings.Contains(err.Error(), "no such token balance") {
			fmt.Println("Your account is not created in the TestNet, please apply tokens from our faucet first.")
		} else {
			ConsoleLog.WithError(err).Error("get Wave balance failed")
		}

		SetExitStatus(1)
		return
	}

	fmt.Printf("Particle balance is: %d\n", stableCoinBalance)
	fmt.Printf("Wave balance is: %d\n", covenantCoinBalance)
}

func showDatabaseDeposit(dsn string) {
	dsnCfg, err := client.ParseDSN(dsn)
	if err != nil {
		ConsoleLog.WithError(err).Error("parse database dsn failed")
		SetExitStatus(1)
		return
	}

	var (
		req    = new(types.QuerySQLChainProfileReq)
		resp   = new(types.QuerySQLChainProfileResp)
		pubKey *asymmetric.PublicKey
		addr   proto.AccountAddress
	)

	req.DBID = proto.DatabaseID(dsnCfg.DatabaseID)

	if err = mux.RequestBP(route.MCCQuerySQLChainProfile.String(), req, resp); err != nil {
		ConsoleLog.WithError(err).Error("query database chain profile failed")
		SetExitStatus(1)
		return
	}

	if pubKey, err = kms.GetLocalPublicKey(); err != nil {
		ConsoleLog.WithError(err).Error("query database chain profile failed")
		SetExitStatus(1)
		return
	}

	if addr, err = crypto.PubKeyHash(pubKey); err != nil {
		ConsoleLog.WithError(err).Error("query database chain profile failed")
		SetExitStatus(1)
		return
	}

	for _, user := range resp.Profile.Users {
		if user.Address == addr && user.Permission != nil && user.Permission.Role != types.Void {
			fmt.Printf("Chain token type: %s\n", resp.Profile.TokenType.String())
			fmt.Printf("Depsoit: %d\n", user.Deposit)
			fmt.Printf("Arrears: %d\n", user.Arrears)
			fmt.Printf("AdvancePayment: %d\n", user.AdvancePayment)
			return
		}
	}

	ConsoleLog.Error("no permission to the database")
	SetExitStatus(1)
	return
}

func showAllDatabaseDeposit() {
	var (
		req    = new(types.QueryAccountSQLChainProfilesReq)
		resp   = new(types.QueryAccountSQLChainProfilesResp)
		pubKey *asymmetric.PublicKey
		err    error
	)

	if pubKey, err = kms.GetLocalPublicKey(); err != nil {
		ConsoleLog.WithError(err).Error("query database chain profile failed")
		SetExitStatus(1)
		return
	}

	if req.Addr, err = crypto.PubKeyHash(pubKey); err != nil {
		ConsoleLog.WithError(err).Error("query database chain profile failed")
		SetExitStatus(1)
		return
	}

	if err = mux.RequestBP(route.MCCQueryAccountSQLChainProfiles.String(), req, resp); err != nil {
		if strings.Contains(err.Error(), "can't find method") {
			// old version block producer
			ConsoleLog.WithError(err).Warning("query account database profiles is not supported in old version block producer")
			return
		}

		ConsoleLog.WithError(err).Error("query account database profiles failed")
		SetExitStatus(1)
		return
	}

	if len(resp.Profiles) == 0 {
		fmt.Println("found no related database")
		return
	}

	fmt.Printf("Database Deposits:\n\n")
	fmt.Printf("%-64s\tDeposit\tArrears\tAdvancePayment\n", "DatabaseID")

	for _, p := range resp.Profiles {
		for _, user := range p.Users {
			if user.Address == req.Addr && user.Permission != nil && user.Permission.Role != types.Void {
				fmt.Printf("%s\t%d\t%d\t%d\n",
					p.ID, user.Deposit, user.Arrears, user.AdvancePayment)
			}
		}
	}
}

func runWallet(cmd *Command, args []string) {
	commonFlagsInit(cmd)
	configInit()

	fmt.Printf("\n\nwallet address: %s\n", conf.GConf.WalletAddress)

	if databaseID != "" {
		showDatabaseDeposit(databaseID)
	} else if tokenName == "" {
		showAllTokenBalance()
		showAllDatabaseDeposit()
	} else {
		showTokenBalance(tokenName)
	}
}
