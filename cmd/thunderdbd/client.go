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
	"net"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"gitlab.com/thunderdb/ThunderDB/conf"
	"gitlab.com/thunderdb/ThunderDB/crypto/kms"
	"gitlab.com/thunderdb/ThunderDB/proto"
	"gitlab.com/thunderdb/ThunderDB/rpc"
	"gitlab.com/thunderdb/ThunderDB/utils/log"
	"golang.org/x/crypto/ssh/terminal"
)

func runClient(nodeID proto.NodeID) (err error) {
	var idx int
	for i, n := range (*conf.GConf.KnownNodes)[:] {
		if n.ID == nodeID {
			idx = i
			break
		}
	}

	rootPath := conf.GConf.WorkingRoot
	pubKeyStorePath := filepath.Join(rootPath, conf.GConf.PubKeyStoreFile)
	privateKeyPath := filepath.Join(rootPath, conf.GConf.PrivateKeyFile)

	// read master key
	var masterKey []byte
	if !conf.GConf.IsTestMode {
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

	(*conf.GConf.KnownNodes)[idx].PublicKey, err = kms.GetLocalPublicKey()
	if err != nil {
		log.Errorf("get local public key failed: %s", err)
		return
	}
	//nodeInfo := asymmetric.GetPubKeyNonce(AllNodes[idx].PublicKey, 20, 500*time.Millisecond, nil)
	//log.Debugf("client pubkey:\n%x", AllNodes[idx].PublicKey.Serialize())
	//log.Debugf("client nonce:\n%v", nodeInfo)

	// init nodes
	log.Infof("init peers")
	_, _, _, err = initNodePeers(nodeID, pubKeyStorePath)
	if err != nil {
		return
	}

	// do client request
	if err = clientRequest(clientOperation, flag.Arg(0)); err != nil {
		return
	}

	return
}

func clientRequest(reqType string, sql string) (err error) {
	log.SetLevel(log.DebugLevel)
	leaderNodeID := kms.BP.NodeID
	var conn net.Conn
	var client *rpc.Client

	if len(reqType) > 0 && strings.Title(reqType[:1]) == "P" {
		if conn, err = rpc.DialToNode(leaderNodeID, rpc.GetSessionPoolInstance(), false); err != nil {
			return
		}
		if client, err = rpc.InitClientConn(conn); err != nil {
			return
		}
		reqType = "Ping"
		node1 := proto.NewNode()
		node1.InitNodeCryptoInfo(100 * time.Millisecond)

		reqA := &proto.PingReq{
			Node: *node1,
		}

		respA := new(proto.PingResp)
		log.Debugf("req %s: %v", reqType, reqA)
		err = client.Call("DHT."+reqType, reqA, respA)
		if err != nil {
			log.Fatal(err)
		}
		log.Debugf("resp %s: %v", reqType, respA)
	} else {
		for _, bp := range (*conf.GConf.KnownNodes)[:] {
			if bp.Role == proto.Leader || bp.Role == proto.Follower {
				if conn, err = rpc.DialToNode(bp.ID, rpc.GetSessionPoolInstance(), false); err != nil {
					return
				}
				if client, err = rpc.InitClientConn(conn); err != nil {
					return
				}
				log.Debugf("Calling BP: %s", bp.ID)
				reqType = "FindNeighbor"
				req := &proto.FindNeighborReq{
					NodeID: proto.NodeID(flag.Arg(0)),
					Count:  10,
				}
				resp := new(proto.FindNeighborResp)
				log.Debugf("req %s: %v", reqType, req)
				err = client.Call("DHT."+reqType, req, resp)
				if err != nil {
					log.Fatal(err)
				}
				log.Debugf("resp %s: %v", reqType, resp)
			}
		}
	}

	return
}
