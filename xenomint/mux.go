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

	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	wt "github.com/CovenantSQL/CovenantSQL/worker/types"
)

// MuxService defines multiplexing service of xenomint chain.
type MuxService struct {
	ServiceName string
	// serviceMap maps DatabaseID to *Chain instance.
	serviceMap sync.Map
}

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

type MuxQueryRequest struct {
	proto.DatabaseID
	proto.Envelope
	Request *wt.Request
}

type MuxQueryResponse struct {
	proto.DatabaseID
	proto.Envelope
	Response *wt.Response
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

func (s *MuxService) Query(req *MuxQueryRequest, resp *MuxQueryResponse) (err error) {
	defer log.WithFields(log.Fields{
		"req":  req,
		"resp": resp,
		"err":  &err,
	}).Debug("Processed query request")
	var (
		c *Chain
		r *wt.Response
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
