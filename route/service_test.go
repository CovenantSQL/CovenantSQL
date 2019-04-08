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

package route

import (
	"fmt"
	"net"
	"net/rpc"
	"os"
	"strings"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/consistent"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	. "github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

const DHTStorePath = "./DHTStore.keystore"

func TestDHTService_FindNeighbor_FindNode(t *testing.T) {
	utils.RemoveAll(DHTStorePath + "*")
	defer utils.RemoveAll(DHTStorePath + "*")
	log.SetLevel(log.DebugLevel)
	addr := "127.0.0.1:0"
	dht, _ := NewDHTService(DHTStorePath+"1", new(consistent.KMSStorage), false)
	kms.ResetBucket()
	rpc.RegisterName(DHTRPCName, dht)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Println(err)
		return
	}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				continue
			}
			go rpc.ServeConn(c)
		}
	}()

	client, err := rpc.Dial("tcp", ln.Addr().String())
	if err != nil {
		log.Error(err)
	}

	req := &FindNeighborReq{
		ID:    "123",
		Count: 2,
	}
	resp := new(FindNeighborResp)
	err = client.Call("DHT.FindNeighbor", req, resp)
	if err != nil {
		log.Error(err)
	}
	log.Debugf("resp: %v", resp)

	Convey("test FindNeighbor empty", t, func() {
		So(resp.Nodes, ShouldBeEmpty)
		So(err.Error(), ShouldContainSubstring, consistent.ErrEmptyCircle.Error())
	})

	reqFN1 := &FindNodeReq{
		ID: "123",
	}
	respFN1 := new(FindNodeResp)
	err = client.Call("DHT.FindNode", reqFN1, respFN1)
	if err != nil {
		log.Error(err)
	}
	log.Debugf("respFN1: %v", respFN1)
	Convey("test FindNode", t, func() {
		So(respFN1.Node, ShouldBeNil)
		So(err.Error(), ShouldContainSubstring, consistent.ErrKeyNotFound.Error())
	})

	node1 := NewNode()
	node1.InitNodeCryptoInfo(100 * time.Millisecond)
	node1.Addr = "node1 addr"
	node1.Role = Miner

	reqA := &PingReq{
		Node: *node1,
	}
	respA := new(PingResp)
	err = client.Call("DHT.Ping", reqA, respA)
	if err != nil {
		log.Error(err)
	}
	log.Debugf("respA: %v", respA)

	reqFN2 := &FindNodeReq{
		ID: node1.ID,
	}
	respFN2 := new(FindNodeResp)
	err = client.Call("DHT.FindNode", reqFN2, respFN2)
	if err != nil {
		log.Error(err)
	}
	log.Debugf("respFN1: %v", respFN2)
	Convey("test FindNode", t, func() {
		So(respFN2.Node.ID, ShouldEqual, node1.ID)
		So(respFN2.Node.Nonce == node1.Nonce, ShouldBeTrue)
		So(respFN2.Node.Addr, ShouldEqual, node1.Addr)
		So(respFN2.Node.PublicKey.IsEqual(node1.PublicKey), ShouldBeTrue)
		So(err, ShouldBeNil)
	})

	node2 := NewNode()
	node2.InitNodeCryptoInfo(100 * time.Millisecond)

	reqB := &PingReq{
		Node: *node2,
	}
	respB := new(PingResp)
	err = client.Call("DHT.Ping", reqB, respB)
	if err != nil {
		log.Error(err)
	}
	log.Debugf("respA: %v", respB)

	req = &FindNeighborReq{
		ID:    "123",
		Count: 10,
	}
	resp = new(FindNeighborResp)
	err = client.Call("DHT.FindNeighbor", req, resp)
	if err != nil {
		log.Error(err)
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
	})

	req = &FindNeighborReq{
		ID:    "123",
		Count: 10,
		Roles: ServerRoles{Miner},
	}
	resp = new(FindNeighborResp)
	err = client.Call("DHT.FindNeighbor", req, resp)
	if err != nil {
		log.Error(err)
	}
	log.Debugf("resp filter: %v", resp)
	nodeIDList = make([]string, 0)
	for _, n := range resp.Nodes[:] {
		nodeIDList = append(nodeIDList, string(n.ID))
	}
	log.Debugf("nodeIDList filter: %v", nodeIDList)
	Convey("test FindNeighbor", t, func() {
		So(nodeIDList, ShouldContain, string(node1.ID))
		So(nodeIDList, ShouldNotContain, string(node2.ID))
	})

	reqFN3 := &FindNodeReq{
		ID: node2.ID,
	}
	respFN3 := new(FindNodeResp)
	err = client.Call("DHT.FindNode", reqFN3, respFN3)
	if err != nil {
		log.Error(err)
	}
	log.Debugf("respFN1: %v", respFN3)
	Convey("test FindNode", t, func() {
		So(respFN3.Node.ID, ShouldEqual, node2.ID)
		So(respFN3.Node.Nonce == node2.Nonce, ShouldBeTrue)
		So(respFN3.Node.Addr, ShouldEqual, node2.Addr)
		So(respFN3.Node.PublicKey.IsEqual(node2.PublicKey), ShouldBeTrue)
		So(err, ShouldBeNil)
	})

	client.Close()
}

