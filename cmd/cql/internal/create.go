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
	"encoding/json"
	"fmt"

	"github.com/CovenantSQL/CovenantSQL/client"
)

// CmdCreate is cql create command entity.
var CmdCreate = &Command{
	UsageLine: "cql create [common params] [-wait-tx-confirm] db_meta_json",
	Short:     "create a database",
	Long: `
Create creates a CovenantSQL database by database meta info JSON string. The meta info must include
node count.
e.g.
    cql create '{"node": 2}'

A complete introduction of db_meta_json fields：

    target-miners          []string // List of target miner addresses
    node                   int      // Target node number
    space                  int      // Minimum disk space requirement, 0 for none
    memory                 int      // Minimum memory requirement, 0 for none
    load-avg-per-cpu       float    // Minimum idle CPU requirement, 0 for none
    encrypt-key            string   // Encryption key for persistence data
    eventual-consistency   bool     // Use eventual consistency to sync among miner nodes
    consistency-level      float    // Consistency level, node*consistency_level is the node number to perform strong consistency
    isolation-level        int      // Isolation level in a single node

Since CovenantSQL is built on top of blockchains, you may want to wait for the transaction
confirmation before the creation takes effect.
e.g.
    cql create -wait-tx-confirm '{"node": 2}'
`,
}

func init() {
	CmdCreate.Run = runCreate

	addCommonFlags(CmdCreate)
	addWaitFlag(CmdCreate)
}

func runCreate(cmd *Command, args []string) {
	configInit(cmd)

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
	fmt.Println(dsn)
}
