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
	"gitlab.com/thunderdb/ThunderDB/route"
	"gitlab.com/thunderdb/ThunderDB/rpc"
	"gitlab.com/thunderdb/ThunderDB/twopc"
	"golang.org/x/crypto/ssh/terminal"
)

const (
	nodeDirPattern   = "./node_%v"
	pubKeyStoreFile  = "public.keystore"
	privateKeyFile   = "private.key"
	kayakServiceName = "kayak"
	dhtServiceName   = "DHT"
	dhtFileName      = "dht.db"
)

var (
	genConf         bool
	clientMode      bool
	nodeOffset      int
	clientOperation string
)

func init() {
	flag.BoolVar(&genConf, "generate", false, "run conf generation")
	flag.BoolVar(&clientMode, "client", false, "run as client")
	flag.IntVar(&nodeOffset, "nodeOffset", 0, "node offset in peers conf")
	flag.StringVar(&clientOperation, "operation", "FindValue", "client operation")
}

func runNode(idx int) (err error) {
	rootPath := fmt.Sprintf(nodeDirPattern, idx)
	pubKeyStorePath := filepath.Join(rootPath, pubKeyStoreFile)
	privateKeyPath := filepath.Join(rootPath, privateKeyFile)
	dbFile := filepath.Join(rootPath, dhtFileName)

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
		log.Errorf("init nodes and peers failed: %s", err)
		return
	}

	var service *kt.ETLSTransportService
	var server *rpc.Server

	// create server
	log.Infof("create server")
	if service, server, err = createServer(privateKeyPath, pubKeyStorePath, masterKey, (*nodes)[idx].Addr); err != nil {
		log.Errorf("create server failed: %s", err)
		return
	}

	// init storage
	log.Infof("init storage")
	var st *LocalStorage
	if st, err = initStorage(dbFile); err != nil {
		log.Errorf("init storage failed: %s", err)
		return
	}

	// init kayak
	log.Infof("init kayak runtime")
	var kayakRuntime *kayak.Runtime
	if _, kayakRuntime, err = initKayakTwoPC(rootPath, &(*nodes)[idx], peers, st, service); err != nil {
		log.Errorf("init kayak runtime failed: %s", err)
		return
	}

	// init kayak and consistent
	log.Infof("init kayak and consistent runtime")
	kayak := &KayakKVServer{
		Runtime: kayakRuntime,
		Storage: st,
	}
	dht, err := route.NewDHTService(dbFile, kayak, false)
	if err != nil {
		log.Errorf("init consistent hash failed: %s", err)
		return
	}

	// set consistent handler to kayak storage
	kayak.Storage.consistent = dht.Consistent

	// register service rpc
	log.Infof("register dht service rpc")
	err = server.RegisterService(dhtServiceName, dht)
	if err != nil {
		log.Errorf("register dht service failed: %s", err)
		return
	}

	// start server
	server.Serve()

	return
}

func createServer(privateKeyPath, pubKeyStorePath string, masterKey []byte, listenAddr string) (service *kt.ETLSTransportService, server *rpc.Server, err error) {
	os.Remove(pubKeyStorePath)

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
