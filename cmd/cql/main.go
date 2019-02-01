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
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	sqlite3 "github.com/CovenantSQL/go-sqlite3-encrypt"
	"github.com/sirupsen/logrus"
	"github.com/xo/dburl"
	"github.com/xo/usql/drivers"
	"github.com/xo/usql/env"
	"github.com/xo/usql/handler"
	"github.com/xo/usql/rline"
	"github.com/xo/usql/text"

	pi "github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	"github.com/CovenantSQL/CovenantSQL/client"
	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

const name = "cql"

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
	variables         varsFlag

	// Shard chain explorer stuff
	tmpPath    string         // background observer and explorer block and log file path
	cLog       *logrus.Logger // console logger
	bgLogLevel string         // background log level

	// DML variables
	createDB                string // as a instance meta json string or simply a node count
	dropDB                  string // database id to drop
	updatePermission        string // update user's permission on specific sqlchain
	transferToken           string // transfer token to target account
	getBalance              bool   // get balance of current account
	getBalanceWithTokenName string // get specific token's balance of current account
	waitTxConfirmation      bool   // wait for transaction confirmation before exiting

	waitTxConfirmationMaxDuration time.Duration
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

type varsFlag struct {
	flag.Value
	vars []string
}

func (v *varsFlag) Get() []string {
	return append([]string{}, v.vars...)
}

func (v *varsFlag) String() string {
	return fmt.Sprintf("%#v", v.vars)
}

func (v *varsFlag) Set(value string) error {
	v.vars = append(v.vars, value)
	return nil
}

