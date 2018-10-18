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
	"os"
	"reflect"
	"strings"

	"github.com/CovenantSQL/CovenantSQL/blockproducer"
	bp "github.com/CovenantSQL/CovenantSQL/blockproducer"
	"github.com/CovenantSQL/CovenantSQL/client"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	"github.com/CovenantSQL/CovenantSQL/sqlchain"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/CovenantSQL/CovenantSQL/worker"
)

var (
	rpcServiceMap = map[string]interface{}{
		route.DHTRPCName:           &route.DHTService{},
		route.DBRPCName:            &worker.DBMSRPCService{},
		route.BPDBRPCName:          &blockproducer.DBService{},
		route.SQLChainRPCName:      &sqlchain.MuxService{},
		route.BlockProducerRPCName: &bp.ChainRPCService{},
	}
	rpcName     string
	rpcEndpoint string
	rpcReq      string
)

func init() {
	flag.StringVar(&rpcName, "rpc", "", "rpc name to do test call")
	flag.StringVar(&rpcEndpoint, "rpc-endpoint", "", "rpc endpoint to do test call")
	flag.StringVar(&rpcReq, "rpc-req", "", "rpc request to do test call, in json format")
}

func runRPC() {
	if configFile == "" {
		// error
		log.Error("config file path is required for rpc tool")
		os.Exit(1)
	}
	if rpcEndpoint == "" || rpcName == "" || rpcReq == "" {
		// error
		log.Error("rpc payload is required for rpc tool")
		os.Exit(1)
	}

	if err := client.Init(configFile, []byte("")); err != nil {
		fmt.Printf("init rpc client failed: %v\n", err)
		os.Exit(1)
		return
	}

	req, resp := resolveRPCEntities()

	// fill the req with request body
	if err := json.Unmarshal([]byte(rpcReq), req); err != nil {
		fmt.Printf("decode request body failed: %v\n", err)
		os.Exit(1)
		return
	}

	if err := rpc.NewCaller().CallNode(proto.NodeID(rpcEndpoint), rpcName, req, resp); err != nil {
		// send request failed
		fmt.Printf("call rpc failed: %v\n", err)
		os.Exit(1)
		return
	}

	// print the response
	if resBytes, err := json.MarshalIndent(resp, "", "  "); err != nil {
		fmt.Printf("marshal response failed: %v\n", err)
		os.Exit(1)
	} else {
		fmt.Println(string(resBytes))
	}
}

func resolveRPCEntities() (req interface{}, resp interface{}) {
	rpcParts := strings.SplitN(rpcName, ".", 2)

	if len(rpcParts) != 2 {
		// error rpc name
		fmt.Printf("%v is not a valid rpc name\n", rpcName)
		os.Exit(1)
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
					fmt.Printf("%v is not a valid rpc endpoint method\n", rpcName)
					os.Exit(1)
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
	fmt.Printf("rpc method %v not found\n", rpcName)
	os.Exit(1)

	return
}
