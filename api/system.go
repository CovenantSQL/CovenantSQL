package api

import (
	"context"

	"github.com/CovenantSQL/CovenantSQL/api/models"
	"github.com/sourcegraph/jsonrpc2"
)

func init() {
	rpc.RegisterMethod("bp_getRunningStatus", bpGetRunningStatus, nil)
}

func bpGetRunningStatus(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (
	result interface{}, err error,
) {
	var model models.SystemModel
	return model.GetRunningStatus()
}
