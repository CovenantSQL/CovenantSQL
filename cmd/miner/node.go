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

	log "github.com/sirupsen/logrus"
	"gitlab.com/thunderdb/ThunderDB/conf"
	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
	"gitlab.com/thunderdb/ThunderDB/crypto/kms"
	"gitlab.com/thunderdb/ThunderDB/proto"
	"gitlab.com/thunderdb/ThunderDB/route"
	"gitlab.com/thunderdb/ThunderDB/rpc"
	"golang.org/x/crypto/ssh/terminal"
)

func initNode() (server *rpc.Server, err error) {
	kms.InitBP()
	keyPairRootPath := conf.GConf.WorkingRoot
	pubKeyPath := filepath.Join(keyPairRootPath, conf.GConf.PubKeyStoreFile)
	privKeyPath := filepath.Join(keyPairRootPath, conf.GConf.PrivateKeyFile)

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

	if err = kms.InitLocalKeyPair(privKeyPath, masterKey); err != nil {
		log.Errorf("init local key pair failed: %s", err)
		return
	}

	log.Infof("init routes")

	route.InitResolver()
	kms.InitPublicKeyStore(pubKeyPath, nil)

	// set known nodes
	for _, p := range (*conf.GConf.KnownNodes)[:] {
		var rawNodeIDHash *hash.Hash
		rawNodeIDHash, err = hash.NewHashFromStr(string(p.ID))
		if err != nil {
			log.Errorf("load hash from node id failed: %s", err)
			return
		}
		log.Debugf("set node addr: %v, %v", rawNodeIDHash, p.Addr)
		rawNodeID := &proto.RawNodeID{Hash: *rawNodeIDHash}
		route.SetNodeAddrCache(rawNodeID, p.Addr)
		node := &proto.Node{
			ID:        p.ID,
			Addr:      p.Addr,
			PublicKey: p.PublicKey,
			Nonce:     p.Nonce,
		}
		err = kms.SetNode(node)
		if err != nil {
			log.Errorf("set node failed: %v\n %s", node, err)
		}
		if p.ID == conf.GConf.ThisNodeID {
			kms.SetLocalNodeIDNonce(rawNodeID.CloneBytes(), &p.Nonce)
		}
	}

	// init server
	if server, err = createServer(privKeyPath, pubKeyPath, masterKey, conf.GConf.ListenAddr); err != nil {
		log.Errorf("create server failed: %v", err)
		return
	}

	return
}

func createServer(privateKeyPath, pubKeyStorePath string, masterKey []byte, listenAddr string) (server *rpc.Server, err error) {
	os.Remove(pubKeyStorePath)

	server = rpc.NewServer()
	if err != nil {
		return
	}

	err = server.InitRPCServer(listenAddr, privateKeyPath, masterKey)

	return
}
