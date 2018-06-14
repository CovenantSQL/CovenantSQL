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

	log "github.com/sirupsen/logrus"
	"github.com/thunderdb/ThunderDB/consistent"
	"github.com/thunderdb/ThunderDB/proto"
)

// DHTService is server side RPC implementation
type DHTService struct {
	hashRing *consistent.Consistent
}

// NewDHTService will return a new DHTService
func NewDHTService(DHTStorePath string, initBP bool) (s *DHTService, err error) {
	c, err := consistent.InitConsistent(DHTStorePath, initBP)
	if err != nil {
		log.Errorf("init DHT service failed: %s", err)
		return
	}
	s = &DHTService{
		hashRing: c,
	}
	return
}

// FindValue RPC returns FindValueReq.Count closest node from DHT
func (DHT *DHTService) FindValue(req *proto.FindValueReq, resp *proto.FindValueResp) (err error) {
	nodes, err := DHT.hashRing.GetN(string(req.NodeID), req.Count)
	if err != nil {
		log.Errorf("get nodes from DHT failed: %s", err)
		resp.Msg = fmt.Sprint(err)
		return
	}
	resp.Nodes = nodes
	return
}

// Ping RPC add PingReq.Node to DHT
func (DHT *DHTService) Ping(req *proto.PingReq, resp *proto.PingResp) (err error) {
	log.Debugf("got req: %#v", req)
	DHT.hashRing.Add(req.Node)
	resp = new(proto.PingResp)
	resp.Msg = "Pong"
	return
}
