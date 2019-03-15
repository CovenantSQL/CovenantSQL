package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/CovenantSQL/CovenantSQL/client"
)

var CmdCreate = &Command{
	UsageLine:   "cql create [-wait-tx-confirm] [dbmeta/nodecount]",
	Description: "Create CovenantSQL database by database metainfo or just number for node count",
}

func init() {
	CmdCreate.Run = runCreate

	addCommonFlags(CmdCreate)
	addWaitFlag(CmdCreate)
}

func runCreate(cmd *Command, args []string) {
	configInit()

	if len(args) != 1 {
		ConsoleLog.Error("Create command need database_meta_info string or node_count as params")
		SetExitStatus(1)
		return
	}
	metaStr := args[0]
	// create database
	// parse instance requirement
	var meta client.ResourceMeta

	if err := json.Unmarshal([]byte(metaStr), &meta); err != nil {
		// not a instance json, try if it is a number describing node count
		nodeCnt, err := strconv.ParseUint(metaStr, 10, 16)
		if err != nil {
			// still failing
			ConsoleLog.WithField("db", metaStr).Error("create database failed: invalid instance description")
			SetExitStatus(1)
			return
		}

		meta = client.ResourceMeta{}
		meta.Node = uint16(nodeCnt)
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
