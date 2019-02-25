package api

import (
	"context"

	"github.com/sourcegraph/jsonrpc2"

	"github.com/CovenantSQL/CovenantSQL/rpc/jsonrpc"
	"github.com/CovenantSQL/CovenantSQL/types"
)

func init() {
	rpc.RegisterMethod("bp_addTx", bpAddTx, jsonrpc.MsgpackProxyParams{})
}

func bpAddTx(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (
	result interface{}, err error,
) {
	var params = ctx.Value("_params").(*jsonrpc.MsgpackProxyParams)
	reqAddTx := new(types.AddTxReq)
	if err := params.Unmarshal(req); err != nil {
		return nil, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidParams,
			Message: err.Error(),
		}
	}

	return nil, server.ChainRPC.AddTx(reqAddTx, nil)
}
