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
	"bufio"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"
	"os"
	"os/signal"
	"path"
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
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/CovenantSQL/CovenantSQL/worker"

	"golang.org/x/crypto/ssh/terminal"
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
	workingRoot    string
	isTestNetAddr  bool
	rpcServiceMap  = map[string]interface{}{
		"DHT":  &route.DHTService{},
		"DBS":  &worker.DBMSRPCService{},
		"BPDB": &blockproducer.DBService{},
		"SQLC": &sqlchain.MuxService{},
	}
)

func init() {
	log.SetLevel(log.ErrorLevel)

	flag.StringVar(&tool, "tool", "", "tool type, miner, keygen, keytool, rpc, nonce, confgen, addrgen")
	flag.StringVar(&publicKeyHex, "public", "", "public key hex string to mine node id/nonce")
	flag.StringVar(&privateKeyFile, "private", "private.key", "private key file to generate/show")
	flag.IntVar(&difficulty, "difficulty", 24, "difficulty for miner to mine nodes and generating nonce")
	flag.StringVar(&rpcName, "rpc", "", "rpc name to do test call")
	flag.StringVar(&rpcEndpoint, "endpoint", "", "rpc endpoint to do test call")
	flag.StringVar(&rpcReq, "req", "", "rpc request to do test call, in json format")
	flag.StringVar(&configFile, "config", "", "rpc config file")
	flag.StringVar(&workingRoot, "root", "node", "confgen root is the working root directory containing all auto-generating keys and certifications")
	flag.BoolVar(&isTestNetAddr, "addrgen", false, "addrgen generates a testnet address from your key pair")
}

func main() {
	log.Infof("idminer build: %s\n", version)
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
	case "nonce":
		runNonce()
	case "confgen":
		if workingRoot == "" {
			log.Error("root directory is required for confgen")
			os.Exit(1)
		}
		runConfgen()
	case "addrgen":
		if privateKeyFile == "" && publicKeyHex == "" {
			log.Error("privateKey path or publicKey hex is required for addrgen")
			os.Exit(1)
		}
		runAddrgen()
	default:
		flag.Usage()
		os.Exit(1)
	}
}

func runMiner() {
	masterKey, err := readMasterKey()
	if err != nil {
		fmt.Printf("read master key failed: %v\n", err)
		os.Exit(1)
	}

	var publicKey *asymmetric.PublicKey

	if publicKeyHex != "" {
		publicKeyBytes, err := hex.DecodeString(publicKeyHex)
		if err != nil {
			log.Fatalf("error converting hex: %s\n", err)
		}
		publicKey, err = asymmetric.ParsePubKey(publicKeyBytes)
		if err != nil {
			log.Fatalf("error converting public key: %s\n", err)
		}
	} else if privateKeyFile != "" {
		privateKey, err := kms.LoadPrivateKey(privateKeyFile, []byte(masterKey))
		if err != nil {
			log.Fatalf("load private key file faile: %v\n", err)
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
	log.Infof("cpu: %d\n", cpuCount)
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
			log.Infof("miner #%d start: %v\n", i, start)
			miner.ComputeBlockNonce(block, start, difficulty)
		}(i)
	}

	sig := <-signalCh
	log.Infof("received signal %s\n", sig)
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
	log.Infof("verify result: %v\n", kms.IsIDPubNonceValid(&proto.RawNodeID{Hash: max.Hash}, &max.Nonce, publicKey))

	// print result
	fmt.Printf("nonce: %v\n", max)
	fmt.Printf("node id: %v\n", max.Hash.String())
}

func runKeygen() *asymmetric.PublicKey {
	if _, err := os.Stat(privateKeyFile); err == nil {
		reader := bufio.NewReader(os.Stdin)
		fmt.Println("Private key file has already existed. \nDo you want to delete it? (y or n, press Enter for default n):")
		t, err := reader.ReadString('\n')
		t = strings.Trim(t, "\n")
		if err != nil {
			log.Errorf("Unexpected error: %v\n", err)
			os.Exit(1)
		}
		if strings.Compare(t, "y") == 0 || strings.Compare(t, "yes") == 0 {
			err = os.Remove(privateKeyFile)
			if err != nil {
				log.Errorf("Unexpected error: %v\n", err)
				os.Exit(1)
			}
		} else {
			os.Exit(0)
		}
	}

	privateKey, _, err := asymmetric.GenSecp256k1KeyPair()
	if err != nil {
		log.Fatalf("generate key pair failed: %v\n", err)
	}

	masterKey, err := readMasterKey()
	if err != nil {
		log.Fatalf("read master key failed: %v\n", err)
	}

	if err = kms.SavePrivateKey(privateKeyFile, privateKey, []byte(masterKey)); err != nil {
		log.Fatalf("save generated keypair failed: %v\n", err)
	}

	fmt.Printf("Private key file: %s\n", privateKeyFile)
	fmt.Printf("Public key's hex: %s\n", hex.EncodeToString(privateKey.PubKey().Serialize()))
	return privateKey.PubKey()
}

