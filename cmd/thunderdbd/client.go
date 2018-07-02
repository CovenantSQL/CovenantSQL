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

	log "github.com/sirupsen/logrus"
	"gitlab.com/thunderdb/ThunderDB/crypto/kms"
	"gitlab.com/thunderdb/ThunderDB/kayak"
	"gitlab.com/thunderdb/ThunderDB/proto"
	"gitlab.com/thunderdb/ThunderDB/rpc"
	"golang.org/x/crypto/ssh/terminal"
)

func runClient() (err error) {
	var idx = -1
	for i, n := range AllNodes[:] {
		if n.Role == kayak.Client {
			idx = i
			break
		}
	}

	rootPath := fmt.Sprintf(nodeDirPattern, "c")
	pubKeyStorePath := filepath.Join(rootPath, pubKeyStoreFile)
	privateKeyPath := filepath.Join(rootPath, privateKeyFile)

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

	AllNodes[idx].PublicKey, err = kms.GetLocalPublicKey()
	if err != nil {
		log.Errorf("get local public key failed: %s", err)
		return
	}
	//nodeInfo := asymmetric.GetPubKeyNonce(AllNodes[idx].PublicKey, 20, 500*time.Millisecond, nil)
	//log.Debugf("client pubkey:\n%x", AllNodes[idx].PublicKey.Serialize())
	//log.Debugf("client nonce:\n%v", nodeInfo)

	// init nodes
	log.Infof("init peers")
	_, _, err = initNodePeers(idx, pubKeyStorePath)
	if err != nil {
		return
	}

	connPool := rpc.NewSessionPool(rpc.DefaultDialer)
	// do client request
	if err = clientRequest(connPool, clientOperation, flag.Arg(0)); err != nil {
		return
	}

	return
}

func clientRequest(connPool *rpc.SessionPool, reqType string, sql string) (err error) {
	log.SetLevel(log.DebugLevel)
	leaderNodeID := kms.BP.NodeID
	var conn net.Conn
	var client *rpc.Client

	if len(reqType) > 0 && strings.Title(reqType[:1]) == "P" {
		if conn, err = rpc.DialToNode(leaderNodeID, connPool); err != nil {
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
		for _, bp := range AllNodes[:] {
			if bp.Role == kayak.Leader || bp.Role == kayak.Follower {
				if conn, err = rpc.DialToNode(bp.ID, connPool); err != nil {
					return
				}
				if client, err = rpc.InitClientConn(conn); err != nil {
					return
				}
				log.Debugf("Calling BP: %s", bp.ID)
				reqType = "FindValue"
				req := &proto.FindValueReq{
					NodeID: proto.NodeID(flag.Arg(0)),
					Count:  10,
				}
				resp := new(proto.FindValueResp)
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
