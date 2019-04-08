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

package mux

import (
	"net"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	mux "github.com/xtaci/smux"

	"github.com/CovenantSQL/CovenantSQL/consistent"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

const PubKeyStorePath = "./public.keystore"

type nilSessionPool struct{}

func (p *nilSessionPool) Get(id proto.NodeID) (rpc.Client, error) {
	return p.GetEx(id, false)
}

func (p *nilSessionPool) GetEx(id proto.NodeID, isAnonymous bool) (conn rpc.Client, err error) {
	var (
		sess   *mux.Session
		stream *mux.Stream
	)
	if sess, err = newSession(id, isAnonymous); err != nil {
		return
	}
	if stream, err = sess.OpenStream(); err != nil {
		return
	}
	return rpc.NewClient(&oneOffMuxConn{
		sess:   sess,
		Stream: stream,
	}), nil
}

func (p *nilSessionPool) Close() error { return nil }

// CheckNum make int assertion.
func CheckNum(num, expected int32, t *testing.T) {
	if num != expected {
		t.Errorf("got %d, expected %d", num, expected)
	}
}

type TestService struct {
	counter int32
}

type TestReq struct {
	Step int32
}

type TestRep struct {
	Ret int32
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
	rep.Ret = atomic.AddInt32(&s.counter, req.Step)
	return nil
}

func (s *TestService) IncCounterSimpleArgs(step int32, ret *int32) error {
	log.WithFields(log.Fields{
		"req":   step,
		"reply": ret,
	}).Debug("calling IncCounter")
	*ret = atomic.AddInt32(&s.counter, step)

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
	go server.WithAcceptConnFunc(rpc.AcceptRawConn).Serve()

	conn, err := net.Dial("tcp", l.Addr().String())
	if err != nil {
		log.Fatal(err)
	}
	stream, err := NewOneOffMuxConn(conn)
	if err != nil {
		log.Fatal(err)
	}
	client := rpc.NewClient(stream)

	rep := new(TestRep)
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

	repSimple := new(int32)
	err = client.Call("Test.IncCounterSimpleArgs", 10, repSimple)
	if err != nil {
		log.Fatal(err)
	}
	CheckNum(*repSimple, 30, t)

	_ = client.Close()
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
	go server.WithAcceptConnFunc(rpc.AcceptRawConn).Serve()

	conn, err := net.Dial("tcp", l.Addr().String())
	if err != nil {
		log.Fatal(err)
	}
	stream, err := NewOneOffMuxConn(conn)
	if err != nil {
		log.Fatal(err)
	}
	client := rpc.NewClient(stream)

	repSimple := new(int32)
	err = client.Call("Test.IncCounterSimpleArgs", 10, repSimple)
	if err != nil {
		log.Fatal(err)
	}
	CheckNum(*repSimple, 10, t)

	_ = client.Close()
	server.Stop()
}

func TestEncryptIncCounterSimpleArgs(t *testing.T) {
	defer utils.RemoveAll(PubKeyStorePath + "*")
	log.SetLevel(log.FatalLevel)
	addr := "127.0.0.1:0"
	masterKey := []byte("abc")
	server, err := NewServerWithService(ServiceMap{"Test": NewTestService()})
	if err != nil {
		log.Fatal(err)
	}

	_, _ = route.NewDHTService(PubKeyStorePath, new(consistent.KMSStorage), true)
	_ = server.InitRPCServer(addr, "../keys/test.key", masterKey)
	go server.Serve()

	publicKey, err := kms.GetLocalPublicKey()
	nonce := asymmetric.GetPubKeyNonce(publicKey, 10, 100*time.Millisecond, nil)
	serverNodeID := proto.NodeID(nonce.Hash.String())
	_ = kms.SetPublicKey(serverNodeID, nonce.Nonce, publicKey)
	kms.SetLocalNodeIDNonce(nonce.Hash.CloneBytes(), &nonce.Nonce)
	_ = route.SetNodeAddrCache(&proto.RawNodeID{Hash: nonce.Hash}, server.Listener.Addr().String())

	client, err := rpc.DialToNodeWithPool(&nilSessionPool{}, serverNodeID, false)

	repSimple := new(int32)
	err = client.Call("Test.IncCounterSimpleArgs", 10, repSimple)
	if err != nil {
		log.Fatal(err)
	}
	CheckNum(*repSimple, 10, t)

	_ = client.Close()
	server.Stop()
}

func TestETLSBug(t *testing.T) {
	defer utils.RemoveAll(PubKeyStorePath + "*")
	log.SetLevel(log.FatalLevel)
	addr := "127.0.0.1:0"
	masterKey := []byte("abc")
	server, err := NewServerWithService(ServiceMap{"Test": NewTestService()})
	if err != nil {
		log.Fatal(err)
	}

	_, _ = route.NewDHTService(PubKeyStorePath, new(consistent.KMSStorage), true)
	_ = server.InitRPCServer(addr, "../keys/test.key", masterKey)
	go server.Serve()
	defer server.Stop()

	// This should not block listener
	var rawConn net.Conn
	rawConn, err = net.Dial("tcp", server.Listener.Addr().String())
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = rawConn.Close() }()

	publicKey, err := kms.GetLocalPublicKey()
	nonce := asymmetric.GetPubKeyNonce(publicKey, 10, 100*time.Millisecond, nil)
	serverNodeID := proto.NodeID(nonce.Hash.String())
	_ = kms.SetPublicKey(serverNodeID, nonce.Nonce, publicKey)
	kms.SetLocalNodeIDNonce(nonce.Hash.CloneBytes(), &nonce.Nonce)
	_ = route.SetNodeAddrCache(&proto.RawNodeID{Hash: nonce.Hash}, server.Listener.Addr().String())

	cryptoConn, err := rpc.Dial(serverNodeID)
	if err != nil {
		log.Fatal(err)
	}
	stream, err := NewOneOffMuxConn(cryptoConn)
	if err != nil {
		log.Fatal(err)
	}
	_ = stream.SetDeadline(time.Now().Add(3 * time.Second))
	client := rpc.NewClient(stream)
	defer func() { _ = client.Close() }()

	repSimple := new(int32)
	err = client.Call("Test.IncCounterSimpleArgs", 10, repSimple)
	if err != nil {
		log.Fatal(err)
	}
	CheckNum(*repSimple, 10, t)
}

