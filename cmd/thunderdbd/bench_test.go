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
	"net"
	"path/filepath"
	"testing"
	"time"

	"gitlab.com/thunderdb/ThunderDB/conf"
	"gitlab.com/thunderdb/ThunderDB/crypto/kms"
	"gitlab.com/thunderdb/ThunderDB/proto"
	"gitlab.com/thunderdb/ThunderDB/rpc"
	"gitlab.com/thunderdb/ThunderDB/utils/log"
)

func BenchmarkKayakKVServer_GetAllNodeInfo(b *testing.B) {
	log.SetLevel(log.DebugLevel)

	var err error
	conf.GConf, err = conf.LoadConfig(configFile)
	if err != nil {
		log.Fatalf("load config from %s failed: %s", configFile, err)
	}
	kms.InitBP()
	log.Debugf("config:\n%#v", conf.GConf)
	conf.GConf.GenerateKeyPair = false

	nodeID := conf.GConf.ThisNodeID

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

	// init nodes
	log.Infof("init peers")
	_, _, _, err = initNodePeers(nodeID, pubKeyStorePath)
	if err != nil {
		return
	}

	//connPool := rpc.newSessionPool(rpc.DefaultDialer)
	//// do client request
	//if err = clientRequest(connPool, clientOperation, ""); err != nil {
	//	return
	//}

	leaderNodeID := kms.BP.NodeID
	var conn net.Conn
	var client *rpc.Client

	//if len(reqType) > 0 && strings.Title(reqType[:1]) == "P" {
	if conn, err = rpc.DialToNode(leaderNodeID, nil); err != nil {
		return
	}
	if client, err = rpc.InitClientConn(conn); err != nil {
		return
	}
	var reqType = "FindNeighbor"
	nodePayload := proto.NewNode()
	nodePayload.InitNodeCryptoInfo(100 * time.Millisecond)

	reqFindNeighbor := &proto.FindNeighborReq{
		NodeID: proto.NodeID(nodePayload.ID),
		Count:  1,
	}
	respFindNeighbor := new(proto.FindNeighborResp)
	log.Debugf("req %s: %v", reqType, reqFindNeighbor)

	b.Run("benchmark "+reqType, func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err = client.Call("DHT."+reqType, reqFindNeighbor, respFindNeighbor)
			if err != nil {
				log.Fatal(err)
			}
			//log.Debugf("resp2 %s: %v", reqType, respA)
		}
	})

	reqType = "Ping"

	reqPing := &proto.PingReq{
		Node: *nodePayload,
	}

	respPing := new(proto.PingResp)

	b.Run("benchmark "+reqType, func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err = client.Call("DHT."+reqType, reqPing, respPing)
			if err != nil {
				log.Fatal(err)
			}
			//log.Debugf("respPing %s: %v", reqType, respPing)
		}
	})
}
