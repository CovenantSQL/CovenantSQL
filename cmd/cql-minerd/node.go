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
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/kayak"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh/terminal"
)

func initNode() (server *rpc.Server, err error) {
	var masterKey []byte
	if !conf.GConf.IsTestMode {
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

	err = registerNodeToBP(15 * time.Second)
	if err != nil {
		log.Fatalf("register node to BP failed: %v", err)
	}

	// init server
	if server, err = createServer(
		conf.GConf.PrivateKeyFile, conf.GConf.PubKeyStoreFile, masterKey, conf.GConf.ListenAddr); err != nil {
		log.WithError(err).Error("create server failed")
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

func registerNodeToBP(timeout time.Duration) (err error) {
	// get local node id
	localNodeID, err := kms.GetLocalNodeID()
	if err != nil {
		err = errors.Wrap(err, "register node to BP")
		return
	}

	// get local node info
	localNodeInfo, err := kms.GetNodeInfo(localNodeID)
	if err != nil {
		err = errors.Wrap(err, "register node to BP")
		return
	}

	log.WithField("node", localNodeInfo).Debug("construct local node info")

	pingWaitCh := make(chan proto.NodeID)
	bpNodeIDs := route.GetBPs()
	for _, bpNodeID := range bpNodeIDs {
		go func(ch chan proto.NodeID, id proto.NodeID) {
			for {
				err := rpc.PingBP(localNodeInfo, id)
				if err == nil {
					log.Infof("ping BP succeed: %v", localNodeInfo)
					ch <- id
					return
				}
				if strings.Contains(err.Error(), kayak.ErrNotLeader.Error()) {
					log.Debug("stop ping non leader BP node")
					return
				}

				log.Warnf("ping BP failed: %v", err)
				time.Sleep(3 * time.Second)
			}
		}(pingWaitCh, bpNodeID)
	}

	select {
	case bp := <-pingWaitCh:
		close(pingWaitCh)
		log.WithField("BP", bp).Infof("ping BP succeed")
	case <-time.After(timeout):
		return errors.New("ping BP timeout")
	}

	return
}