func runKeytool() {
	masterKey, err := readMasterKey()
	if err != nil {
		fmt.Printf("read master key failed: %v\n", err)
		os.Exit(1)
	}

	privateKey, err := kms.LoadPrivateKey(privateKeyFile, []byte(masterKey))
	if err != nil {
		log.Errorf("load private key failed: %v\n", err)
	}

	fmt.Printf("Public key's hex: %s\n", hex.EncodeToString(privateKey.PubKey().Serialize()))
}

func runRPC() {
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

func runNonce() {
	var publicKey *asymmetric.PublicKey

	if publicKeyHex != "" {
		publicKeyBytes, err := hex.DecodeString(publicKeyHex)
		if err != nil {
			log.Fatalf("error converting hex: %s\n", err)
		}
		publicKey, err = asymmetric.ParsePubKey(publicKeyBytes)
		if err != nil {
			log.Fatalf("error converting public key: %s\n", err)
		}
	} else if privateKeyFile != "" {
		masterKey, err := readMasterKey()
		if err != nil {
			fmt.Printf("read master key failed: %v\n", err)
			os.Exit(1)
		}
		privateKey, err := kms.LoadPrivateKey(privateKeyFile, []byte(masterKey))
		if err != nil {
			log.Fatalf("load private key file fail: %v\n", err)
		}
		publicKey = privateKey.PubKey()
	} else {
		log.Fatalln("can neither convert public key nor load private key")
	}

	noncegen(publicKey)
}

func noncegen(publicKey *asymmetric.PublicKey) *mine.NonceInfo {
	publicKeyBytes := publicKey.Serialize()

	cpuCount := runtime.NumCPU()
	log.Infof("cpu: %d\n", cpuCount)
	stopCh := make(chan struct{})
	nonceCh := make(chan mine.NonceInfo)

	rand.Seed(time.Now().UnixNano())
	step := 256 / cpuCount
	for i := 0; i < cpuCount; i++ {
		go func(i int) {
			startBit := i * step
			position := startBit / 64
			shift := uint(startBit % 64)
			log.Infof("position: %d, shift: %d, i: %d", position, shift, i)
			var start mine.Uint256
			if position == 0 {
				start = mine.Uint256{A: uint64(1<<shift) + uint64(rand.Uint32())}
			} else if position == 1 {
				start = mine.Uint256{B: uint64(1<<shift) + uint64(rand.Uint32())}
			} else if position == 2 {
				start = mine.Uint256{C: uint64(1<<shift) + uint64(rand.Uint32())}
			} else if position == 3 {
				start = mine.Uint256{D: uint64(1<<shift) + uint64(rand.Uint32())}
			}

			for j := start; ; j.Inc() {
				select {
				case <-stopCh:
					break
				default:
					currentHash := mine.HashBlock(publicKeyBytes, j)
					currentDifficulty := currentHash.Difficulty()
					if currentDifficulty >= difficulty {
						nonce := mine.NonceInfo{
							Nonce:      j,
							Difficulty: currentDifficulty,
							Hash:       currentHash,
						}
						nonceCh <- nonce
					}
				}
			}
		}(i)
	}

	nonce := <-nonceCh
	close(stopCh)

	// verify result
	if !kms.IsIDPubNonceValid(&proto.RawNodeID{Hash: nonce.Hash}, &nonce.Nonce, publicKey) {
		log.Fatalf("nonce: %v\nnode id: %s", nonce, nonce.Hash.String())
	}

	// print result
	fmt.Printf("nonce: %v\n", nonce)
	fmt.Printf("node id: %v\n", nonce.Hash.String())

	return &nonce
}

func runConfgen() {
	privateKeyFileName := "private.key"
	publicKeystoreFileName := "public.keystore"

	privateKeyFile = path.Join(workingRoot, privateKeyFileName)

	if _, err := os.Stat(workingRoot); err == nil {
		reader := bufio.NewReader(os.Stdin)
		fmt.Println("The directory has already existed. \nDo you want to delete it? (y or n, press Enter for default n):")
		t, err := reader.ReadString('\n')
		t = strings.Trim(t, "\n")
		if err != nil {
			log.Errorf("Unexpected error: %v\n", err)
			os.Exit(1)
		}
		if strings.Compare(t, "y") == 0 || strings.Compare(t, "yes") == 0 {
			os.RemoveAll(workingRoot)
		} else {
			os.Exit(0)
		}
	}

	err := os.Mkdir(workingRoot, 0755)
	if err != nil {
		log.Errorf("Unexpected error: %v", err)
	}

	fmt.Println("Generating key pair...")
	publicKey := runKeygen()
	fmt.Println("Generated key pair.")

	fmt.Println("Generating nonce...")
	nonce := noncegen(publicKey)
	fmt.Println("Generated nonce.")

	fmt.Println("Generating config file...")

	configContent := fmt.Sprintf(`IsTestMode: true
WorkingRoot: "./"
PrivateKeyFile: "%s"
PubKeyStoreFile: "%s"
DHTFileName: "dht.db"
ListenAddr: "0.0.0.0:4661"
ThisNodeID: %s
MinNodeIDDifficulty: 24
BlockProducer:
  PublicKey: 034b4319f2e2a9d9f3fd55d1233ff7a2f2ea2e815e7227b3861b4a6a24a8d62697
  NodeID: 0000011839f464418166658ef6dec09ea68da1619a7a9e0f247f16e0d6c6504d
  Nonce:
    a: 761802
    b: 0
    c: 0
    d: 4611686019290328603
  ChainFileName: "chain.db"
  BPGenesisInfo:
    Version: 1
    BlockHash: f745ca6427237aac858dd3c7f2df8e6f3c18d0f1c164e07a1c6b8eebeba6b154
    Producer: 0000000000000000000000000000000000000000000000000000000000000001
    MerkleRoot: 0000000000000000000000000000000000000000000000000000000000000001
    ParentHash: 0000000000000000000000000000000000000000000000000000000000000001
    Timestamp: 2018-09-01T00:00:00Z
    BaseAccounts:
      - Address: d3dce44e0a4f1dae79b93f04ce13fb5ab719059f7409d7ca899d4c921da70129
        StableCoinBalance: 100000000
        CovenantCoinBalance: 100000000
KnownNodes:
- ID: 0000011839f464418166658ef6dec09ea68da1619a7a9e0f247f16e0d6c6504d
  Nonce:
    a: 761802
    b: 0
    c: 0
    d: 4611686019290328603
  Addr: 120.79.254.36:11105
  PublicKey: 034b4319f2e2a9d9f3fd55d1233ff7a2f2ea2e815e7227b3861b4a6a24a8d62697
  Role: Leader
- ID: 00000177647ade3bd86a085510113ccae4b8e690424bb99b95b3545039ae8e8c
  Nonce:
    a: 197619
    b: 0
    c: 0
    d: 4611686019249700888
  Addr: 120.79.254.36:11106
  PublicKey: 02d6f3afcd26aa8de25f5d088c5f8d6b052b4ad1b27ce5b84939bc9f105556844e
  Role: Miner
- ID: 000004b0267f959e645b0df5cd38ae0652c1160b960cdcb97b322caafe627e4f
  Nonce:
    a: 455820
    b: 0
    c: 0
    d: 3627017019
  Addr: 120.79.254.36:11107
  PublicKey: 034b4319f2e2a9d9f3fd55d1233ff7a2f2ea2e815e7227b3861b4a6a24a8d62697
  Role: Follower
- ID: 00000328ef30233890f61d7504b640b45e8ba33d5671157a0cee81745e46b963
  Nonce:
    a: 333847
    b: 0
    c: 0
    d: 6917529031239958890
  Addr: 120.79.254.36:11108
  PublicKey: 0202361b87a087cd61137ba3b5bd83c48c180566c8d7f1a0b386c3277bf0dc6ebd
  Role: Miner
- ID: %s
  Nonce:
    a: %d
    b: %d
    c: %d
    d: %d
  Addr: 127.0.0.1:11109
  PublicKey: %s
  Role: Client
`, privateKeyFileName, publicKeystoreFileName,
		nonce.Hash.String(), nonce.Hash.String(),
		nonce.Nonce.A, nonce.Nonce.B, nonce.Nonce.C, nonce.Nonce.D, hex.EncodeToString(publicKey.Serialize()))

	err = ioutil.WriteFile(path.Join(workingRoot, "config.yaml"), []byte(configContent), 0755)
	if err != nil {
		log.Errorf("Unexpected error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Generated nonce.")
}

func runAddrgen() {
	var publicKey *asymmetric.PublicKey

	if publicKeyHex != "" {
		publicKeyBytes, err := hex.DecodeString(publicKeyHex)
		if err != nil {
			log.Fatalf("error converting hex: %s\n", err)
		}
		publicKey, err = asymmetric.ParsePubKey(publicKeyBytes)
		if err != nil {
			log.Fatalf("error converting public key: %s\n", err)
		}
	} else if privateKeyFile != "" {
		masterKey, err := readMasterKey()
		if err != nil {
			fmt.Printf("read master key failed: %v\n", err)
			os.Exit(1)
		}
		privateKey, err := kms.LoadPrivateKey(privateKeyFile, []byte(masterKey))
		if err != nil {
			log.Fatalf("load private key file fail: %v\n", err)
		}
		publicKey = privateKey.PubKey()
	} else {
		fmt.Println("privateKey path or publicKey hex is required for addrgen")
		os.Exit(1)
	}

	addr, err := utils.PubKey2Addr(publicKey, utils.TestNet)
	if err != nil {
		log.Fatalf("unexpected error: %v\n", err)
	}
	fmt.Printf("wallet address: %s\n", addr)
}

func readMasterKey() (string, error) {
	fmt.Println("Enter master key(press Enter for default: \"\"): ")
	bytePwd, err := terminal.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	return string(bytePwd), err
}
