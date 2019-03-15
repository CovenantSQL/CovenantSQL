package internal

import (
	"github.com/CovenantSQL/CovenantSQL/client"
)

var CmdDrop = &Command{
	UsageLine:   "cql drop [-wait-tx-confirm] [dsn/dbid]",
	Description: "Drop CovenantSQL database by dsn or database id",
}

func init() {
	CmdDrop.Run = runDrop

	addCommonFlags(CmdDrop)
	addWaitFlag(CmdDrop)
}

func runDrop(cmd *Command, args []string) {
	configInit()

	if len(args) != 1 {
		ConsoleLog.Error("Drop command need CovenantSQL dsn or database_id string as param")
		SetExitStatus(1)
		return
	}
	dsn := args[0]

	// drop database
	if _, err := client.ParseDSN(dsn); err != nil {
		// not a dsn
		ConsoleLog.WithField("db", dsn).WithError(err).Error("Not a valid dsn")
		SetExitStatus(1)
		return
	}

	txHash, err := client.Drop(dsn)
	if err != nil {
		// drop database failed
		ConsoleLog.WithField("db", dsn).WithError(err).Error("drop database failed")
		SetExitStatus(1)
		return
	}

	if waitTxConfirmation {
		err = wait(txHash)
		if err != nil {
			ConsoleLog.WithField("db", dsn).WithError(err).Error("drop database failed")
			SetExitStatus(1)
			return
		}
	}

	// drop database success
	ConsoleLog.Infof("drop database %#v success", dsn)
	return
}
