package main

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"time"

	"github.com/pkg/errors"
	"github.com/sourcegraph/jsonrpc2"

	"github.com/CovenantSQL/CovenantSQL/crypto"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/rpc/jsonrpc"
	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/CovenantSQL/CovenantSQL/worker"
)

func startWebsocketAPI(addr string, dbms *worker.DBMS) {
	// TODO: update miner profile to BP

	// register methods
	h := jsonrpc.NewHandler()
	dbmsProxy := &dbmsProxy{dbms}
	h.RegisterMethod("__register", registerClient, registerClientParams{})
	h.RegisterMethod("dbms_query", dbmsProxy.Query, dbmsProxyParams{})
	h.RegisterMethod("dbms_ack", dbmsProxy.Ack, dbmsProxyParams{})

	server := &jsonrpc.WebsocketServer{
		Addr:         addr,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		Handler:      h,
	}

	log.WithField("addr", addr).Info("wsapi: start websocket api server")
	go server.Serve()
}

type registerClientParams struct {
	Address   string `json:"address"`
	PublicKey string `json:"public_key"`
}

// registerClient MUST be the first method to called by client after a successful connection
func registerClient(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (
	result interface{}, err error,
) {
	params := ctx.Value("_params").(*registerClientParams)

	hexPubKey, err := hex.DecodeString(params.PublicKey)
	if err != nil {
		return nil, errors.WithMessage(err, "invalid hex format of public key")
	}

	pubKey := &asymmetric.PublicKey{}
	if err := pubKey.UnmarshalBinary(hexPubKey); err != nil {
		return nil, errors.WithMessage(err, "invalid public key: unmarshal binary error")
	}

	expectedAddr, err := crypto.PubKeyHash(pubKey)
	if err != nil {
		return nil, errors.WithMessage(err, "invalid public key: generate address error")
	}
	if expectedAddr.String() != params.Address {
		return nil, errors.New("invalid pair of address and public key")
	}

	nodeInfo := &proto.Node{
		ID:        proto.NodeID(params.Address), // a fake node id
		Role:      proto.Client,
		Addr:      params.Address,
		PublicKey: pubKey,
	}

	if err := kms.SetNode(nodeInfo); err != nil {
		return nil, errors.WithMessage(err, "kms: update client node info failed")
	}

	return nil, nil
}

type dbmsProxy struct {
	dbms *worker.DBMS
}

type dbmsProxyParams struct {
	Payload string `json:"payload"`
}

func (p *dbmsProxyParams) Unmarshal(params interface{}) error {
	// base64 -> bytes
	bs, err := base64.StdEncoding.DecodeString(p.Payload)
	if err != nil {
		return errors.WithMessage(err, "invalid base64 format")
	}

	// bytes (msgpack) -> object
	return utils.DecodeMsgPack(bs, params)
}

func (p *dbmsProxy) Query(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (
	result interface{}, err error,
) {
	var params = ctx.Value("_params").(*dbmsProxyParams)
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
	var params = ctx.Value("_params").(*dbmsProxyParams)
	rpcReq := new(types.Ack)
	if err := params.Unmarshal(rpcReq); err != nil {
		return nil, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidParams,
			Message: err.Error(),
		}
	}
	return nil, p.dbms.Ack(rpcReq)
}
