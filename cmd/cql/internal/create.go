package internal

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/CovenantSQL/CovenantSQL/client"
)

// CmdCreate is cql create command entity.
var CmdCreate = &Command{
	UsageLine: "cql create [-wait-tx-confirm] [dbmeta]",
	Description: `
Create CovenantSQL database by database metainfo json string(must include node count)
	e.g.
		cql create -wait-tx-confirm '{"node":2}'
`,
}

func init() {
	CmdCreate.Run = runCreate

	addCommonFlags(CmdCreate)
	addWaitFlag(CmdCreate)
}

func runCreate(cmd *Command, args []string) {
	configInit()

	if len(args) != 1 {
		ConsoleLog.Error("Create command need database_meta_info string as params")
		SetExitStatus(1)
		return
	}
	metaStr := args[0]
	// create database
	// parse instance requirement
	var meta client.ResourceMeta

	if err := json.Unmarshal([]byte(metaStr), &meta); err != nil {
		ConsoleLog.WithField("db", metaStr).Error("create database failed: invalid instance description")
		SetExitStatus(1)
		return
	}

	if meta.Node == 0 {
		ConsoleLog.WithField("db", metaStr).Error("create database failed: request node count must > 1")
		SetExitStatus(1)
		return
	}

	txHash, dsn, err := client.Create(meta)
	if err != nil {
		ConsoleLog.WithError(err).Error("create database failed")
		SetExitStatus(1)
		return
	}

	if waitTxConfirmation {
		wait(txHash)
		var ctx, cancel = context.WithTimeout(context.Background(), waitTxConfirmationMaxDuration)
		defer cancel()
		err = client.WaitDBCreation(ctx, dsn)
		if err != nil {
			ConsoleLog.WithError(err).Error("create database failed durating creation")
			SetExitStatus(1)
			return
		}
	}

	ConsoleLog.Infof("the newly created database is: %#v", dsn)
	fmt.Printf(dsn)
}
