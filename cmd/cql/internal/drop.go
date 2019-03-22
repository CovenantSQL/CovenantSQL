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
	"github.com/CovenantSQL/CovenantSQL/client"
)

// CmdDrop is cql drop command entity.
var CmdDrop = &Command{
	UsageLine: "cql drop [-config file] [-wait-tx-confirm] dsn/dbid",
	Short:     "drop a database by dsn or database id",
	Long: `
Drop command can drop a database by DSN or database id
e.g.
    cql drop covenantsql://the_dsn_of_your_database

Since CovenantSQL is blockchain database, you may want get confirm of drop operation.
e.g.
    cql drop -wait-tx-confirm covenantsql://the_dsn_of_your_database
`,
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
		// not a dsn/dbid
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
