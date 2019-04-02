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

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/consistent"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

// DHTService is server side RPC implementation.
type DHTService struct {
	Consistent *consistent.Consistent
}

// NewDHTServiceWithRing will return a new DHTService and set an existing hash ring.
func NewDHTServiceWithRing(c *consistent.Consistent) (s *DHTService, err error) {
	s = &DHTService{
		Consistent: c,
	}
	return
}

// NewDHTService will return a new DHTService.
func NewDHTService(DHTStorePath string, persistImpl consistent.Persistence, initBP bool) (s *DHTService, err error) {
	c, err := consistent.InitConsistent(DHTStorePath, persistImpl, initBP)
	if err != nil {
		log.WithError(err).Error("init DHT service failed")
		return
	}
	return NewDHTServiceWithRing(c)
}

// Nil RPC does nothing just for probe.
func (DHT *DHTService) Nil(req *interface{}, resp *interface{}) (err error) {
	return
}

var permissionCheckFunc = IsPermitted

// FindNode RPC returns node with requested node id from DHT.
func (DHT *DHTService) FindNode(req *proto.FindNodeReq, resp *proto.FindNodeResp) (err error) {
	if permissionCheckFunc != nil && !permissionCheckFunc(&req.Envelope, DHTFindNode) {
		err = fmt.Errorf("calling from node %s is not permitted", req.GetNodeID())
		log.Error(err)
		return
	}
	node, err := DHT.Consistent.GetNode(string(req.ID))
	if err != nil {
		err = fmt.Errorf("get node %s from DHT failed: %s", req.NodeID, err)
		log.Error(err)
		return
	}
	resp.Node = node
	return
}

// FindNeighbor RPC returns FindNeighborReq.Count closest node from DHT.
func (DHT *DHTService) FindNeighbor(req *proto.FindNeighborReq, resp *proto.FindNeighborResp) (err error) {
	if permissionCheckFunc != nil && !permissionCheckFunc(&req.Envelope, DHTFindNeighbor) {
		err = fmt.Errorf("calling from node %s is not permitted", req.GetNodeID())
		log.Error(err)
		return
	}

	nodes, err := DHT.Consistent.GetNeighborsEx(string(req.ID), req.Count, req.Roles)
	if err != nil {
		err = fmt.Errorf("get nodes from DHT failed: %s", err)
		log.Error(err)
		return
	}
	resp.Nodes = nodes
	log.WithFields(log.Fields{
		"neighCount": len(nodes),
		"req":        req,
	}).Debug("found nodes for find neighbor request")
	return
}

// Ping RPC adds PingReq.Node to DHT.
func (DHT *DHTService) Ping(req *proto.PingReq, resp *proto.PingResp) (err error) {
	log.Debugf("got req: %#v", req)
	if permissionCheckFunc != nil && !permissionCheckFunc(&req.Envelope, DHTPing) {
		err = fmt.Errorf("calling Ping from node %s is not permitted", req.GetNodeID())
		log.Error(err)
		return
	}

	// BP node is not permitted to set by RPC
	if req.Node.Role == proto.Leader || req.Node.Role == proto.Follower {
		err = fmt.Errorf("setting %s node is not permitted", req.Node.Role.String())
		log.Error(err)
		return
	}

	// Checking if ID Nonce Pubkey matched
	if !kms.IsIDPubNonceValid(req.Node.ID.ToRawNodeID(), &req.Node.Nonce, req.Node.PublicKey) {
		err = fmt.Errorf("node: %s nonce public key not match", req.Node.ID)
		log.Error(err)
		return
	}

	// Checking MinNodeIDDifficulty
	if req.Node.ID.Difficulty() < conf.GConf.MinNodeIDDifficulty {
		err = fmt.Errorf("node: %s difficulty too low", req.Node.ID)
		log.Error(err)
		return
	}

	err = DHT.Consistent.Add(req.Node)
	if err != nil {
		err = fmt.Errorf("DHT.Consistent.Add %v failed: %s", req.Node, err)
	} else {
		resp.Msg = "Pong"
	}
	return
}
