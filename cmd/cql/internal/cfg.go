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
	"os"
	"path/filepath"

	pi "github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	"github.com/CovenantSQL/CovenantSQL/client"
	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/sirupsen/logrus"
)

// These are general flags used by console and other commands.
var (
	configFile string
	password   string

	waitTxConfirmation bool // wait for transaction confirmation before exiting
	// Shard chain explorer/adapter stuff
	tmpPath    string // background observer and explorer block and log file path
	bgLogLevel string // background log level
)

func addCommonFlags(cmd *Command) {
	cmd.Flag.StringVar(&configFile, "config", "~/.cql/config.yaml", "Config file for covenantsql")
	cmd.Flag.StringVar(&password, "password", "", "Master key password for covenantsql")

	// Undocumented, unstable debugging flags.
	cmd.Flag.BoolVar(&asymmetric.BypassSignature, "bypass-signature", false,
		"Disable signature sign and verify, for testing")
}

func configInit() {
	configFile = utils.HomeDirExpand(configFile)

	// init covenantsql driver
	if err := client.Init(configFile, []byte(password)); err != nil {
		ConsoleLog.WithError(err).Error("init covenantsql client failed")
		SetExitStatus(1)
		Exit()
	}

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
