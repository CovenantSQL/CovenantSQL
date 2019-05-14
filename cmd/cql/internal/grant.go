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
	"flag"
	"strings"

	"github.com/CovenantSQL/CovenantSQL/client"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/types"
)

var (
	perm string
)

// CmdGrant is cql grant command entity.
var CmdGrant = &Command{
	UsageLine: "cql grant [common params] [-wait-tx-confirm] [-to-user wallet] [-to-dsn dsn] [-perm perm_struct]",
	Short:     "grant a user's permissions on specific sqlchain",
	Long: `
Grant grants specific permissions for the target user on target dsn.
e.g.
    cql grant -to-user=43602c17adcc96acf2f68964830bb6ebfbca6834961c0eca0915fcc5270e0b40 -to-dsn="covenantsql://xxxx" -perm perm_struct

Since CovenantSQL is built on top of blockchains, you may want to wait for the transaction
confirmation before the permission takes effect.
e.g.
    cql grant -wait-tx-confirm -to-user=43602c17adcc96acf2f68964830bb6ebfbca6834961c0eca0915fcc5270e0b40 -to-dsn="covenantsql://xxxx" -perm perm_struct
`,
	Flag:       flag.NewFlagSet("Grant params", flag.ExitOnError),
	CommonFlag: flag.NewFlagSet("Common params", flag.ExitOnError),
	DebugFlag:  flag.NewFlagSet("Debug params", flag.ExitOnError),
}

func init() {
	CmdGrant.Run = runGrant

	addCommonFlags(CmdGrant)
	addConfigFlag(CmdGrant)
	addWaitFlag(CmdGrant)
	CmdGrant.Flag.StringVar(&toUser, "to-user", "", "Target address of an user account to grant permission.")
	CmdGrant.Flag.StringVar(&toDSN, "to-dsn", "", "Target database dsn to grant permission.")
	CmdGrant.Flag.StringVar(&perm, "perm", "", "Permission type struct for grant.")
}

type userPermPayload struct {
	// User role to access database.
	Role types.UserPermissionRole `json:"role"`
	// SQL pattern regulations for user queries
	// only a fully matched (case-sensitive) sql query is permitted to execute.
	Patterns []string `json:"patterns"`
}

func runGrant(cmd *Command, args []string) {
	commonFlagsInit(cmd)

	if len(args) > 0 || toUser == "" || toDSN == "" || perm == "" {
		ConsoleLog.Error("grant command need to-user, to-dsn address and permission struct as param")
		SetExitStatus(1)
		printCommandHelp(cmd)
		Exit()
	}

	if !strings.HasPrefix(toDSN, client.DBScheme) && !strings.HasPrefix(toDSN, client.DBSchemeAlias) {
		ConsoleLog.Error("grant permission failed: invalid dsn provided, use address start with 'covenantsql://'")
		SetExitStatus(1)
		return
	}
	toDSN = strings.TrimLeft(toDSN, client.DBScheme+"://")
	toDSN = strings.TrimLeft(toDSN, client.DBSchemeAlias+"://")

	targetUserHash, err := hash.NewHashFromStr(toUser)
	if err != nil {
		ConsoleLog.WithError(err).Error("target user address is not valid")
		SetExitStatus(1)
		return
	}
	targetUser := proto.AccountAddress(*targetUserHash)

	targetChainHash, err := hash.NewHashFromStr(toDSN)
	if err != nil {
		ConsoleLog.WithError(err).Error("target dsn address is not valid")
		SetExitStatus(1)
		return
	}
	targetChain := proto.AccountAddress(*targetChainHash)

	var permPayload userPermPayload

	if err := json.Unmarshal([]byte(perm), &permPayload); err != nil {
		// try again using role string representation
		permPayload.Role.FromString(perm)
		// invalid struct will set to void
		if permPayload.Role == types.Void && !strings.EqualFold(perm, "Void") {
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

	configInit()

	txHash, err := client.UpdatePermission(targetUser, targetChain, p)
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

	ConsoleLog.Info("succeed in grant permission on target database")
}
