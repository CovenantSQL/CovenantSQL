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

package rpc

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/thunderdb/ThunderDB/conf"
	"gitlab.com/thunderdb/ThunderDB/consistent"
	"gitlab.com/thunderdb/ThunderDB/proto"
	"gitlab.com/thunderdb/ThunderDB/route"
)

const DHTStorePath = "./DHTStore"

func TestCaller_CallNode(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	os.Remove(PubKeyStorePath)
	defer os.Remove(PubKeyStorePath)
	os.Remove(publicKeyStore)
	defer os.Remove(publicKeyStore)

	_, testFile, _, _ := runtime.Caller(0)
	confFile := filepath.Join(filepath.Dir(testFile), "../test/node_0/config.yaml")
	privateKeyPath := filepath.Join(filepath.Dir(testFile), "../test/node_0/private.key")

	conf.GConf, _ = conf.LoadConfig(confFile)
	log.Debugf("GConf: %#v", conf.GConf)
	// reset the once
	route.Once = sync.Once{}
	route.InitKMS(publicKeyStore)

	addr := conf.GConf.ListenAddr // see ../test/node_c/config.yaml
	masterKey := []byte("")
	dht, err := route.NewDHTService(PubKeyStorePath, new(consistent.KMSStorage), true)

	server, err := NewServerWithService(ServiceMap{"DHT": dht})
	if err != nil {
		log.Fatal(err)
	}

	server.InitRPCServer(addr, privateKeyPath, masterKey)
	go server.Serve()

	//publicKey, err := kms.GetLocalPublicKey()
	//nonce := asymmetric.GetPubKeyNonce(publicKey, 10, 100*time.Millisecond, nil)
	//serverNodeID := proto.NodeID(nonce.Hash.String())
	//kms.SetPublicKey(serverNodeID, nonce.Nonce, publicKey)
	//
	//kms.SetLocalNodeIDNonce(nonce.Hash.CloneBytes(), &nonce.Nonce)
	//route.SetNodeAddrCache(&proto.RawNodeID{Hash: nonce.Hash}, server.Listener.Addr().String())

	client := NewCaller()
	node1 := proto.NewNode()
	node1.InitNodeCryptoInfo(100 * time.Millisecond)
	node1.Addr = "1.1.1.1:1"

	reqA := &proto.PingReq{
		Node: *node1,
	}

	respA := new(proto.PingResp)
	err = client.CallNode(conf.GConf.BP.NodeID, "DHT.Ping", reqA, respA)
	if err != nil {
		log.Fatal(err)
	}
	log.Debugf("respA: %v", respA)

	node1addr, err := GetNodeAddr(node1.ID.ToRawNodeID())
	Convey("test GetNodeAddr", t, func() {
		So(err, ShouldBeNil)
		So(node1addr, ShouldEqual, node1.Addr)
	})

	err = client.CallNode(conf.GConf.BP.NodeID, "DHT.Ping", reqA, respA)
	if err != nil {
		log.Fatal(err)
	}
	log.Debugf("respA2: %v", respA)
	server.Stop()
}