func usqlRegister() {
	// set command name of usql
	text.CommandName = "covenantsql"

	// register SQLite3 database
	drivers.Register("sqlite3", drivers.Driver{
		AllowMultilineComments: true,
		ForceParams: drivers.ForceQueryParameters([]string{
			"loc", "auto",
		}),
		Version: func(db drivers.DB) (string, error) {
			var ver string
			err := db.QueryRow(`SELECT sqlite_version()`).Scan(&ver)
			if err != nil {
				return "", err
			}
			return "SQLite3 " + ver, nil
		},
		Err: func(err error) (string, string) {
			if e, ok := err.(sqlite3.Error); ok {
				return strconv.Itoa(int(e.Code)), e.Error()
			}

			code, msg := "", err.Error()
			if e, ok := err.(sqlite3.ErrNo); ok {
				code = strconv.Itoa(int(e))
			}

			return code, msg
		},
		ConvertBytes: func(buf []byte, tfmt string) (string, error) {
			// attempt to convert buf if it matches a time format, and if it
			// does, then return a formatted time string.
			s := string(buf)
			if s != "" && strings.TrimSpace(s) != "" {
				t := new(SqTime)
				if err := t.Scan(buf); err == nil {
					return t.Format(tfmt), nil
				}
			}
			return s, nil
		},
	})

	// register CovenantSQL database
	drivers.Register("covenantsql", drivers.Driver{
		AllowMultilineComments: true,
		Version: func(db drivers.DB) (string, error) {
			return version, nil
		},
		Err: func(err error) (string, string) {
			return "", err.Error()
		},
		ConvertBytes: func(buf []byte, tfmt string) (string, error) {
			// attempt to convert buf if it matches a time format, and if it
			// does, then return a formatted time string.
			s := string(buf)
			if s != "" && strings.TrimSpace(s) != "" {
				t := new(SqTime)
				if err := t.Scan(buf); err == nil {
					return t.Format(tfmt), nil
				}
			}
			return s, nil
		},
		RowsAffected: func(sql.Result) (int64, error) {
			return 0, nil
		},
		Open: func(url *dburl.URL) (handler func(driver string, dsn string) (*sql.DB, error), err error) {
			log.Infof("connecting to %#v", url.DSN)

			// wait for database to become ready
			ctx, cancel := context.WithTimeout(context.Background(), waitTxConfirmationMaxDuration)
			defer cancel()
			if err = client.WaitDBCreation(ctx, dsn); err != nil {
				return
			}

			return sql.Open, nil
		},
	})

	// register covenantsql:// scheme to dburl
	dburl.Register(dburl.Scheme{
		Driver: "covenantsql",
		Generator: func(url *dburl.URL) (string, error) {
			dbID, err := dburl.GenOpaque(url)
			if err != nil {
				return "", err
			}
			cfg := client.NewConfig()
			cfg.DatabaseID = dbID
			return cfg.FormatDSN(), nil
		},
		Proto:    0,
		Opaque:   true,
		Aliases:  []string{},
		Override: "",
	})
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

	// Explorer
	flag.StringVar(&tmpPath, "tmp-path", "", "Explorer temp file path, use os.TempDir for default")
	flag.StringVar(&bgLogLevel, "bg-log-level", "", "Background service log level")

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
	var err error
	// set random
	rand.Seed(time.Now().UnixNano())

	flag.Parse()
	if tmpPath == "" {
		tmpPath = os.TempDir()
	}
	logPath := filepath.Join(tmpPath, "covenant_explorer.log")
	bgLog, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open log file failed: %s, %v", logPath, err)
		os.Exit(-1)
	}
	log.SetOutput(bgLog)
	log.SetStringLevel(bgLogLevel, log.InfoLevel)

	cLog = logrus.New()

	if showVersion {
		fmt.Printf("%v %v %v %v %v\n",
			name, version, runtime.GOOS, runtime.GOARCH, runtime.Version())
		os.Exit(0)
	}
	cLog.Infof("cql build: %#v\n", version)

	configFile = utils.HomeDirExpand(configFile)

	// init covenantsql driver
	if err = client.Init(configFile, []byte(password)); err != nil {
		cLog.WithError(err).Error("init covenantsql client failed")
		os.Exit(-1)
		return
	}

	// TODO(leventeliu): discover more specific confirmation duration from config. We don't have
	// enough informations from config to do that currently, so just use a fixed and long enough
	// duration.
	waitTxConfirmationMaxDuration = 20 * conf.GConf.BPPeriod

	usqlRegister()

	if getBalance {
		var stableCoinBalance, covenantCoinBalance uint64

		if stableCoinBalance, err = client.GetTokenBalance(types.Particle); err != nil {
			cLog.WithError(err).Error("get Particle balance failed")
			return
		}
		if covenantCoinBalance, err = client.GetTokenBalance(types.Wave); err != nil {
			cLog.WithError(err).Error("get Wave balance failed")
			return
		}

		cLog.Infof("Particle balance is: %d", stableCoinBalance)
		cLog.Infof("Wave balance is: %d", covenantCoinBalance)

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
			cLog.Errorf("no such token supporting in CovenantSQL (what we support: %s)",
				strings.Join(values, ", "))
			os.Exit(-1)
			return
		}
		if tokenBalance, err = client.GetTokenBalance(tokenType); err != nil {
			cLog.WithError(err).Error("get token balance failed")
			os.Exit(-1)
			return
		}
		cLog.Infof("%s balance is: %d", tokenType.String(), tokenBalance)
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

		if err := client.Drop(dropDB); err != nil {
			// drop database failed
			cLog.WithField("db", dropDB).WithError(err).Error("drop database failed")
			return
		}

		// drop database success
		cLog.Infof("drop database %#v success", dropDB)
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
				cLog.WithField("db", createDB).Error("create database failed: invalid instance description")
				os.Exit(-1)
				return
			}

			meta = client.ResourceMeta{}
			meta.Node = uint16(nodeCnt)
		}

		dsn, err := client.Create(meta)
		if err != nil {
			cLog.WithError(err).Error("create database failed")
			os.Exit(-1)
			return
		}

		if waitTxConfirmation {
			var ctx, cancel = context.WithTimeout(context.Background(), waitTxConfirmationMaxDuration)
			defer cancel()
			err = client.WaitDBCreation(ctx, dsn)
			if err != nil {
				cLog.WithError(err).Error("create database failed durating creation")
				os.Exit(-1)
				return
			}
		}

		cLog.Infof("the newly created database is: %#v", dsn)
		fmt.Printf(dsn)
		return
	}

	if updatePermission != "" {
		// update user's permission on sqlchain
		var perm userPermission
		if err := json.Unmarshal([]byte(updatePermission), &perm); err != nil {
			cLog.WithError(err).Errorf("update permission failed: invalid permission description")
			os.Exit(-1)
			return
		}

		var permPayload userPermPayload

		if err := json.Unmarshal(perm.Perm, &permPayload); err != nil {
			// try again using role string representation
			if err := json.Unmarshal(perm.Perm, &permPayload.Role); err != nil {
				cLog.WithError(err).Errorf("update permission failed: invalid permission description")
				os.Exit(-1)
				return
			}
		}

		p := &types.UserPermission{
			Role:     permPayload.Role,
			Patterns: permPayload.Patterns,
		}

		if !p.IsValid() {
			cLog.Errorf("update permission failed: invalid permission description")
			os.Exit(-1)
			return
		}

		txHash, err := client.UpdatePermission(perm.TargetUser, perm.TargetChain, p)
		if err != nil {
			cLog.WithError(err).Error("update permission failed")
			os.Exit(-1)
			return
		}

		if waitTxConfirmation {
			wait(txHash)
		}

		cLog.Info("succeed in sending transaction to CovenantSQL")
		return
	}

	if transferToken != "" {
		// transfer token
		var tran tranToken
		if err := json.Unmarshal([]byte(transferToken), &tran); err != nil {
			cLog.WithError(err).Errorf("transfer token failed: invalid transfer description")
			os.Exit(-1)
			return
		}

		var validAmount = regexp.MustCompile(`^([0-9]+) *([a-zA-Z]+)$`)
		if !validAmount.MatchString(tran.Amount) {
			cLog.Error("transfer token failed: invalid transfer description")
			os.Exit(-1)
			return
		}
		amountUnit := validAmount.FindStringSubmatch(tran.Amount)
		if len(amountUnit) != 3 {
			cLog.Error("transfer token failed: invalid transfer description")
			for _, v := range amountUnit {
				cLog.Error(v)
			}
			os.Exit(-1)
			return
		}
		amount, err := strconv.ParseUint(amountUnit[1], 10, 64)
		if err != nil {
			cLog.Error("transfer token failed: invalid token amount")
			os.Exit(-1)
			return
		}
		unit := types.FromString(amountUnit[2])
		if !unit.Listed() {
			cLog.Error("transfer token failed: invalid token type")
			os.Exit(-1)
			return
		}

		var txHash hash.Hash
		txHash, err = client.TransferToken(tran.TargetUser, amount, unit)
		if err != nil {
			cLog.WithError(err).Error("transfer token failed")
			os.Exit(-1)
			return
		}

		if waitTxConfirmation {
			wait(txHash)
		}

		cLog.Info("succeed in sending transaction to CovenantSQL")
		return
	}

	var (
		curUser   *user.User
		available = drivers.Available()
	)
	if st, err := os.Stat("/.dockerenv"); err == nil && !st.IsDir() {
		// in docker, fake user
		var wd string
		if wd, err = os.Getwd(); err != nil {
			cLog.WithError(err).Error("get working directory failed")
			os.Exit(-1)
			return
		}
		curUser = &user.User{
			Uid:      "0",
			Gid:      "0",
			Username: "docker",
			Name:     "docker",
			HomeDir:  wd,
		}
	} else {
		if curUser, err = user.Current(); err != nil {
			cLog.WithError(err).Error("get current user failed")
			os.Exit(-1)
			return
		}
	}

	// run
	err = run(curUser)
	if err != nil && err != io.EOF && err != rline.ErrInterrupt {
		cLog.WithError(err).Error("run cli error")

		if e, ok := err.(*drivers.Error); ok && e.Err == text.ErrDriverNotAvailable {
			bindings := make([]string, 0, len(available))
			for name := range available {
				bindings = append(bindings, name)
			}
			cLog.Infof("available drivers are: %#v", bindings)
		}
		os.Exit(-1)
	}
}

