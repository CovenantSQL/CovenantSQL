package internal

import (
	"encoding/json"

	"github.com/CovenantSQL/CovenantSQL/client"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/types"
)

var CmdPermission = &Command{
	UsageLine:   "cql permission [-wait-tx-confirm] [perm_meta]",
	Description: "Update user's permission on specific sqlchain",
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
