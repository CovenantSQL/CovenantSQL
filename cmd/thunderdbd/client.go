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
	"encoding/json"
	"flag"
	"fmt"
	"path/filepath"
	"strings"
	"syscall"

	log "github.com/sirupsen/logrus"
	"gitlab.com/thunderdb/ThunderDB/crypto/etls"
	"gitlab.com/thunderdb/ThunderDB/crypto/kms"
	"gitlab.com/thunderdb/ThunderDB/kayak"
	"gitlab.com/thunderdb/ThunderDB/rpc"
	"golang.org/x/crypto/ssh/terminal"
)

func runClient() (err error) {
	// load conf
	//log.Infof("load conf")
	//if nodes, peers, err = initNodePeers(confPath); err != nil {
	//	return
	//}
	//
	//var leader *NodeInfo
	//var client *NodeInfo
	//
	//// get leader node
	//for i := range nodes {
	//	if nodes[i].Role == kayak.Leader {
	//		leader = &nodes[i]
	//	} else if nodes[i].Role != kayak.Follower {
	//		client = &nodes[i]
	//	}
	//}
	//
	//// create client key
	//if err = initClientKey(client, leader); err != nil {
	//	return
	//}
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

	// do client request
	if err = clientRequest(clientOperation, flag.Arg(0)); err != nil {
		return
	}

	return
}

func clientRequest(reqType string, sql string) (err error) {
	leaderNodeID := kms.BPNodeID
	var conn *etls.CryptoConn
	if conn, err = rpc.DialToNode(leaderNodeID); err != nil {
		return
	}

	var client *rpc.Client
	if client, err = rpc.InitClientConn(conn); err != nil {
		return
	}

	reqType = strings.Title(strings.ToLower(reqType))

	var rows ResponseRows
	if err = client.Call(fmt.Sprintf("%v.%v", dbServiceName, reqType), sql, &rows); err != nil {
		return
	}
	var res []byte
	if res, err = json.MarshalIndent(rows, "", strings.Repeat(" ", 4)); err != nil {
		return
	}

	fmt.Println(string(res))

	return
}

//
//func initClientKey(client *NodeInfo, leader *NodeInfo) (err error) {
//	clientRootDir := fmt.Sprintf(nodeDirPattern, "client")
//	os.MkdirAll(clientRootDir, 0755)
//
//	// init local key store
//	pubKeyStorePath := filepath.Join(clientRootDir, pubKeyStoreFile)
//	if _, err = consistent.InitConsistent(pubKeyStorePath, true); err != nil {
//		return
//	}
//
//	// init client private key
//	route.InitResolver()
//	privateKeyStorePath := filepath.Join(clientRootDir, privateKeyFile)
//	if err = kms.InitLocalKeyPair(privateKeyStorePath, []byte(privateKeyMasterKey)); err != nil {
//		return
//	}
//
//	kms.SetLocalNodeIDNonce(client.Nonce.Hash.CloneBytes(), &client.Nonce.Nonce)
//
//	// set leader key
//	leaderNodeID := proto.NodeID(leader.Nonce.Hash.String())
//	kms.SetPublicKey(leaderNodeID, leader.Nonce.Nonce, leader.PublicKey)
//
//	// set route to leader
//	route.SetNodeAddr(&proto.RawNodeID{Hash: leader.Nonce.Hash}, fmt.Sprintf(listenAddrPattern, leader.Port))
//
//	return
//}
