/*
 * Copyright 2016-2018 Kenneth Shaw.
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

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	pi "github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	"github.com/CovenantSQL/CovenantSQL/client"
	"github.com/CovenantSQL/CovenantSQL/cmd/cql/internal"
	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/sqlchain/adapter"
	"github.com/CovenantSQL/CovenantSQL/sqlchain/observer"
	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

var (
	version           = "unknown"
	dsn               string
	command           string
	fileName          string
	outFile           string
	noRC              bool
	configFile        string
	password          string
	singleTransaction bool
	showVersion       bool
	variables         internal.VarsFlag

	// Shard chain explorer/adapter stuff
	tmpPath      string // background observer and explorer block and log file path
	bgLogLevel   string // background log level
	explorerAddr string // explorer Web addr
	adapterAddr  string // adapter listen addr

	// DML variables
	createDB                string // as a instance meta json string or simply a node count
	dropDB                  string // database id to drop
	updatePermission        string // update user's permission on specific sqlchain
	transferToken           string // transfer token to target account
	getBalance              bool   // get balance of current account
	getBalanceWithTokenName string // get specific token's balance of current account
	waitTxConfirmation      bool   // wait for transaction confirmation before exiting

	service    *observer.Service
	httpServer *http.Server
)

type userPermission struct {
	TargetChain proto.AccountAddress `json:"chain"`
	TargetUser  proto.AccountAddress `json:"user"`
	Perm        json.RawMessage      `json:"perm"`
}

type userPermPayload struct {
	// User role to access database.
	Role types.UserPermissionRole `json:"role"`
	// SQL pattern regulations for user queries
	// only a fully matched (case-sensitive) sql query is permitted to execute.
	Patterns []string `json:"patterns"`
}

type tranToken struct {
	TargetUser proto.AccountAddress `json:"addr"`
	Amount     string               `json:"amount"`
}

func init() {
	flag.StringVar(&dsn, "dsn", "", "Database url")
	flag.StringVar(&command, "command", "", "Run only single command (SQL or usql internal command) and exit")
	flag.StringVar(&fileName, "file", "", "Execute commands from file and exit")
	flag.BoolVar(&showVersion, "version", false, "Show version information and exit")
	flag.BoolVar(&noRC, "no-rc", false, "Do not read start up file")
	flag.BoolVar(&asymmetric.BypassSignature, "bypass-signature", false,
		"Disable signature sign and verify, for testing")
	flag.StringVar(&outFile, "out", "", "Record stdout to file")
	flag.StringVar(&configFile, "config", "~/.cql/config.yaml", "Config file for covenantsql")
	flag.StringVar(&password, "password", "", "Master key password for covenantsql")
	flag.BoolVar(&singleTransaction, "single-transaction", false, "Execute as a single transaction (if non-interactive)")
	flag.Var(&variables, "variable", "Set variable")

	// Explorer/Adapter
	flag.StringVar(&tmpPath, "tmp-path", "", "Background service temp file path, use os.TempDir for default")
	flag.StringVar(&bgLogLevel, "bg-log-level", "", "Background service log level")
	flag.StringVar(&explorerAddr, "web", "", "Address to serve a database chain explorer, e.g. :8546")
	flag.StringVar(&adapterAddr, "adapter", "", "Address to serve a database chain adapter, e.g. :7784")

	// DML flags
	flag.StringVar(&createDB, "create", "", "Create database, argument can be instance requirement json or simply a node count requirement")
	flag.StringVar(&dropDB, "drop", "", "Drop database, argument should be a database id (without covenantsql:// scheme is acceptable)")
	flag.StringVar(&updatePermission, "update-perm", "", "Update user's permission on specific sqlchain")
	flag.StringVar(&transferToken, "transfer", "", "Transfer token to target account")
	flag.BoolVar(&getBalance, "get-balance", false, "Get balance of current account")
	flag.StringVar(&getBalanceWithTokenName, "token-balance", "", "Get specific token's balance of current account, e.g. Particle, Wave, and etc.")
	flag.BoolVar(&waitTxConfirmation, "wait-tx-confirm", false, "Wait for transaction confirmation")
}

func main() {

	internal.Version = version

	var err error
	// set random
	rand.Seed(time.Now().UnixNano())

	flag.Parse()

	internal.ConsoleLog = logrus.New()

	if showVersion {
		fmt.Printf("%v %v %v %v %v\n",
			internal.Name, internal.Version, runtime.GOOS, runtime.GOARCH, runtime.Version())
		os.Exit(0)
	}
	internal.ConsoleLog.Infof("cql build: %#v\n", internal.Version)

	configFile = utils.HomeDirExpand(configFile)

	// init covenantsql driver
	if err = client.Init(configFile, []byte(password)); err != nil {
		internal.ConsoleLog.WithError(err).Error("init covenantsql client failed")
		os.Exit(-1)
		return
	}

	if tmpPath == "" {
		tmpPath = os.TempDir()
	}
	logPath := filepath.Join(tmpPath, "covenant_service.log")
	bgLog, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open log file failed: %s, %v", logPath, err)
		os.Exit(-1)
	}
	log.SetOutput(bgLog)
	log.SetStringLevel(bgLogLevel, log.InfoLevel)

	if explorerAddr != "" {
		service, httpServer, err = observer.StartObserver(explorerAddr, internal.Version)
		if err != nil {
			log.WithError(err).Fatal("start explorer failed")
		} else {
			internal.ConsoleLog.Infof("explorer started on %s", explorerAddr)
		}

		defer func() {
			_ = observer.StopObserver(service, httpServer)
			log.Info("explorer stopped")
		}()
	}

	if adapterAddr != "" {
		server, err := adapter.NewHTTPAdapter(adapterAddr, configFile)
		if err != nil {
			log.WithError(err).Fatal("init adapter failed")
		}

		if err = server.Serve(); err != nil {
			log.WithError(err).Fatal("start adapter failed")
		} else {
			internal.ConsoleLog.Infof("adapter started on %s", adapterAddr)

			defer func() {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
				defer cancel()
				server.Shutdown(ctx)
				log.Info("stopped adapter")
			}()
		}
	}

	// TODO(leventeliu): discover more specific confirmation duration from config. We don't have
	// enough informations from config to do that currently, so just use a fixed and long enough
	// duration.
	internal.WaitTxConfirmationMaxDuration = 20 * conf.GConf.BPPeriod

	if getBalance {
		var stableCoinBalance, covenantCoinBalance uint64

		if stableCoinBalance, err = client.GetTokenBalance(types.Particle); err != nil {
			internal.ConsoleLog.WithError(err).Error("get Particle balance failed")
			return
		}
		if covenantCoinBalance, err = client.GetTokenBalance(types.Wave); err != nil {
			internal.ConsoleLog.WithError(err).Error("get Wave balance failed")
			return
		}

		internal.ConsoleLog.Infof("Particle balance is: %d", stableCoinBalance)
		internal.ConsoleLog.Infof("Wave balance is: %d", covenantCoinBalance)

		return
	}

	if getBalanceWithTokenName != "" {
		var tokenBalance uint64
		tokenType := types.FromString(getBalanceWithTokenName)
		if !tokenType.Listed() {
			values := make([]string, len(types.TokenList))
			for i := types.Particle; i < types.SupportTokenNumber; i++ {
				values[i] = types.TokenList[i]
			}
			internal.ConsoleLog.Errorf("no such token supporting in CovenantSQL (what we support: %s)",
				strings.Join(values, ", "))
			os.Exit(-1)
			return
		}
		if tokenBalance, err = client.GetTokenBalance(tokenType); err != nil {
			internal.ConsoleLog.WithError(err).Error("get token balance failed")
			os.Exit(-1)
			return
		}
		internal.ConsoleLog.Infof("%s balance is: %d", tokenType.String(), tokenBalance)
		return
	}

	if dropDB != "" {
		// drop database
		if _, err := client.ParseDSN(dropDB); err != nil {
			// not a dsn
			cfg := client.NewConfig()
			cfg.DatabaseID = dropDB
			dropDB = cfg.FormatDSN()
		}

		txHash, err := client.Drop(dropDB)
		if err != nil {
			// drop database failed
			internal.ConsoleLog.WithField("db", dropDB).WithError(err).Error("drop database failed")
			return
		}

		if waitTxConfirmation {
			wait(txHash)
		}

		// drop database success
		internal.ConsoleLog.Infof("drop database %#v success", dropDB)
		return
	}

	if createDB != "" {
		// create database
		// parse instance requirement
		var meta client.ResourceMeta

		if err := json.Unmarshal([]byte(createDB), &meta); err != nil {
			// not a instance json, try if it is a number describing node count
			nodeCnt, err := strconv.ParseUint(createDB, 10, 16)
			if err != nil {
				// still failing
				internal.ConsoleLog.WithField("db", createDB).Error("create database failed: invalid instance description")
				os.Exit(-1)
				return
			}

			meta = client.ResourceMeta{}
			meta.Node = uint16(nodeCnt)
		}

		txHash, dsn, err := client.Create(meta)
		if err != nil {
			internal.ConsoleLog.WithError(err).Error("create database failed")
			os.Exit(-1)
			return
		}

		if waitTxConfirmation {
			wait(txHash)
			var ctx, cancel = context.WithTimeout(context.Background(), internal.WaitTxConfirmationMaxDuration)
			defer cancel()
			err = client.WaitDBCreation(ctx, dsn)
			if err != nil {
				internal.ConsoleLog.WithError(err).Error("create database failed durating creation")
				os.Exit(-1)
			}
		}

		internal.ConsoleLog.Infof("the newly created database is: %#v", dsn)
		fmt.Printf(dsn)
		return
	}

	if updatePermission != "" {
		// update user's permission on sqlchain
		var perm userPermission
		if err := json.Unmarshal([]byte(updatePermission), &perm); err != nil {
			internal.ConsoleLog.WithError(err).Errorf("update permission failed: invalid permission description")
			os.Exit(-1)
			return
		}

		var permPayload userPermPayload

		if err := json.Unmarshal(perm.Perm, &permPayload); err != nil {
			// try again using role string representation
			if err := json.Unmarshal(perm.Perm, &permPayload.Role); err != nil {
				internal.ConsoleLog.WithError(err).Errorf("update permission failed: invalid permission description")
				os.Exit(-1)
				return
			}
		}

		p := &types.UserPermission{
			Role:     permPayload.Role,
			Patterns: permPayload.Patterns,
		}

		if !p.IsValid() {
			internal.ConsoleLog.Errorf("update permission failed: invalid permission description")
			os.Exit(-1)
			return
		}

		txHash, err := client.UpdatePermission(perm.TargetUser, perm.TargetChain, p)
		if err != nil {
			internal.ConsoleLog.WithError(err).Error("update permission failed")
			os.Exit(-1)
			return
		}

		if waitTxConfirmation {
			err = wait(txHash)
			if err != nil {
				os.Exit(-1)
				return
			}
		}

		internal.ConsoleLog.Info("succeed in sending transaction to CovenantSQL")
		return
	}

	if transferToken != "" {
		// transfer token
		var tran tranToken
		if err := json.Unmarshal([]byte(transferToken), &tran); err != nil {
			internal.ConsoleLog.WithError(err).Errorf("transfer token failed: invalid transfer description")
			os.Exit(-1)
			return
		}

		var validAmount = regexp.MustCompile(`^([0-9]+) *([a-zA-Z]+)$`)
		if !validAmount.MatchString(tran.Amount) {
			internal.ConsoleLog.Error("transfer token failed: invalid transfer description")
			os.Exit(-1)
			return
		}
		amountUnit := validAmount.FindStringSubmatch(tran.Amount)
		if len(amountUnit) != 3 {
			internal.ConsoleLog.Error("transfer token failed: invalid transfer description")
			for _, v := range amountUnit {
				internal.ConsoleLog.Error(v)
			}
			os.Exit(-1)
			return
		}
		amount, err := strconv.ParseUint(amountUnit[1], 10, 64)
		if err != nil {
			internal.ConsoleLog.Error("transfer token failed: invalid token amount")
			os.Exit(-1)
			return
		}
		unit := types.FromString(amountUnit[2])
		if !unit.Listed() {
			internal.ConsoleLog.Error("transfer token failed: invalid token type")
			os.Exit(-1)
			return
		}

		var txHash hash.Hash
		txHash, err = client.TransferToken(tran.TargetUser, amount, unit)
		if err != nil {
			internal.ConsoleLog.WithError(err).Error("transfer token failed")
			os.Exit(-1)
			return
		}

		if waitTxConfirmation {
			err = wait(txHash)
			if err != nil {
				os.Exit(-1)
				return
			}
		}

		internal.ConsoleLog.Info("succeed in sending transaction to CovenantSQL")
		return
	}

	internal.RunConsole(dsn, command, fileName, outFile, noRC, singleTransaction, variables)

	// if web flag is enabled
	if explorerAddr != "" || adapterAddr != "" {
		fmt.Printf("Ctrl + C to stop explorer on %s and adapter on %s\n", explorerAddr, adapterAddr)
		<-utils.WaitForExit()
		return
	}
}

func wait(txHash hash.Hash) (err error) {
	var ctx, cancel = context.WithTimeout(context.Background(), internal.WaitTxConfirmationMaxDuration)
	defer cancel()
	var state pi.TransactionState
	state, err = client.WaitTxConfirmation(ctx, txHash)
	internal.ConsoleLog.WithFields(logrus.Fields{
		"tx_hash":  txHash,
		"tx_state": state,
	}).WithError(err).Info("wait transaction confirmation")
	if err == nil && state != pi.TransactionStateConfirmed {
		err = errors.New("bad transaction state")
	}
	return
}
