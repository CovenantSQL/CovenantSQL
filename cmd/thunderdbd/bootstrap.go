/*
 * Copyright 2018 The ThunderDB Authors.
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
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	log "github.com/sirupsen/logrus"
	"gitlab.com/thunderdb/ThunderDB/crypto/kms"
	"gitlab.com/thunderdb/ThunderDB/kayak"
	ka "gitlab.com/thunderdb/ThunderDB/kayak/api"
	kt "gitlab.com/thunderdb/ThunderDB/kayak/transport"
	"gitlab.com/thunderdb/ThunderDB/rpc"
	"gitlab.com/thunderdb/ThunderDB/twopc"
	"golang.org/x/crypto/ssh/terminal"
)

const (
	listenHost          = "127.0.0.1"
	listenAddrPattern   = listenHost + ":%v"
	nodeDirPattern      = "./node_%v"
	pubKeyStoreFile     = "public.keystore"
	privateKeyFile      = "private.key"
	privateKeyMasterKey = "abc"
	kayakServiceName    = "kayak"
	dbServiceName       = "Database"
	dbFileName          = "storage.db"
)

var (
	genConf         bool
	clientMode      bool
	nodeOffset      int
	clientOperation string
	payloadCodec    PayloadCodec
)

func init() {
	flag.BoolVar(&genConf, "generate", false, "run conf generation")
	flag.BoolVar(&clientMode, "client", false, "run as client")
	flag.IntVar(&nodeOffset, "nodeOffset", 0, "node offset in peers conf")
	flag.StringVar(&clientOperation, "operation", "read", "client operation")
}

func main() {
	log.SetLevel(log.DebugLevel)
	flag.Parse()

	if clientMode {
		if err := runClient(); err != nil {
			log.Fatalf("run client failed: %v", err.Error())
		} else {
			log.Infof("run client success")
		}
		return
	}

	if err := runNode(nodeOffset); err != nil {
		log.Fatalf("run kayak failed: %v", err.Error())
	}
}

func runNode(idx int) (err error) {
	rootPath := fmt.Sprintf(nodeDirPattern, idx)
	pubKeyStorePath := filepath.Join(rootPath, pubKeyStoreFile)
	privateKeyPath := filepath.Join(rootPath, privateKeyFile)
	dbFile := filepath.Join(rootPath, dbFileName)

	// read master key
	fmt.Print("Type in Master key to continue: ")
	masterKey, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		fmt.Printf("Failed to read Master Key: %v", err)
	}
	fmt.Println("")

	err = kms.InitLocalKeyPair(privateKeyPath, masterKey)
	if err != nil {
		log.Errorf("init local key pair failed: %s", err)
		return
	}

	// init nodes
	log.Infof("init peers")
	nodes, peers, err := initNodePeers(idx, pubKeyStorePath)
	if err != nil {
		return
	}

	var service *kt.ETLSTransportService
	var server *rpc.Server

	// create server
	log.Infof("create server")
	if service, server, err = createServer(privateKeyPath, pubKeyStorePath, masterKey, (*nodes)[idx].Addr); err != nil {
		return
	}

	// init storage
	log.Infof("init storage")
	var st *Storage
	if st, err = initStorage(dbFile); err != nil {
		return
	}

	// init kayak
	log.Infof("init kayak runtime")
	var kayakRuntime *kayak.Runtime
	if _, kayakRuntime, err = initKayakTwoPC(rootPath, &(*nodes)[idx], peers, st, service); err != nil {
		return
	}

	// register service rpc
	server.RegisterService(dbServiceName, &StubServer{
		Runtime: kayakRuntime,
		Storage: st,
	})

	// start server
	server.Serve()

	return
}

func createServer(privateKeyPath, pubKeyStorePath string, masterKey []byte, listenAddr string) (service *kt.ETLSTransportService, server *rpc.Server, err error) {
	os.Remove(pubKeyStorePath)
	//var dht *route.DHTService
	//if dht, err = route.NewDHTService(pubKeyStorePath, true); err != nil {
	//	return
	//}

	//server, err = rpc.NewServerWithService(rpc.ServiceMap{
	//	"DHT": dht,
	//})
	server = rpc.NewServer()
	if err != nil {
		return
	}

	err = server.InitRPCServer(listenAddr, privateKeyPath, masterKey)
	service = ka.NewMuxService(kayakServiceName, server)

	return
}

func initKayakTwoPC(rootDir string, node *NodeInfo, peers *kayak.Peers, worker twopc.Worker, service *kt.ETLSTransportService) (config kayak.Config, runtime *kayak.Runtime, err error) {
	// create kayak config
	log.Infof("create twopc config")
	config = ka.NewTwoPCConfig(rootDir, service, worker)

	// create kayak runtime
	log.Infof("create kayak runtime")
	runtime, err = ka.NewTwoPCKayak(peers, config)
	if err != nil {
		return
	}

	// init runtime
	log.Infof("init kayak twopc runtime")
	err = runtime.Init()

	return
}
