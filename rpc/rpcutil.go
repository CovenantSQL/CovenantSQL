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
	"context"
	"errors"
	"math/rand"
	"net/rpc"
	"sync"

	"github.com/hashicorp/yamux"
	"gitlab.com/thunderdb/ThunderDB/crypto/kms"
	"gitlab.com/thunderdb/ThunderDB/proto"
	"gitlab.com/thunderdb/ThunderDB/route"
	"gitlab.com/thunderdb/ThunderDB/utils/log"
)

var (
	// ErrNoChiefBlockProducerAvailable defines failure on find chief block producer.
	ErrNoChiefBlockProducerAvailable = errors.New("no chief block producer found")

	// currentBP represents current chief block producer node.
	currentBP proto.NodeID
	// currentBPLock represents the chief block producer access lock.
	currentBPLock sync.Mutex
)

// Caller is a wrapper for session pooling and RPC calling.
type Caller struct {
	pool *SessionPool
}

// NewCaller returns a new RPCCaller.
func NewCaller() *Caller {
	return &Caller{
		pool: GetSessionPoolInstance(),
	}
}

// CallNode invokes the named function, waits for it to complete, and returns its error status.
func (c *Caller) CallNode(
	node proto.NodeID, method string, args interface{}, reply interface{}) (err error) {
	return c.CallNodeWithContext(context.Background(), node, method, args, reply)
}

// CallNodeWithContext invokes the named function, waits for it to complete or context timeout, and returns its error status.
func (c *Caller) CallNodeWithContext(
	ctx context.Context, node proto.NodeID, method string, args interface{}, reply interface{}) (err error) {
	conn, err := DialToNode(node, c.pool, method == route.DHTPing.String())
	if err != nil {
		log.Errorf("dialing to node: %s failed: %s", node, err)
		return
	}

	defer func() {
		// call the yamux stream Close explicitly
		//TODO(auxten) maybe a rpc client pool will gain much more performance
		stream, ok := conn.(*yamux.Stream)
		if ok {
			stream.Close()
		}
	}()

	client, err := InitClientConn(conn)
	if err != nil {
		log.Errorf("init RPC client failed: %s", err)
		return
	}

	defer client.Close()

	// TODO(xq262144): golang net/rpc does not support cancel in progress calls
	ch := client.Go(method, args, reply, make(chan *rpc.Call, 1))

	select {
	case <-ctx.Done():
		err = ctx.Err()
	case call := <-ch.Done:
		err = call.Error
	}

	return
}

// GetNodeAddr tries best to get node addr.
func GetNodeAddr(id *proto.RawNodeID) (addr string, err error) {
	addr, err = route.GetNodeAddrCache(id)
	if err != nil {
		log.Infof("get node %s addr failed: %s", id, err)
		if err == route.ErrUnknownNodeID {
			BPs := route.GetBPs()
			if len(BPs) == 0 {
				log.Errorf("no available BP")
				return
			}
			client := NewCaller()
			reqFN := &proto.FindNodeReq{
				NodeID: proto.NodeID(id.String()),
			}
			respFN := new(proto.FindNodeResp)

			bp := BPs[rand.Intn(len(BPs))]
			method := "DHT.FindNode"
			err = client.CallNode(bp, method, reqFN, respFN)
			if err != nil {
				log.Errorf("call %s %s failed: %s", bp, method, err)
				return
			}
			route.SetNodeAddrCache(id, respFN.Node.Addr)
			addr = respFN.Node.Addr
		}
	}
	return
}

// GetNodeInfo tries best to get node info.
func GetNodeInfo(id *proto.RawNodeID) (nodeInfo *proto.Node, err error) {
	nodeInfo, err = kms.GetNodeInfo(proto.NodeID(id.String()))
	if err != nil {
		log.Infof("get node info from KMS for %s failed: %s", id, err)
		if err == kms.ErrKeyNotFound {
			BPs := route.GetBPs()
			if len(BPs) == 0 {
				log.Errorf("no available BP")
				return
			}
			client := NewCaller()
			reqFN := &proto.FindNodeReq{
				NodeID: proto.NodeID(id.String()),
			}
			respFN := new(proto.FindNodeResp)
			bp := BPs[rand.Intn(len(BPs))]
			method := "DHT.FindNode"
			err = client.CallNode(bp, method, reqFN, respFN)
			if err != nil {
				log.Errorf("call %s %s failed: %s", bp, method, err)
				return
			}
			nodeInfo = respFN.Node
			errSet := route.SetNodeAddrCache(id, nodeInfo.Addr)
			if errSet != nil {
				log.Warnf("set node addr cache failed: %v", errSet)
			}
			errSet = kms.SetNode(nodeInfo)
			if errSet != nil {
				log.Warnf("set node to kms failed: %v", errSet)
			}
		}
	}
	return
}

// PingBP Send DHT.Ping Request with Anonymous ETLS session.
func PingBP(node *proto.Node, BPNodeID proto.NodeID) (err error) {
	client := NewCaller()

	req := &proto.PingReq{
		Node: *node,
	}

	resp := new(proto.PingResp)
	err = client.CallNode(BPNodeID, "DHT.Ping", req, resp)
	if err != nil {
		log.Errorf("call DHT.Ping failed: %v", err)
		return
	}
	log.Debugf("PingBP resp: %v", resp)

	return
}

// GetCurrentBP returns nearest hash distance block producer as current node chief block producer.
func GetCurrentBP() (bpNodeID proto.NodeID, err error) {
	currentBPLock.Lock()
	defer currentBPLock.Unlock()

	if !currentBP.IsEmpty() {
		bpNodeID = currentBP
		return
	}

	var localNodeID proto.NodeID
	if localNodeID, err = kms.GetLocalNodeID(); err != nil {
		return
	}

	// get random block producer first
	bpList := route.GetBPs()

	if len(bpList) == 0 {
		err = ErrNoChiefBlockProducerAvailable
		return
	}

	randomBP := bpList[rand.Intn(len(bpList))]

	// call random block producer for nearest block producer node
	req := &proto.FindNeighborReq{
		NodeID: localNodeID,
		Roles: []proto.ServerRole{
			proto.Leader,
			// only leader is capable of allocating database in current implementation
			//proto.Follower,
		},
		Count: 1,
	}
	res := new(proto.FindNeighborResp)
	if err = NewCaller().CallNode(randomBP, "DHT.FindNeighbor", req, res); err != nil {
		return
	}

	if len(res.Nodes) <= 0 {
		log.Error("get no hash nearest block producer nodes")
		// node not found
		err = ErrNoChiefBlockProducerAvailable
		return
	}

	if res.Nodes[0].Role != proto.Leader && res.Nodes[0].Role != proto.Follower {
		log.Error("no suitable nodes with proper block producer role")
		// not block producer
		err = ErrNoChiefBlockProducerAvailable
		return
	}

	currentBP = res.Nodes[0].ID
	bpNodeID = currentBP

	return
}

// SetCurrentBP sets current node chief block producer.
func SetCurrentBP(bpNodeID proto.NodeID) {
	currentBPLock.Lock()
	defer currentBPLock.Unlock()

	currentBP = bpNodeID

	return
}
