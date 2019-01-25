package main

import (
	"context"
	"net/http"
	"time"

	"github.com/sourcegraph/jsonrpc2"

	"github.com/CovenantSQL/CovenantSQL/rpc/jsonrpc"
	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/CovenantSQL/CovenantSQL/worker"
)

func startWebsocketAPI(addr string, dbms *worker.DBMS) {
	// TODO: update miner profile to BP

	// register methods
	h := jsonrpc.NewHandler()
	dbmsProxy := &dbmsProxy{dbms}
	h.RegisterMethod("__register", jsonrpc.RegisterClient, jsonrpc.RegisterClientParams{})
	h.RegisterMethod("dbms_query", dbmsProxy.Query, jsonrpc.MsgpackProxyParams{})
	h.RegisterMethod("dbms_ack", dbmsProxy.Ack, jsonrpc.MsgpackProxyParams{})

	server := &jsonrpc.WebsocketServer{
		Server: http.Server{
			Addr:         addr,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 60 * time.Second,
		},
		RPCHandler: h,
	}

	log.WithField("addr", addr).Info("wsapi: start websocket api server")
	go server.Serve()
}

type dbmsProxy struct {
	dbms *worker.DBMS
}

func (p *dbmsProxy) Query(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (
	result interface{}, err error,
) {
	var params = ctx.Value("_params").(*jsonrpc.MsgpackProxyParams)
	rpcReq := new(types.Request)
	if err := params.Unmarshal(rpcReq); err != nil {
		return nil, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidParams,
			Message: err.Error(),
		}
	}
	return p.dbms.Query(rpcReq)
}

func (p *dbmsProxy) Ack(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (
	result interface{}, err error,
) {
	var params = ctx.Value("_params").(*jsonrpc.MsgpackProxyParams)
	rpcReq := new(types.Ack)
	if err := params.Unmarshal(rpcReq); err != nil {
		return nil, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidParams,
			Message: err.Error(),
		}
	}
	return nil, p.dbms.Ack(rpcReq)
}
