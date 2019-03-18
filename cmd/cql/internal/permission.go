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
	"encoding/json"

	"github.com/CovenantSQL/CovenantSQL/client"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/types"
)

// CmdPermission is cql permission command entity.
var CmdPermission = &Command{
	UsageLine: "cql permission [-config file] [-password masterkey] [-wait-tx-confirm] [perm_meta]",
	Short:     "update user's permission on specific sqlchain",
	Long: `
Permission command can give a user specific permissions on your database
e.g.
	cql permission '{"chain":"your_chain_addr","user":"user_addr","perm":"perm_struct"}'

Since CovenantSQL is blockchain database, you may want get confirm of permission update.
e.g.
	cql permission -wait-tx-confirm '{"chain":"your_chain_addr","user":"user_addr","perm":"perm_struct"}'
`,
}

func init() {
	CmdPermission.Run = runPermission

	addCommonFlags(CmdPermission)
	addWaitFlag(CmdPermission)
}

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

func runPermission(cmd *Command, args []string) {
	configInit()

	if len(args) != 1 {
		ConsoleLog.Error("Permission command need CovenantSQL perm_meta json string as param")
		SetExitStatus(1)
		return
	}

	updatePermission := args[0]
	// update user's permission on sqlchain
	var perm userPermission
	if err := json.Unmarshal([]byte(updatePermission), &perm); err != nil {
		ConsoleLog.WithError(err).Errorf("update permission failed: invalid permission description")
		SetExitStatus(1)
		return
	}

	var permPayload userPermPayload

	if err := json.Unmarshal(perm.Perm, &permPayload); err != nil {
		// try again using role string representation
		if err := json.Unmarshal(perm.Perm, &permPayload.Role); err != nil {
			ConsoleLog.WithError(err).Errorf("update permission failed: invalid permission description")
			SetExitStatus(1)
			return
		}
	}

	p := &types.UserPermission{
		Role:     permPayload.Role,
		Patterns: permPayload.Patterns,
	}

	if !p.IsValid() {
		ConsoleLog.Errorf("update permission failed: invalid permission description")
		SetExitStatus(1)
		return
	}

	txHash, err := client.UpdatePermission(perm.TargetUser, perm.TargetChain, p)
	if err != nil {
		ConsoleLog.WithError(err).Error("update permission failed")
		SetExitStatus(1)
		return
	}

	if waitTxConfirmation {
		err = wait(txHash)
		if err != nil {
			SetExitStatus(1)
			return
		}
	}

	ConsoleLog.Info("succeed in sending transaction to CovenantSQL")
}
