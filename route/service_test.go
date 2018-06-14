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

package route

import (
	"fmt"
	"net"
	"testing"

	"net/rpc"

	"os"

	"time"

	log "github.com/sirupsen/logrus"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/thunderdb/ThunderDB/consistent"
	"github.com/thunderdb/ThunderDB/crypto/kms"
	. "github.com/thunderdb/ThunderDB/proto"
)

const DHTStorePath = "./DHTStore"

func TestPingFindValue(t *testing.T) {
	os.Remove(DHTStorePath)
	defer os.Remove(DHTStorePath)
	log.SetLevel(log.DebugLevel)
	addr := "127.0.0.1:0"
	dht, _ := NewDHTService(DHTStorePath, false)
	kms.ResetBucket()
	rpc.RegisterName("DHT", dht)
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

	req := &FindValueReq{
		NodeID: "123",
		Count:  2,
	}
	resp := new(FindValueResp)
	err = client.Call("DHT.FindValue", req, resp)
	if err != nil {
		log.Error(err)
	}
	log.Debugf("resp: %v", resp)

	Convey("test FindValue empty", t, func() {
		So(resp.Nodes, ShouldBeEmpty)
		So(err.Error(), ShouldEqual, consistent.ErrEmptyCircle.Error())
	})

	NewNodeIDDifficultyTimeout = 100 * time.Millisecond
	node1 := NewNode()
	node1.InitNodeCryptoInfo()
	nodeBytes1, _ := node1.Marshal()

	reqA := &PingReq{
		Node: nodeBytes1,
	}
	respA := new(PingResp)
	err = client.Call("DHT.Ping", reqA, respA)
	if err != nil {
		log.Error(err)
	}
	log.Debugf("respA: %v", respA)

	node2 := NewNode()
	node2.InitNodeCryptoInfo()
	nodeBytes2, _ := node2.Marshal()

	reqB := &PingReq{
		Node: nodeBytes2,
	}
	respB := new(PingResp)
	err = client.Call("DHT.Ping", reqB, respB)
	if err != nil {
		log.Error(err)
	}
	log.Debugf("respA: %v", respB)

	req = &FindValueReq{
		NodeID: "123",
		Count:  10,
	}
	resp = new(FindValueResp)
	err = client.Call("DHT.FindValue", req, resp)
	if err != nil {
		log.Error(err)
	}
	log.Debugf("resp: %v", resp)
	var nodeIDList []string
	for _, n := range resp.Nodes[:] {
		nodeResp, _ := UnmarshalNode(n)
		nodeIDList = append(nodeIDList, string(nodeResp.ID))
	}
	log.Debugf("nodeIDList: %v", nodeIDList)
	Convey("test FindValue", t, func() {
		So(nodeIDList, ShouldContain, string(node1.ID))
		So(nodeIDList, ShouldContain, string(node2.ID))
	})

	client.Close()
}
