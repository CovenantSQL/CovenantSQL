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
	"sync"
	"time"

	"github.com/CovenantSQL/CovenantSQL/proto"
	rpc "github.com/CovenantSQL/CovenantSQL/rpc/mux"
	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
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
	var (
		c *Chain
		r *types.Response

		start = time.Now()

		routed, queried, responded time.Duration
	)
	defer func() {
		var fields = log.Fields{}
		if routed > 0 {
			fields["1#routed"] = float64(routed.Nanoseconds()) / 1000
		}
		if queried > 0 {
			fields["2#queried"] = float64((queried - routed).Nanoseconds()) / 1000
		}
		if responded > 0 {
			fields["3#responded"] = float64((responded - queried).Nanoseconds()) / 1000
		}
		log.WithFields(fields).Debug("MuxService.Query duration stat (us)")
	}()
	if c, err = s.route(req.DatabaseID); err != nil {
		return
	}
	routed = time.Since(start)
	if r, err = c.Query(req.Request); err != nil {
		return
	}
	queried = time.Since(start)
	resp = &MuxQueryResponse{
		Envelope:   req.Envelope,
		DatabaseID: req.DatabaseID,
		Response:   r,
	}
	responded = time.Since(start)
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
