/*
 * Copyright 2019 The CovenantSQL Authors.
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

import "github.com/CovenantSQL/CovenantSQL/proto"

// GossipRequest defines the gossip request payload.
type GossipRequest struct {
	proto.Envelope
	Node *proto.Node
	TTL  uint32
}

// GossipService defines the gossip service instance.
type GossipService struct {
	s *KVServer
}

// NewGossipService returns new gossip service.
func NewGossipService(s *KVServer) *GossipService {
	return &GossipService{
		s: s,
	}
}

// SetNode update current node info and broadcast node update request.
func (s *GossipService) SetNode(req *GossipRequest, resp *interface{}) (err error) {
	return s.s.SetNodeEx(req.Node, req.TTL, req.GetNodeID().ToNodeID())
}
