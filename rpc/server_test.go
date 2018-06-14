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
	"net"
	"testing"

	"time"

	"os"

	log "github.com/sirupsen/logrus"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/thunderdb/ThunderDB/crypto/asymmetric"
	"github.com/thunderdb/ThunderDB/crypto/kms"
	"github.com/thunderdb/ThunderDB/proto"
	"github.com/thunderdb/ThunderDB/route"
	"github.com/thunderdb/ThunderDB/utils"
)

const PubKeyStorePath = "./public.keystore"

type TestService struct {
	counter int
}

type TestReq struct {
	Step int
}

type TestRep struct {
	Ret int
}

func NewTestService() *TestService {
	return &TestService{
		counter: 0,
	}
}

func (s *TestService) IncCounter(req *TestReq, rep *TestRep) error {
	log.Debugf("calling IncCounter req:%v, rep:%v", *req, *rep)
	s.counter += req.Step
	rep.Ret = s.counter
	return nil
}

func (s *TestService) IncCounterSimpleArgs(step int, ret *int) error {
	log.Debugf("calling IncCounter req:%v, rep:%v", step, ret)
	s.counter += step
	*ret = s.counter
	return nil
}

func TestIncCounter(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	addr := "127.0.0.1:0"
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}

	server, err := NewServerWithService(ServiceMap{"Test": NewTestService()})
	server.SetListener(l)
	go server.Serve()

	rep := new(TestRep)
	client, err := InitClient(l.Addr().String())
	if err != nil {
		log.Fatal(err)
	}

	err = client.Call("Test.IncCounter", &TestReq{Step: 10}, rep)
	if err != nil {
		log.Fatal(err)
	}
	utils.CheckNum(rep.Ret, 10, t)

	err = client.Call("Test.IncCounter", &TestReq{Step: 10}, rep)
	if err != nil {
		log.Fatal(err)
	}
	utils.CheckNum(rep.Ret, 20, t)

	repSimple := new(int)
	err = client.Call("Test.IncCounterSimpleArgs", 10, repSimple)
	if err != nil {
		log.Fatal(err)
	}
	utils.CheckNum(*repSimple, 30, t)

	client.Close()
	server.Stop()
}

func TestIncCounterSimpleArgs(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	addr := "127.0.0.1:0"
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}

	server, err := NewServerWithService(ServiceMap{"Test": NewTestService()})
	server.SetListener(l)
	go server.Serve()

	client, err := InitClient(l.Addr().String())
	if err != nil {
		log.Fatal(err)
	}

	repSimple := new(int)
	err = client.Call("Test.IncCounterSimpleArgs", 10, repSimple)
	if err != nil {
		log.Fatal(err)
	}
	utils.CheckNum(*repSimple, 10, t)

	client.Close()
	server.Stop()
}

func TestEncryptIncCounterSimpleArgs(t *testing.T) {
	defer os.Remove(PubKeyStorePath)
	log.SetLevel(log.DebugLevel)
	addr := "127.0.0.1:0"
	masterKey := []byte("abc")
	server, err := NewServerWithService(ServiceMap{"Test": NewTestService()})
	if err != nil {
		log.Fatal(err)
	}

	route.NewDHTService(PubKeyStorePath, false)
	server.InitRPCServer(addr, "../keys/test.key", masterKey)
	go server.Serve()

	publicKey, err := kms.GetLocalPublicKey()
	nonce := asymmetric.GetPubKeyNonce(publicKey, 10, 100*time.Millisecond, nil)
	serverNodeID := proto.NodeID(nonce.Hash.String())
	kms.SetPublicKey(serverNodeID, nonce.Nonce, publicKey)
	kms.SetLocalNodeIDNonce(nonce.Hash.CloneBytes(), &nonce.Nonce)
	route.SetNodeAddr(serverNodeID, server.Listener.Addr().String())

	cryptoConn, err := DailToNode(serverNodeID)
	client, err := InitClientConn(cryptoConn)
	if err != nil {
		log.Fatal(err)
	}

	repSimple := new(int)
	err = client.Call("Test.IncCounterSimpleArgs", 10, repSimple)
	if err != nil {
		log.Fatal(err)
	}
	utils.CheckNum(*repSimple, 10, t)

	client.Close()
	server.Stop()
}

