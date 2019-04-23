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
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"

	bp "github.com/CovenantSQL/CovenantSQL/blockproducer"
	pi "github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	"github.com/CovenantSQL/CovenantSQL/crypto"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	rpc "github.com/CovenantSQL/CovenantSQL/rpc/mux"
	"github.com/CovenantSQL/CovenantSQL/sqlchain"
	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/worker"
)

var (
	rpcServiceMap = map[string]interface{}{
		route.DHTRPCName:           &route.DHTService{},
		route.DBRPCName:            &worker.DBMSRPCService{},
		route.SQLChainRPCName:      &sqlchain.MuxService{},
		route.BlockProducerRPCName: &bp.ChainRPCService{},
	}
	rpcName     string
	rpcEndpoint string
	rpcReq      string
)

// CmdRPC is cql rpc command entity.
var CmdRPC = &Command{
	UsageLine: "cql rpc [common params] [-wait-tx-confirm] -name rpc_name -endpoint rpc_endpoint -req rpc_request",
	Short:     "make a rpc request",
	Long: `
RPC makes a RPC request to the target endpoint.
e.g.
    cql rpc -name 'MCC.QuerySQLChainProfile' \
            -endpoint 000000fd2c8f68d54d55d97d0ad06c6c0d91104e4e51a7247f3629cc2a0127cf \
            -req '{"DBID": "c8328272ba9377acdf1ee8e73b17f2b0f7430c798141080d0282195507eb94e7"}'
`,
}

type canSign interface {
	Sign(signer *asymmetric.PrivateKey) error
}

func init() {
	CmdRPC.Run = runRPC

	addCommonFlags(CmdRPC)
	addWaitFlag(CmdRPC)

	CmdRPC.Flag.StringVar(&rpcName, "name", "", "RPC name to do test call")
	CmdRPC.Flag.StringVar(&rpcEndpoint, "endpoint", "", "RPC endpoint Node ID to do test call")
	CmdRPC.Flag.StringVar(&rpcReq, "req", "", "RPC request to do test call, in json format")
}

func runRPC(cmd *Command, args []string) {
	configInit(cmd)

	if rpcEndpoint == "" || rpcName == "" || rpcReq == "" {
		// error
		ConsoleLog.Error("rpc payload is required for rpc tool")
		SetExitStatus(1)
		return
	}

	req, resp := resolveRPCEntities()
	ExitIfErrors()

	if rpcName == route.MCCAddTx.String() {
		// special type of query
		if addTxReqType, ok := req.(*types.AddTxReq); ok {
			addTxReqType.TTL = 1
			addTxReqType.Tx = &pi.TransactionWrapper{}
		}
	}

	// fill the req with request body
	if err := json.Unmarshal([]byte(rpcReq), req); err != nil {
		ConsoleLog.WithError(err).Error("decode request body failed")
		SetExitStatus(1)
		return
	}

	// fill nonce if this is a AddTx request
	var tx pi.Transaction
	if rpcName == route.MCCAddTx.String() {
		if addTxReqType, ok := req.(*types.AddTxReq); ok {
			tx = addTxReqType.Tx
			for {
				if txWrapper, ok := tx.(*pi.TransactionWrapper); ok {
					tx = txWrapper.Unwrap()
				} else {
					break
				}
			}

			// unwrapped tx, find account nonce field and set
			if err := fillTxNonce(tx); err != nil {
				ConsoleLog.WithError(err).Error("fill block producer transaction nonce failed")
				SetExitStatus(1)
				return
			}

			if err := checkAndSign(tx); err != nil {
				ConsoleLog.WithError(err).Error("sign transaction failed")
				SetExitStatus(1)
				return
			}
		}
	}

	// requires signature?
	if err := checkAndSign(req); err != nil {
		ConsoleLog.WithError(err).Error("sign request failed")
		SetExitStatus(1)
		return
	}

	ConsoleLog.Info("sending request")
	spewCfg := spew.NewDefaultConfig()
	spewCfg.MaxDepth = 6
	spewCfg.Dump(req)
	if err := rpc.NewCaller().CallNode(proto.NodeID(rpcEndpoint), rpcName, req, resp); err != nil {
		// send request failed
		ConsoleLog.Infof("call rpc failed: %v\n", err)
		return
	}

	// print the response
	ConsoleLog.Info("got response")
	spewCfg.Dump(resp)

	if rpcName == route.MCCAddTx.String() && waitTxConfirmation {
		ConsoleLog.Info("waiting for transaction confirmation...")
		var (
			err    error
			ticker = time.NewTicker(1 * time.Second)
			req    = &types.QueryTxStateReq{Hash: tx.Hash()}
			resp   = &types.QueryTxStateResp{}
		)
		defer ticker.Stop()
		for {
			if err = rpc.NewCaller().CallNode(
				proto.NodeID(rpcEndpoint),
				route.MCCQueryTxState.String(),
				req, resp,
			); err != nil {
				ConsoleLog.WithError(err).Error("query transaction state failed")
				SetExitStatus(1)
				return
			}
			switch resp.State {
			case pi.TransactionStatePending:
				fmt.Print(".")
			case pi.TransactionStatePacked:
				fmt.Print("+")
			case pi.TransactionStateConfirmed:
				fmt.Print("✔\n")
				return
			case pi.TransactionStateExpired, pi.TransactionStateNotFound:
				fmt.Print("✘\n")
				ConsoleLog.Errorf("bad transaction state: %s", resp.State)
				SetExitStatus(1)
				return
			default:
				fmt.Print("✘\n")
				ConsoleLog.Error("unknown transaction state")
				SetExitStatus(1)
				return
			}
			<-ticker.C
		}
	}
}

