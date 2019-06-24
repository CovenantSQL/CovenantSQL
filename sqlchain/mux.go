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

package sqlchain

import (
	"sync"

	"github.com/CovenantSQL/CovenantSQL/proto"
	rpc "github.com/CovenantSQL/CovenantSQL/rpc/mux"
)

// MuxService defines multiplexing service of sql-chain.
type MuxService struct {
	ServiceName string
	serviceMap  sync.Map
}

// NewMuxService creates a new multiplexing service and registers it to rpc server.
func NewMuxService(serviceName string, server *rpc.Server) (service *MuxService, err error) {
	service = &MuxService{
		ServiceName: serviceName,
	}

	err = server.RegisterService(serviceName, service)
	return
}

func (s *MuxService) register(id proto.DatabaseID, service *ChainRPCService) {
	s.serviceMap.Store(id, service)
}

func (s *MuxService) unregister(id proto.DatabaseID) {
	s.serviceMap.Delete(id)
}

// MuxAdviseNewBlockReq defines a request of the AdviseNewBlock RPC method.
type MuxAdviseNewBlockReq struct {
	proto.Envelope
	proto.DatabaseID
	AdviseNewBlockReq
}

// MuxAdviseNewBlockResp defines a response of the AdviseNewBlock RPC method.
type MuxAdviseNewBlockResp struct {
	proto.Envelope
	proto.DatabaseID
	AdviseNewBlockResp
}

// MuxFetchBlockReq defines a request of the FetchBlock RPC method.
type MuxFetchBlockReq struct {
	proto.Envelope
	proto.DatabaseID
	FetchBlockReq
}

// MuxFetchBlockResp defines a response of the FetchBlock RPC method.
type MuxFetchBlockResp struct {
	proto.Envelope
	proto.DatabaseID
	FetchBlockResp
}

// AdviseNewBlock is the RPC method to advise a new produced block to the target server.
func (s *MuxService) AdviseNewBlock(req *MuxAdviseNewBlockReq, resp *MuxAdviseNewBlockResp) error {
	if v, ok := s.serviceMap.Load(req.DatabaseID); ok {
		resp.Envelope = req.Envelope
		resp.DatabaseID = req.DatabaseID
		return v.(*ChainRPCService).AdviseNewBlock(&req.AdviseNewBlockReq, &resp.AdviseNewBlockResp)
	}

	return ErrUnknownMuxRequest
}

// FetchBlock is the RPC method to fetch a known block from the target server.
func (s *MuxService) FetchBlock(req *MuxFetchBlockReq, resp *MuxFetchBlockResp) (err error) {
	if v, ok := s.serviceMap.Load(req.DatabaseID); ok {
		resp.Envelope = req.Envelope
		resp.DatabaseID = req.DatabaseID
		innerReq := &FetchBlockReq{
			node: req.NodeID.ToNodeID(),
		}
		return v.(*ChainRPCService).FetchBlock(innerReq, &resp.FetchBlockResp)
	}

	return ErrUnknownMuxRequest
}
