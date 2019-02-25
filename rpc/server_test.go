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

package rpc

import (
	"net"
	"os"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/CovenantSQL/CovenantSQL/consistent"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

const PubKeyStorePath = "./public.keystore"

// CheckNum make int assertion.
func CheckNum(num, expected int, t *testing.T) {
	if num != expected {
		t.Errorf("got %d, expected %d", num, expected)
	}
}

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
	log.WithFields(log.Fields{
		"req":   *req,
		"reply": *rep,
	}).Debug("calling IncCounter")
	s.counter += req.Step
	rep.Ret = s.counter
	return nil
}

func (s *TestService) IncCounterSimpleArgs(step int, ret *int) error {
	log.WithFields(log.Fields{
		"req":   step,
		"reply": ret,
	}).Debug("calling IncCounter")
	s.counter += step
	*ret = s.counter
	return nil
}

func TestIncCounter(t *testing.T) {
	log.SetLevel(log.FatalLevel)
	addr := "127.0.0.1:0"
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}

	server, err := NewServerWithService(ServiceMap{"Test": NewTestService()})
	server.SetListener(l)
	go server.Serve()

	rep := new(TestRep)
	client, err := initClient(l.Addr().String())
	if err != nil {
		log.Fatal(err)
	}

	err = client.Call("Test.IncCounter", &TestReq{Step: 10}, rep)
	if err != nil {
		log.Fatal(err)
	}
	CheckNum(rep.Ret, 10, t)

	err = client.Call("Test.IncCounter", &TestReq{Step: 10}, rep)
	if err != nil {
		log.Fatal(err)
	}
	CheckNum(rep.Ret, 20, t)

	repSimple := new(int)
	err = client.Call("Test.IncCounterSimpleArgs", 10, repSimple)
	if err != nil {
		log.Fatal(err)
	}
	CheckNum(*repSimple, 30, t)

	client.Close()
	server.Stop()
}

func TestIncCounterSimpleArgs(t *testing.T) {
	log.SetLevel(log.FatalLevel)
	addr := "127.0.0.1:0"
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}

	server, err := NewServerWithService(ServiceMap{"Test": NewTestService()})
	server.SetListener(l)
	go server.Serve()

	client, err := initClient(l.Addr().String())
	if err != nil {
		log.Fatal(err)
	}

	repSimple := new(int)
	err = client.Call("Test.IncCounterSimpleArgs", 10, repSimple)
	if err != nil {
		log.Fatal(err)
	}
	CheckNum(*repSimple, 10, t)

	client.Close()
	server.Stop()
}

func TestEncryptIncCounterSimpleArgs(t *testing.T) {
	defer os.Remove(PubKeyStorePath)
	log.SetLevel(log.FatalLevel)
	addr := "127.0.0.1:0"
	masterKey := []byte("abc")
	server, err := NewServerWithService(ServiceMap{"Test": NewTestService()})
	if err != nil {
		log.Fatal(err)
	}

	route.NewDHTService(PubKeyStorePath, new(consistent.KMSStorage), true)
	server.InitRPCServer(addr, "../keys/test.key", masterKey)
	go server.Serve()

	publicKey, err := kms.GetLocalPublicKey()
	nonce := asymmetric.GetPubKeyNonce(publicKey, 10, 100*time.Millisecond, nil)
	serverNodeID := proto.NodeID(nonce.Hash.String())
	kms.SetPublicKey(serverNodeID, nonce.Nonce, publicKey)
	kms.SetLocalNodeIDNonce(nonce.Hash.CloneBytes(), &nonce.Nonce)
	route.SetNodeAddrCache(&proto.RawNodeID{Hash: nonce.Hash}, server.Listener.Addr().String())

	cryptoConn, err := DialToNode(serverNodeID, nil, false)
	client, err := InitClientConn(cryptoConn)
	if err != nil {
		log.Fatal(err)
	}

	repSimple := new(int)
	err = client.Call("Test.IncCounterSimpleArgs", 10, repSimple)
	if err != nil {
		log.Fatal(err)
	}
	CheckNum(*repSimple, 10, t)

	client.Close()
	server.Stop()
}

