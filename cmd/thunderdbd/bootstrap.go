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
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	bp "gitlab.com/thunderdb/ThunderDB/blockproducer"
	"gitlab.com/thunderdb/ThunderDB/conf"
	"gitlab.com/thunderdb/ThunderDB/crypto/kms"
	"gitlab.com/thunderdb/ThunderDB/kayak"
	ka "gitlab.com/thunderdb/ThunderDB/kayak/api"
	kt "gitlab.com/thunderdb/ThunderDB/kayak/transport"
	"gitlab.com/thunderdb/ThunderDB/metric"
	"gitlab.com/thunderdb/ThunderDB/proto"
	"gitlab.com/thunderdb/ThunderDB/route"
	"gitlab.com/thunderdb/ThunderDB/rpc"
	"gitlab.com/thunderdb/ThunderDB/twopc"
	"gitlab.com/thunderdb/ThunderDB/utils/log"
	"golang.org/x/crypto/ssh/terminal"
)

const (
	//nodeDirPattern   = "./node_%v"
	//pubKeyStoreFile  = "public.keystore"
	//privateKeyFile   = "private.key"
	//dhtFileName      = "dht.db"
	kayakServiceName = "Kayak"
	dhtServiceName   = "DHT"
)

func runNode(nodeID proto.NodeID, listenAddr string) (err error) {
	rootPath := conf.GConf.WorkingRoot
	pubKeyStorePath := filepath.Join(rootPath, conf.GConf.PubKeyStoreFile)
	privateKeyPath := filepath.Join(rootPath, conf.GConf.PrivateKeyFile)
	dbFile := filepath.Join(rootPath, conf.GConf.DHTFileName)

	var masterKey []byte
	if !conf.GConf.IsTestMode {
		// read master key
		fmt.Print("Type in Master key to continue: ")
		masterKey, err = terminal.ReadPassword(int(syscall.Stdin))
		if err != nil {
			fmt.Printf("Failed to read Master Key: %v", err)
		}
		fmt.Println("")
	}

	err = kms.InitLocalKeyPair(privateKeyPath, masterKey)
	if err != nil {
		log.Errorf("init local key pair failed: %s", err)
		return
	}

	// init nodes
	log.Infof("init peers")
	_, peers, thisNode, err := initNodePeers(nodeID, pubKeyStorePath)
	if err != nil {
		log.Errorf("init nodes and peers failed: %s", err)
		return
	}

	var service *kt.ETLSTransportService
	var server *rpc.Server

	// create server
	log.Infof("create server")
	if service, server, err = createServer(privateKeyPath, pubKeyStorePath, masterKey, listenAddr); err != nil {
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
	if _, kayakRuntime, err = initKayakTwoPC(rootPath, thisNode, peers, st, service); err != nil {
		log.Errorf("init kayak runtime failed: %s", err)
		return
	}

	// init kayak and consistent
	log.Infof("init kayak and consistent runtime")
	kvServer := &KayakKVServer{
		Runtime: kayakRuntime,
		Storage: st,
	}
	dht, err := route.NewDHTService(dbFile, kvServer, false)
	if err != nil {
		log.Errorf("init consistent hash failed: %s", err)
		return
	}

	// set consistent handler to kayak storage
	kvServer.Storage.consistent = dht.Consistent

	// register service rpc
	log.Infof("register dht service rpc")
	err = server.RegisterService(dhtServiceName, dht)
	if err != nil {
		log.Errorf("register dht service failed: %s", err)
		return
	}

	// init metrics
	metricService := metric.NewCollectServer()
	if err = server.RegisterService(metric.MetricServiceName, metricService); err != nil {
		log.Errorf("init metric service failed: %v", err)
		return
	}

	// init block producer database service
	var dbService *bp.DBService
	if dbService, err = initDBService(kvServer, metricService); err != nil {
		log.Errorf("init block producer db service failed: %v", err)
		return
	}
	if err = server.RegisterService(bp.DBServiceName, dbService); err != nil {
		log.Error("init block producer db service failed: %v", err)
		return
	}

	// ping all known nodes
	go periodicPingBlockProducer()

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

func initKayakTwoPC(rootDir string, node *conf.NodeInfo, peers *kayak.Peers, worker twopc.Worker, service *kt.ETLSTransportService) (config kayak.Config, runtime *kayak.Runtime, err error) {
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

func initDBService(kvServer *KayakKVServer, metricService *metric.CollectServer) (dbService *bp.DBService, err error) {
	var serviceMap *bp.DBServiceMap
	if serviceMap, err = bp.InitServiceMap(kvServer); err != nil {
		log.Errorf("init bp database service map failed")
		return
	}

	dbService = &bp.DBService{
		AllocationRounds: bp.DefaultAllocationRounds, //
		ServiceMap:       serviceMap,
		Consistent:       kvServer.Storage.consistent,
		NodeMetrics:      &metricService.NodeMetric,
	}

	return
}

func periodicPingBlockProducer() {
	var localNodeID proto.NodeID
	var err error

	// get local node id
	if localNodeID, err = kms.GetLocalNodeID(); err != nil {
		return
	}

	// get local node info
	var localNodeInfo *proto.Node
	if localNodeInfo, err = kms.GetNodeInfo(localNodeID); err != nil {
		return
	}

	log.Debugf("construct local node info: %v", localNodeInfo)

	go func() {
		for {
			select {
			case <-time.After(time.Second):
			}

			// send ping requests to block producer
			bpNodeIDs := route.GetBPs()

			for _, bpNodeID := range bpNodeIDs {
				rpc.PingBP(localNodeInfo, bpNodeID)
			}
		}
	}()
}
