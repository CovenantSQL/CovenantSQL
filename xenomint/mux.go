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

package xenomint

import (
	//"context"
	//"runtime/trace"
	"sync"

	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	"github.com/CovenantSQL/CovenantSQL/types"
)

// MuxService defines multiplexing service of xenomint chain.
type MuxService struct {
	ServiceName string
	// serviceMap maps DatabaseID to *Chain.
	serviceMap sync.Map
}

// NewMuxService returns a new MuxService instance and registers it to server.
func NewMuxService(name string, server *rpc.Server) (service *MuxService, err error) {
	var s = &MuxService{
		ServiceName: name,
	}
	if err = server.RegisterService(name, s); err != nil {
		return
	}
	service = s
	return
}

func (s *MuxService) register(id proto.DatabaseID, c *Chain) {
	s.serviceMap.Store(id, c)
}

func (s *MuxService) unregister(id proto.DatabaseID) {
	s.serviceMap.Delete(id)
}

func (s *MuxService) route(id proto.DatabaseID) (c *Chain, err error) {
	var (
		i  interface{}
		ok bool
	)
	if i, ok = s.serviceMap.Load(id); !ok {
		err = ErrMuxServiceNotFound
		return
	}
	if c, ok = i.(*Chain); !ok {
		err = ErrMuxServiceNotFound
		return
	}
	return
}

// MuxQueryRequest defines a request of the Query RPC method.
type MuxQueryRequest struct {
	proto.DatabaseID
	proto.Envelope
	Request *types.Request
}

// MuxQueryResponse defines a response of the Query RPC method.
type MuxQueryResponse struct {
	proto.DatabaseID
	proto.Envelope
	Response *types.Response
}

// Query is the RPC method to process database query on mux service.
func (s *MuxService) Query(req *MuxQueryRequest, resp *MuxQueryResponse) (err error) {
	//var ctx, task = trace.NewTask(context.Background(), "MuxService.Query")
	//defer task.End()
	//defer trace.StartRegion(ctx, "Total").End()
	var (
		c *Chain
		r *types.Response
	)
	if c, err = s.route(req.DatabaseID); err != nil {
		return
	}
	if r, err = c.Query(req.Request); err != nil {
		return
	}
	resp = &MuxQueryResponse{
		Envelope:   req.Envelope,
		DatabaseID: req.DatabaseID,
		Response:   r,
	}
	return
}

// MuxApplyRequest defines a request of the Apply RPC method.
type MuxApplyRequest struct {
	proto.DatabaseID
	proto.Envelope
	Request  *types.Request
	Response *types.Response
}

// MuxApplyResponse defines a response of the Apply RPC method.
type MuxApplyResponse struct {
	proto.DatabaseID
	proto.Envelope
}

// Apply is the RPC method to apply a write request on mux service.
func (s *MuxService) Apply(req *MuxApplyRequest, resp *MuxApplyResponse) (err error) {
	var c *Chain
	if c, err = s.route(req.DatabaseID); err != nil {
		return
	}
	c.enqueueIn(req)
	resp = &MuxApplyResponse{
		Envelope:   req.Envelope,
		DatabaseID: req.DatabaseID,
	}
	return
}

// MuxLeaderCommitRequest a request of the MuxLeaderCommitResponse RPC method.
type MuxLeaderCommitRequest struct {
	proto.DatabaseID
	proto.Envelope
	// Height is the expected block height of this commit.
	Height int32
}

// MuxLeaderCommitResponse a response of the MuxLeaderCommitResponse RPC method.
type MuxLeaderCommitResponse struct {
	proto.DatabaseID
	proto.Envelope
	// Height is the expected block height of this commit.
	Height int32
	Offset uint64
}

// LeaderCommit is the RPC method to commit block on mux service.
func (s *MuxService) LeaderCommit(
	req *MuxLeaderCommitRequest, resp *MuxLeaderCommitResponse) (err error,
) {
	var c *Chain
	if c, err = s.route(req.DatabaseID); err != nil {
		return
	}
	if err = c.commitBlock(); err != nil {
		return
	}
	resp = &MuxLeaderCommitResponse{
		Envelope:   req.Envelope,
		DatabaseID: req.DatabaseID,
	}
	return
}

// MuxFollowerCommitRequest a request of the FollowerCommit RPC method.
type MuxFollowerCommitRequest struct {
	proto.DatabaseID
	proto.Envelope
	Height int32
	Offset uint64
}

// MuxFollowerCommitResponse a response of the FollowerCommit RPC method.
type MuxFollowerCommitResponse struct {
	proto.DatabaseID
	proto.Envelope
	Height int32
	Offset uint64
}

// FollowerCommit is the RPC method to commit block on mux service.
func (s *MuxService) FollowerCommit(
	req *MuxFollowerCommitRequest, resp *MuxFollowerCommitResponse) (err error,
) {
	var c *Chain
	if c, err = s.route(req.DatabaseID); err != nil {
		return
	}
	c.enqueueIn(req)
	resp = &MuxFollowerCommitResponse{
		Envelope:   req.Envelope,
		DatabaseID: req.DatabaseID,
	}
	return
}