func TestDHTService_Ping(t *testing.T) {
	utils.RemoveAll(DHTStorePath + "*")
	defer utils.RemoveAll(DHTStorePath + "*")
	log.SetLevel(log.DebugLevel)
	addr := "127.0.0.1:0"

	dht, _ := NewDHTService(DHTStorePath, new(consistent.KMSStorage), false)
	rpc.RegisterName(DHTRPCName, dht)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Println(err)
		return
	}

	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				continue
			}
			go rpc.ServeCodec(utils.GetMsgPackServerCodec(c))
		}
	}()

	client, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		log.Error(err)
	}

	NewNodeIDDifficultyTimeout = 100 * time.Millisecond
	node1 := NewNode()
	node1.InitNodeCryptoInfo(100 * time.Millisecond)

	reqA := &PingReq{
		Node: *node1,
	}
	respA := new(PingResp)
	rc := rpc.NewClientWithCodec(utils.GetMsgPackClientCodec(client))
	err = rc.Call("DHT.Ping", reqA, respA)
	if err != nil {
		t.Error(err)
	}
	log.Debugf("respA: %v", respA)
	rc.Close()

	respA3 := new(PingResp)
	conf.GConf.MinNodeIDDifficulty = 256
	client, _ = net.Dial("tcp", ln.Addr().String())
	rc = rpc.NewClientWithCodec(utils.GetMsgPackClientCodec(client))
	err = rc.Call("DHT.Ping", reqA, respA3)
	if err == nil || !strings.Contains(err.Error(), "difficulty too low") {
		t.Error(err)
	}
	log.Debugf("respA3: %v", respA3)
	rc.Close()

	respA2 := new(PingResp)
	reqA.Node.Nonce.A = ^uint64(0)
	client, _ = net.Dial("tcp", ln.Addr().String())
	rc = rpc.NewClientWithCodec(utils.GetMsgPackClientCodec(client))
	err = rc.Call("DHT.Ping", reqA, respA2)
	if err == nil || !strings.Contains(err.Error(), "nonce public key not match") {
		t.Error(err)
	}
	log.Debugf("respA2: %v", respA2)
	rc.Close()
}

func TestPermitCheckFunc(t *testing.T) {
	cases := [...]struct {
		envelop *proto.Envelope
		method  RemoteFunc
		expect  bool
	}{
		{
			envelop: &proto.Envelope{},
			method:  DHTPing,
			expect:  true,
		}, {
			envelop: &Envelope{NodeID: kms.AnonymousRawNodeID},
			method:  DHTPing,
			expect:  true,
		}, {
			envelop: &Envelope{NodeID: kms.AnonymousRawNodeID},
			method:  DHTFindNode,
			expect:  false,
		}, {
			envelop: &Envelope{NodeID: &proto.RawNodeID{}},
			method:  DHTFindNode,
			expect:  true,
		}, {
			envelop: &Envelope{NodeID: &proto.RawNodeID{}},
			method:  DHTFindNeighbor,
			expect:  true,
		}, {
			envelop: &Envelope{NodeID: &proto.RawNodeID{}},
			method:  MetricUploadMetrics,
			expect:  true,
		}, {
			envelop: &Envelope{NodeID: &proto.RawNodeID{}},
			method:  DHTGSetNode,
			expect:  false,
		}, {
			envelop: &Envelope{NodeID: &proto.RawNodeID{}},
			method:  DBSDeploy,
			expect:  false,
		}, {
			envelop: &Envelope{NodeID: &proto.RawNodeID{}},
			method:  0xffffffff,
			expect:  false,
		},
	}
	for i, v := range cases {
		if ret := IsPermitted(v.envelop, v.method); ret != v.expect {
			t.Errorf("case failed: i=%d, case=%v, expect=%v, return=%v",
				i, v, v.expect, ret)
		}
	}
}

func TestMain(m *testing.M) {
	os.Exit(func() int {
		permissionCheckFunc = nil
		return m.Run()
	}())
}
