/*
 * Copyright 2018 The CovenantSQL Authors.
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

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"

	bp "github.com/CovenantSQL/CovenantSQL/blockproducer"
	pi "github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	"github.com/CovenantSQL/CovenantSQL/client"
	"github.com/CovenantSQL/CovenantSQL/crypto"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	"github.com/CovenantSQL/CovenantSQL/sqlchain"
	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/CovenantSQL/CovenantSQL/worker"
)

var (
	rpcServiceMap = map[string]interface{}{
		route.DHTRPCName:           &route.DHTService{},
		route.DBRPCName:            &worker.DBMSRPCService{},
		route.SQLChainRPCName:      &sqlchain.MuxService{},
		route.BlockProducerRPCName: &bp.ChainRPCService{},
	}
	rpcName          string
	rpcEndpoint      string
	rpcReq           string
	rpcTxWaitConfirm bool
)

type canSign interface {
	Sign(signer *asymmetric.PrivateKey) error
}

func init() {
	flag.StringVar(&rpcName, "rpc", "", "rpc name to do test call")
	flag.StringVar(&rpcEndpoint, "rpc-endpoint", "", "rpc endpoint to do test call")
	flag.StringVar(&rpcReq, "rpc-req", "", "rpc request to do test call, in json format")
	flag.BoolVar(&rpcTxWaitConfirm, "rpc-tx-wait-confirm", false, "wait for transaction confirmation")
}

func runRPC() {
	if configFile == "" {
		// error
		log.Fatal("config file path is required for rpc tool")
		return
	}
	if rpcEndpoint == "" || rpcName == "" || rpcReq == "" {
		// error
		log.Fatal("rpc payload is required for rpc tool")
		return
	}

	if err := client.Init(configFile, []byte("")); err != nil {
		log.Fatalf("init rpc client failed: %v\n", err)
		return
	}

	req, resp := resolveRPCEntities()

	if rpcName == route.MCCAddTx.String() {
		// special type of query
		if addTxReqType, ok := req.(*types.AddTxReq); ok {
			addTxReqType.TTL = 1
			addTxReqType.Tx = &pi.TransactionWrapper{}
		}
	}

	// fill the req with request body
	if err := json.Unmarshal([]byte(rpcReq), req); err != nil {
		log.Fatalf("decode request body failed: %v\n", err)
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
				log.Fatalf("fill block producer transaction nonce failed: %v\n", err)
				return
			}

			if err := checkAndSign(tx); err != nil {
				log.Fatalf("sign transaction failed: %v\n", err)
				return
			}
		}
	}

	// requires signature?
	if err := checkAndSign(req); err != nil {
		log.Fatalf("sign request failed: %v\n", err)
		return
	}

	log.Info("sending request")
	spewCfg := spew.NewDefaultConfig()
	spewCfg.MaxDepth = 6
	spewCfg.Dump(req)
	if err := rpc.NewCaller().CallNode(proto.NodeID(rpcEndpoint), rpcName, req, resp); err != nil {
		// send request failed
		log.Infof("call rpc failed: %v\n", err)
		return
	}

	// print the response
	log.Info("got response")
	spewCfg.Dump(resp)

	if rpcName == route.MCCAddTx.String() && rpcTxWaitConfirm {
		log.Info("waiting for transaction confirmation...")
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
				log.Fatalf("query transaction state failed: %v", err)
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
				log.Fatalf("bad transaction state: %s", resp.State)
			default:
				fmt.Print("✘\n")
				log.Fatal("unknown transaction state")
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
		log.Info("signing request")

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
		log.Infof("filled tx type %s nonce field %s with nonce %d",
			tx.GetTransactionType().String(), fieldPath, nonceResp.Nonce)

		return
	})

	return
}

func resolveRPCEntities() (req interface{}, resp interface{}) {
	rpcParts := strings.SplitN(rpcName, ".", 2)

	if len(rpcParts) != 2 {
		// error rpc name
		log.Fatalf("%v is not a valid rpc name\n", rpcName)
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
					log.Infof("%v is not a valid rpc endpoint method\n", rpcName)
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
	log.Infof("rpc method %v not found\n", rpcName)
	return
}