func TestServer_InitRPCServer(t *testing.T) {

}

func TestEncPingFindValue(t *testing.T) {
	os.Remove(PubKeyStorePath)
	defer os.Remove(PubKeyStorePath)
	log.SetLevel(log.DebugLevel)
	addr := "127.0.0.1:0"
	masterKey := []byte("abc")
	dht, err := route.NewDHTService(PubKeyStorePath, true)

	server, err := NewServerWithService(ServiceMap{"DHT": dht})
	if err != nil {
		log.Fatal(err)
	}

	server.InitRPCServer(addr, "../keys/test.key", masterKey)
	go server.Serve()

	publicKey, err := kms.GetLocalPublicKey()
	nonce := asymmetric.GetPubKeyNonce(publicKey, 10, 100*time.Millisecond, nil)
	serverNodeID := proto.NodeID(nonce.Hash.String())
	kms.SetPublicKey(serverNodeID, nonce.Nonce, publicKey)

	kms.SetLocalNodeIDNonce(nonce.Hash.CloneBytes(), &nonce.Nonce)
	route.SetNodeAddr(serverNodeID, server.Listener.Addr().String())

	cryptoConn, err := DailToNode(serverNodeID)
	client, err := InitClientConn(cryptoConn)
	if err != nil {
		log.Fatal(err)
	}

	proto.NewNodeIDDifficultyTimeout = 100 * time.Millisecond
	node1 := proto.NewNode()
	node1.InitNodeCryptoInfo()
	nodeBytes1, _ := node1.Marshal()

	reqA := &proto.PingReq{
		Node: nodeBytes1,
	}

	respA := new(proto.PingResp)
	err = client.Call("DHT.Ping", reqA, respA)
	if err != nil {
		log.Fatal(err)
	}
	log.Debugf("respA: %v", respA)

	node2 := proto.NewNode()
	node2.InitNodeCryptoInfo()
	nodeBytes2, _ := node2.Marshal()

	reqB := &proto.PingReq{
		Node: nodeBytes2,
	}

	respB := new(proto.PingResp)
	err = client.Call("DHT.Ping", reqB, respB)
	if err != nil {
		log.Fatal(err)
	}
	log.Debugf("respA: %v", respB)

	req := &proto.FindValueReq{
		NodeID: "123",
		Count:  3,
	}
	resp := new(proto.FindValueResp)
	err = client.Call("DHT.FindValue", req, resp)
	if err != nil {
		log.Fatal(err)
	}
	log.Debugf("resp: %v", resp)
	nodeResp1, err := proto.UnmarshalNode(resp.Nodes[0])
	nodeResp2, err := proto.UnmarshalNode(resp.Nodes[1])
	nodeResp3, err := proto.UnmarshalNode(resp.Nodes[2])
	nodeIDList := []string{
		string(nodeResp1.ID),
		string(nodeResp2.ID),
		string(nodeResp3.ID),
	}
	log.Debugf("nodeIDList: %v", nodeIDList)
	Convey("test FindValue", t, func() {
		So(nodeIDList, ShouldContain, string(node1.ID))
		So(nodeIDList, ShouldContain, string(node2.ID))
		So(nodeIDList, ShouldContain, string(kms.BPNodeID))
	})
	client.Close()
	server.Stop()
}

func TestServer_Close(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	addr := "127.0.0.1:0"
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}

	server := NewServer()
	testService := NewTestService()
	err = server.RegisterService("Test", testService)
	if err != nil {
		log.Fatal(err)
	}
	server.SetListener(l)
	go server.Serve()

	server.Stop()
}