func wait(txHash hash.Hash) {
	var ctx, cancel = context.WithTimeout(context.Background(), waitTxConfirmationMaxDuration)
	defer cancel()
	var state, err = client.WaitTxConfirmation(ctx, txHash)
	cLog.WithFields(logrus.Fields{
		"tx_hash":  txHash,
		"tx_state": state,
	}).WithError(err).Info("wait transaction confirmation")
	if err != nil || state != pi.TransactionStateConfirmed {
		os.Exit(1)
	}
}

func run(u *user.User) (err error) {
	// get working directory
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	// handle variables
	for _, v := range variables.Get() {
		if i := strings.Index(v, "="); i != -1 {
			env.Set(v[:i], v[i+1:])
		} else {
			env.Unset(v)
		}
	}

	// create input/output
	interactive := command != "" || fileName != ""
	l, err := rline.New(interactive, outFile, env.HistoryFile(u))
	if err != nil {
		return err
	}
	defer l.Close()

	// create handler
	h := handler.New(l, u, wd, true)

	// open dsn
	if err = h.Open(dsn); err != nil {
		return err
	}

	// start transaction
	if singleTransaction {
		if h.IO().Interactive() {
			return text.ErrSingleTransactionCannotBeUsedWithInteractiveMode
		}
		if err = h.Begin(); err != nil {
			return err
		}
	}

	// rc file
	if rc := env.RCFile(u); !noRC && rc != "" {
		if err = h.Include(rc, false); err != nil && err != text.ErrNoSuchFileOrDirectory {
			return err
		}
	}

	if command != "" {
		// one liner command
		h.SetSingleLineMode(true)
		h.Reset([]rune(command))
		if err = h.Run(); err != nil && err != io.EOF {
			cLog.WithError(err).Error("run command failed")
			os.Exit(-1)
			return
		}
	} else if fileName != "" {
		// file
		if err = h.Include(fileName, false); err != nil {
			cLog.WithError(err).Error("run file failed")
			os.Exit(-1)
			return
		}
	} else {
		// interactive
		if err = h.Run(); err != nil {
			return
		}

	}

	// commit
	if singleTransaction {
		return h.Commit()
	}

	return nil
}