func TestETLSBug(t *testing.T) {
	defer os.Remove(PubKeyStorePath)
	log.SetLevel(log.FatalLevel)
	addr := "127.0.0.1:0"
	masterKey := []byte("abc")
	server, err := NewServerWithService(ServiceMap{"Test": NewTestService()})
	if err != nil {
		log.Fatal(err)
	}

	route.NewDHTService(PubKeyStorePath, new(consistent.KMSStorage), true)
	server.InitRPCServer(addr, "../keys/test.key", masterKey)
	go server.Serve()
	defer server.Stop()

	// This should not block listener
	var rawConn net.Conn
	rawConn, err = net.Dial("tcp", server.Listener.Addr().String())
	if err != nil {
		log.Fatal(err)
	}
	defer rawConn.Close()

	publicKey, err := kms.GetLocalPublicKey()
	nonce := asymmetric.GetPubKeyNonce(publicKey, 10, 100*time.Millisecond, nil)
	serverNodeID := proto.NodeID(nonce.Hash.String())
	kms.SetPublicKey(serverNodeID, nonce.Nonce, publicKey)
	kms.SetLocalNodeIDNonce(nonce.Hash.CloneBytes(), &nonce.Nonce)
	route.SetNodeAddrCache(&proto.RawNodeID{Hash: nonce.Hash}, server.Listener.Addr().String())

	cryptoConn, err := DialToNode(serverNodeID, nil, false)
	cryptoConn.SetDeadline(time.Now().Add(3 * time.Second))
	client, err := InitClientConn(cryptoConn)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	repSimple := new(int)
	err = client.Call("Test.IncCounterSimpleArgs", 10, repSimple)
	if err != nil {
		log.Fatal(err)
	}
	CheckNum(*repSimple, 10, t)
}

func TestEncPingFindNeighbor(t *testing.T) {
	os.Remove(PubKeyStorePath)
	defer os.Remove(PubKeyStorePath)
	log.SetLevel(log.FatalLevel)
	addr := "127.0.0.1:0"
	masterKey := []byte("abc")
	dht, err := route.NewDHTService(PubKeyStorePath, new(consistent.KMSStorage), true)

	server, err := NewServerWithService(ServiceMap{route.DHTRPCName: dht})
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
	route.SetNodeAddrCache(&proto.RawNodeID{Hash: nonce.Hash}, server.Listener.Addr().String())

	cryptoConn, err := DialToNode(serverNodeID, nil, false)
	client, err := InitClientConn(cryptoConn)
	if err != nil {
		log.Fatal(err)
	}

	node1 := proto.NewNode()
	node1.InitNodeCryptoInfo(100 * time.Millisecond)

	reqA := &proto.PingReq{
		Node: *node1,
	}

	respA := new(proto.PingResp)
	err = client.Call("DHT.Ping", reqA, respA)
	if err != nil {
		log.Fatal(err)
	}
	log.Debugf("respA: %v", respA)

	node2 := proto.NewNode()
	node2.InitNodeCryptoInfo(100 * time.Millisecond)

	reqB := &proto.PingReq{
		Node: *node2,
	}

	respB := new(proto.PingResp)
	err = client.Call("DHT.Ping", reqB, respB)
	if err != nil {
		log.Fatal(err)
	}
	log.Debugf("respB: %v", respB)

	req := &proto.FindNeighborReq{
		ID:    "1234567812345678123456781234567812345678123456781234567812345678",
		Count: 10,
	}
	resp := new(proto.FindNeighborResp)
	err = client.Call("DHT.FindNeighbor", req, resp)
	if err != nil {
		log.Fatal(err)
	}
	log.Debugf("resp: %v", resp)
	var nodeIDList []string
	for _, n := range resp.Nodes[:] {
		nodeIDList = append(nodeIDList, string(n.ID))
	}

	log.Debugf("nodeIDList: %v", nodeIDList)
	Convey("test FindNeighbor", t, func() {
		So(nodeIDList, ShouldContain, string(node1.ID))
		So(nodeIDList, ShouldContain, string(node2.ID))
		So(nodeIDList, ShouldContain, string(kms.BP.NodeID))
	})
	client.Close()
	server.Stop()
}

func TestServer_Close(t *testing.T) {
	log.SetLevel(log.FatalLevel)
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
