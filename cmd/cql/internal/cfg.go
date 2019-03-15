package internal

import (
	"context"
	"errors"

	pi "github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	"github.com/CovenantSQL/CovenantSQL/client"
	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/sirupsen/logrus"
)

// These are general flags used by console and other commands.
var (
	configFile string
	password   string

	waitTxConfirmation bool // wait for transaction confirmation before exiting

	CmdName string
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
		return
	}

	// TODO(leventeliu): discover more specific confirmation duration from config. We don't have
	// enough informations from config to do that currently, so just use a fixed and long enough
	// duration.
	WaitTxConfirmationMaxDuration = 20 * conf.GConf.BPPeriod
}

func addWaitFlag(cmd *Command) {
	cmd.Flag.BoolVar(&waitTxConfirmation, "wait-tx-confirm", false, "Wait for transaction confirmation")
}

func wait(txHash hash.Hash) (err error) {
	var ctx, cancel = context.WithTimeout(context.Background(), WaitTxConfirmationMaxDuration)
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
