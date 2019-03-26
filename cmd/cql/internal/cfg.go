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
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	pi "github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	"github.com/CovenantSQL/CovenantSQL/client"
	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh/terminal"
)

// These are general flags used by console and other commands.
var (
	configFile string
	password   string
	noPassword bool

	waitTxConfirmation bool // wait for transaction confirmation before exiting
	// Shard chain explorer/adapter stuff
	tmpPath    string // background observer and explorer block and log file path
	bgLogLevel string // background log level
)

func addCommonFlags(cmd *Command) {
	cmd.Flag.StringVar(&configFile, "config", "~/.cql/config.yaml", "Config file for covenantsql (Usually no need to set, default is enough.)")

	// Undocumented, unstable debugging flags.
	cmd.Flag.StringVar(&password, "password", "", "Master key password for covenantsql (NOT SAFE, for debug or script only)")
	cmd.Flag.BoolVar(&noPassword, "no-password", false, "Use empty password for master key")
	cmd.Flag.BoolVar(&asymmetric.BypassSignature, "bypass-signature", false,
		"Disable signature sign and verify, for testing")
}

func configInit() {
	configFile = utils.HomeDirExpand(configFile)

	if password == "" {
		password = readMasterKey(noPassword)
	}

	// init covenantsql driver
	if err := client.Init(configFile, []byte(password)); err != nil {
		ConsoleLog.WithError(err).Error("init covenantsql client failed")
		SetExitStatus(1)
		Exit()
	}

	ConsoleLog.WithField("path", configFile).Info("init config success")

	// TODO(leventeliu): discover more specific confirmation duration from config. We don't have
	// enough informations from config to do that currently, so just use a fixed and long enough
	// duration.
	waitTxConfirmationMaxDuration = 20 * conf.GConf.BPPeriod
}

func addWaitFlag(cmd *Command) {
	cmd.Flag.BoolVar(&waitTxConfirmation, "wait-tx-confirm", false, "Wait for transaction confirmation")
}

func wait(txHash hash.Hash) (err error) {
	var ctx, cancel = context.WithTimeout(context.Background(), waitTxConfirmationMaxDuration)
	defer cancel()
	var state pi.TransactionState
	state, err = client.WaitTxConfirmation(ctx, txHash)
	ConsoleLog.WithFields(logrus.Fields{
		"tx_hash":  txHash,
		"tx_state": state,
	}).WithError(err).Info("wait transaction confirmation")
	if err == nil && state != pi.TransactionStateConfirmed {
		err = errors.New("bad transaction state")
	}
	return
}

func addBgServerFlag(cmd *Command) {
	cmd.Flag.StringVar(&tmpPath, "tmp-path", "", "Background service temp file path, use os.TempDir for default")
	cmd.Flag.StringVar(&bgLogLevel, "bg-log-level", "", "Background service log level")
}

func bgServerInit() {
	if tmpPath == "" {
		tmpPath = os.TempDir()
	}
	logPath := filepath.Join(tmpPath, "covenant_service.log")
	bgLog, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		ConsoleLog.Errorf("open log file failed: %s, %v", logPath, err)
		SetExitStatus(1)
		Exit()
	}

	log.SetOutput(bgLog)
	log.SetStringLevel(bgLogLevel, log.InfoLevel)
}

// readMasterKey reads the password of private key from terminal
func readMasterKey(skip bool) string {
	if skip {
		return ""
	}
	fmt.Println("Enter master key(press Enter for default: \"\"): ")
	bytePwd, err := terminal.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		ConsoleLog.Errorf("read master key failed: %v", err)
		SetExitStatus(1)
		Exit()
	}
	return string(bytePwd)
}

func getPublic() *asymmetric.PublicKey {
	//if config has public, use it
	for _, node := range conf.GConf.KnownNodes {
		if node.ID == conf.GConf.ThisNodeID {
			if node.PublicKey != nil {
				ConsoleLog.Infof("use public key in config file: %s", configFile)
				return node.PublicKey
			}
			break
		}
	}

	//use config specific private key file(already init by configInit())
	ConsoleLog.Infof("generate public key directly from private key: %s", conf.GConf.PrivateKeyFile)
	privateKey, err := kms.LoadPrivateKey(conf.GConf.PrivateKeyFile, []byte(password))
	if err != nil {
		ConsoleLog.WithError(err).Error("load private key file failed")
		SetExitStatus(1)
		ExitIfErrors()
	}
	return privateKey.PubKey()
}