func checkAndSign(req interface{}) (err error) {
	if reflect.ValueOf(req).Kind() != reflect.Ptr {
		return checkAndSign(&req)
	}

	if canSignObj, ok := req.(canSign); ok {
		ConsoleLog.Info("signing request")

		var privKey *asymmetric.PrivateKey
		if privKey, err = kms.GetLocalPrivateKey(); err != nil {
			return
		}
		if err = canSignObj.Sign(privKey); err != nil {
			return
		}
	}

	return
}

func nestedWalkFillTxNonce(rv reflect.Value, fieldPath string, signCallback func(fieldPath string, rv reflect.Value) (err error)) (signed bool, err error) {
	rv = reflect.Indirect(rv)

	switch rv.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Float32,
		reflect.Float64, reflect.Complex64, reflect.Complex128, reflect.Chan, reflect.Interface, reflect.Ptr,
		reflect.String, reflect.UnsafePointer, reflect.Func:
		return
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		if _, ok := rv.Interface().(pi.AccountNonce); ok {
			if err = signCallback(fieldPath, rv); err != nil {
				return
			}
			return true, nil
		}
	case reflect.Array, reflect.Slice:
		for i := 0; i != rv.Len(); i++ {
			fieldName := fieldPath
			if fieldPath != "" {
				fieldName += "."
			}
			fieldName += fmt.Sprintf("[%d]", i)
			if signed, err = nestedWalkFillTxNonce(rv.Index(i), fieldName, signCallback); err != nil || signed {
				return
			}
		}
	case reflect.Map:
		for _, k := range rv.MapKeys() {
			fieldName := fieldPath
			if fieldPath != "" {
				fieldName += "."
			}
			fieldName += fmt.Sprintf("[%v]", k)
			if signed, err = nestedWalkFillTxNonce(rv.MapIndex(k), fieldName, signCallback); err != nil || signed {
				return
			}
		}
	case reflect.Struct:
		for i := 0; i != rv.NumField(); i++ {
			// walk inside
			fieldName := fieldPath
			if fieldPath != "" {
				fieldName += "."
			}
			fieldName += rv.Type().Field(i).Name

			if _, ok := rv.Field(i).Interface().(pi.AccountNonce); ok {
				if err = signCallback(fieldName, rv.Field(i)); err != nil {
					return
				}
				return true, nil
			} else if signed, err = nestedWalkFillTxNonce(rv.Field(i), fieldName, signCallback); err != nil || signed {
				return
			}
		}
	}

	return
}

func fillTxNonce(tx pi.Transaction) (err error) {
	_, err = nestedWalkFillTxNonce(reflect.ValueOf(tx), "", func(fieldPath string, rv reflect.Value) (err error) {
		var (
			pubKey      *asymmetric.PublicKey
			accountAddr proto.AccountAddress
		)
		if pubKey, err = kms.GetLocalPublicKey(); err != nil {
			return
		}
		if accountAddr, err = crypto.PubKeyHash(pubKey); err != nil {
			return
		}
		nonceReq := &types.NextAccountNonceReq{
			Addr: accountAddr,
		}
		nonceResp := &types.NextAccountNonceResp{}
		if err = rpc.NewCaller().CallNode(proto.NodeID(rpcEndpoint),
			route.MCCNextAccountNonce.String(), nonceReq, nonceResp); err != nil {
			return
		}

		rv.SetUint(uint64(nonceResp.Nonce))
		ConsoleLog.Infof("filled tx type %s nonce field %s with nonce %d",
			tx.GetTransactionType().String(), fieldPath, nonceResp.Nonce)

		return
	})

	return
}

func resolveRPCEntities() (req interface{}, resp interface{}) {
	rpcParts := strings.SplitN(rpcName, ".", 2)

	if len(rpcParts) != 2 {
		// error rpc name
		ConsoleLog.Errorf("%v is not a valid rpc name\n", rpcName)
		SetExitStatus(1)
		return
	}

	rpcService := rpcParts[0]

	if s, supported := rpcServiceMap[rpcService]; supported {
		typ := reflect.TypeOf(s)

		// traversing methods
		for m := 0; m < typ.NumMethod(); m++ {
			method := typ.Method(m)
			mtype := method.Type

			if method.Name == rpcParts[1] {
				// name matched
				if mtype.PkgPath() != "" || mtype.NumIn() != 3 || mtype.NumOut() != 1 {
					ConsoleLog.Infof("%v is not a valid rpc endpoint method\n", rpcName)
					return
				}

				argType := mtype.In(1)
				replyType := mtype.In(2)

				if argType.Kind() == reflect.Ptr {
					req = reflect.New(argType.Elem()).Interface()
				} else {
					req = reflect.New(argType).Interface()
				}

				resp = reflect.New(replyType.Elem()).Interface()

				return
			}
		}
	}

	// not found
	ConsoleLog.Infof("rpc method %v not found\n", rpcName)
	return
}
