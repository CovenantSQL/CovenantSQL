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
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"math/rand"
	"os"
	"os/signal"
	"reflect"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/CovenantSQL/CovenantSQL/blockproducer"
	"github.com/CovenantSQL/CovenantSQL/client"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	mine "github.com/CovenantSQL/CovenantSQL/pow/cpuminer"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	"github.com/CovenantSQL/CovenantSQL/sqlchain"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/CovenantSQL/CovenantSQL/worker"
)

var (
	version        = "unknown"
	tool           string
	publicKeyHex   string
	privateKeyFile string
	difficulty     int
	rpcName        string
	rpcEndpoint    string
	rpcReq         string
	configFile     string
	rpcServiceMap  = map[string]interface{}{
		"DHT":  &route.DHTService{},
		"DBS":  &worker.DBMSRPCService{},
		"BPDB": &blockproducer.DBService{},
		"SQLC": &sqlchain.MuxService{},
	}
)

func init() {
	flag.StringVar(&tool, "tool", "miner", "tool type, miner, keygen, keytool, rpc")
	flag.StringVar(&publicKeyHex, "public", "", "public key hex string to mine node id/nonce")
	flag.StringVar(&privateKeyFile, "private", "", "private key file to generate/show")
	flag.IntVar(&difficulty, "difficulty", 256, "difficulty for miner to mine nodes")
	flag.StringVar(&rpcName, "rpc", "", "rpc name to do test call")
	flag.StringVar(&rpcEndpoint, "endpoint", "", "rpc endpoint to do test call")
	flag.StringVar(&rpcReq, "req", "", "rpc request to do test call, in json format")
	flag.StringVar(&configFile, "config", "", "rpc config file")
}

func main() {
	log.Infof("idminer build: %s", version)
	flag.Parse()

	switch tool {
	case "miner":
		if publicKeyHex == "" && privateKeyFile == "" {
			// error
			log.Error("publicKey or privateKey is required in miner mode")
			os.Exit(1)
		}
		runMiner()
	case "keygen":
		if privateKeyFile == "" {
			// error
			log.Error("privateKey path is required for keygen")
			os.Exit(1)
		}
		runKeygen()
	case "keytool":
		if privateKeyFile == "" {
			// error
			log.Error("privateKey path is required for keytool")
			os.Exit(1)
		}
		runKeytool()
	case "rpc":
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
		runRPC()
	default:
		flag.Usage()
		os.Exit(1)
	}
}

func runMiner() {
	var publicKey *asymmetric.PublicKey

	if publicKeyHex != "" {
		publicKeyBytes, err := hex.DecodeString(publicKeyHex)
		if err != nil {
			log.Fatalf("error converting hex: %s", err)
		}
		publicKey, err = asymmetric.ParsePubKey(publicKeyBytes)
		if err != nil {
			log.Fatalf("error converting public key: %s", err)
		}
	} else if privateKeyFile != "" {
		privateKey, err := kms.LoadPrivateKey(privateKeyFile, []byte(""))
		if err != nil {
			log.Fatalf("load private key file faile: %v", err)
		}
		publicKey = privateKey.PubKey()
	}

	signalCh := make(chan os.Signal, 1)
	signal.Notify(
		signalCh,
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	signal.Ignore(syscall.SIGHUP, syscall.SIGTTIN, syscall.SIGTTOU)

	cpuCount := runtime.NumCPU()
	log.Infof("cpu: %d", cpuCount)
	nonceChs := make([]chan mine.NonceInfo, cpuCount)
	stopChs := make([]chan struct{}, cpuCount)

	rand.Seed(time.Now().UnixNano())
	step := math.MaxUint64 / uint64(cpuCount)

	for i := 0; i < cpuCount; i++ {
		nonceChs[i] = make(chan mine.NonceInfo)
		stopChs[i] = make(chan struct{})
		go func(i int) {
			miner := mine.NewCPUMiner(stopChs[i])
			nonceCh := nonceChs[i]
			block := mine.MiningBlock{
				Data:      publicKey.Serialize(),
				NonceChan: nonceCh,
				Stop:      nil,
			}
			start := mine.Uint256{D: step*uint64(i) + uint64(rand.Uint32())}
			log.Infof("miner #%d start: %v", i, start)
			miner.ComputeBlockNonce(block, start, difficulty)
		}(i)
	}

	sig := <-signalCh
	log.Infof("received signal %s", sig)
	for i := 0; i < cpuCount; i++ {
		close(stopChs[i])
	}

	max := mine.NonceInfo{}
	for i := 0; i < cpuCount; i++ {
		newNonce := <-nonceChs[i]
		if max.Difficulty < newNonce.Difficulty {
			max = newNonce
		}
	}

	// verify result
	log.Infof("verify result: %v", kms.IsIDPubNonceValid(&proto.RawNodeID{Hash: max.Hash}, &max.Nonce, publicKey))
	log.Infof("nonce: %v", max)
}

func runKeygen() {
	os.Remove(privateKeyFile)
	privateKey, _, err := asymmetric.GenSecp256k1KeyPair()
	if err != nil {
		log.Fatalf("generate key pair failed: %v", err)
	}

	if err = kms.SavePrivateKey(privateKeyFile, privateKey, []byte("")); err != nil {
		log.Fatalf("save generated keypair failed: %v", err)
	}

	log.Infof("pubkey hex is: %s", hex.EncodeToString(privateKey.PubKey().Serialize()))
}

func runKeytool() {
	privateKey, err := kms.LoadPrivateKey(privateKeyFile, []byte(""))
	if err != nil {
		log.Fatalf("load private key failed: %v", err)
	}

	log.Infof("pubkey hex is: %s", hex.EncodeToString(privateKey.PubKey().Serialize()))
}

func runRPC() {
	if err := client.Init(configFile, []byte("")); err != nil {
		log.Fatalf("init rpc client failed: %v", err)
		os.Exit(-1)
		return
	}

	req, resp := resolveRPCEntities()

	// fill the req with request body
	if err := json.Unmarshal([]byte(rpcReq), req); err != nil {
		log.Fatalf("decode request body failed: %v", err)
		os.Exit(-1)
		return
	}

	if err := rpc.NewCaller().CallNode(proto.NodeID(rpcEndpoint), rpcName, req, resp); err != nil {
		// send request failed
		log.Fatalf("call rpc failed: %v", err)
		os.Exit(-1)
		return
	}

	// print the response
	if resBytes, err := json.MarshalIndent(resp, "", "  "); err != nil {
		log.Fatalf("marshal response failed: %v", err)
		os.Exit(-1)
		return
	} else {
		fmt.Println(string(resBytes))
	}
}

func resolveRPCEntities() (req interface{}, resp interface{}) {
	rpcParts := strings.SplitN(rpcName, ".", 2)

	if len(rpcParts) != 2 {
		// error rpc name
		log.Fatalf("%v is not a valid rpc name", rpcName)
		os.Exit(-1)
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
					log.Fatalf("%v is not a valid rpc endpoint method", rpcName)
					os.Exit(-1)
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
	log.Fatalf("rpc method %v not found", rpcName)
	os.Exit(-1)

	return
}
