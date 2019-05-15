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
	"expvar"
	"fmt"
	"syscall"
	"time"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	"github.com/CovenantSQL/CovenantSQL/rpc/mux"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

const (
	mwMinerAddr     = "service:miner:addr"
	mwMinerNodeID   = "service:miner:node"
	mwMinerWallet   = "service:miner:wallet"
	mwMinerDiskRoot = "service:miner:disk:root"
)

func initNode() (server *mux.Server, direct *rpc.Server, err error) {
	var masterKey []byte
	if !conf.GConf.UseTestMasterKey {
		// read master key
		fmt.Print("Type in Master key to continue: ")
		masterKey, err = terminal.ReadPassword(syscall.Stdin)
		if err != nil {
			fmt.Printf("Failed to read Master Key: %v", err)
		}
		fmt.Println("")
	}

	if err = kms.InitLocalKeyPair(conf.GConf.PrivateKeyFile, masterKey); err != nil {
		log.WithError(err).Error("init local key pair failed")
		return
	}

	log.Info("init routes")

	// init kms routing
	route.InitKMS(conf.GConf.PubKeyStoreFile)

	err = mux.RegisterNodeToBP(30 * time.Second)
	if err != nil {
		log.Fatalf("register node to BP failed: %v", err)
	}

	// init server
	utils.RemoveAll(conf.GConf.PubKeyStoreFile + "*")
	if server, err = createServer(
		conf.GConf.PrivateKeyFile, masterKey, conf.GConf.ListenAddr); err != nil {
		log.WithError(err).Error("create server failed")
		return
	}
	if direct, err = createDirectServer(
		conf.GConf.PrivateKeyFile, masterKey, conf.GConf.ListenDirectAddr); err != nil {
		log.WithError(err).Error("create direct server failed")
		return
	}

	return
}

func createServer(privateKeyPath string, masterKey []byte, listenAddr string) (server *mux.Server, err error) {
	server = mux.NewServer()
	err = server.InitRPCServer(listenAddr, privateKeyPath, masterKey)
	return
}

func createDirectServer(privateKeyPath string, masterKey []byte, listenAddr string) (server *rpc.Server, err error) {
	if listenAddr == "" {
		return nil, nil
	}
	server = rpc.NewServer()
	err = server.InitRPCServer(listenAddr, privateKeyPath, masterKey)
	return
}

func initMetrics() {
	if conf.GConf != nil {
		expvar.NewString(mwMinerAddr).Set(conf.GConf.ListenAddr)
		expvar.NewString(mwMinerNodeID).Set(string(conf.GConf.ThisNodeID))
		expvar.NewString(mwMinerWallet).Set(conf.GConf.WalletAddress)

		if conf.GConf.Miner != nil {
			expvar.NewString(mwMinerDiskRoot).Set(conf.GConf.Miner.RootDir)
		}
	}
}