func TestEncPingFindNeighbor(t *testing.T) {
	utils.RemoveAll(PubKeyStorePath + "*")
	defer utils.RemoveAll(PubKeyStorePath + "*")
	log.SetLevel(log.FatalLevel)
	addr := "127.0.0.1:0"
	masterKey := []byte("abc")
	dht, err := route.NewDHTService(PubKeyStorePath, new(consistent.KMSStorage), true)

	server, err := NewServerWithService(ServiceMap{route.DHTRPCName: dht})
	if err != nil {
		log.Fatal(err)
	}

	_ = server.InitRPCServer(addr, "../keys/test.key", masterKey)
	go server.Serve()

	publicKey, err := kms.GetLocalPublicKey()
	nonce := asymmetric.GetPubKeyNonce(publicKey, 10, 100*time.Millisecond, nil)
	serverNodeID := proto.NodeID(nonce.Hash.String())
	_ = kms.SetPublicKey(serverNodeID, nonce.Nonce, publicKey)

	kms.SetLocalNodeIDNonce(nonce.Hash.CloneBytes(), &nonce.Nonce)
	_ = route.SetNodeAddrCache(&proto.RawNodeID{Hash: nonce.Hash}, server.Listener.Addr().String())

	client, err := rpc.DialToNodeWithPool(&nilSessionPool{}, serverNodeID, false)
	if err != nil {
		log.Fatal(err)
	}

	node1 := proto.NewNode()
	_ = node1.InitNodeCryptoInfo(100 * time.Millisecond)

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
	_ = node2.InitNodeCryptoInfo(100 * time.Millisecond)

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
	_ = client.Close()
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
