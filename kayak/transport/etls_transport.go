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

package transport

import (
	"context"
	"fmt"
	nrpc "net/rpc"
	"sync"

	"gitlab.com/thunderdb/ThunderDB/kayak"
	"gitlab.com/thunderdb/ThunderDB/proto"
	"gitlab.com/thunderdb/ThunderDB/rpc"
)

// ETLSRPCClientBuilder defines a rpc.Client builder.
type ETLSRPCClientBuilder func(context.Context, proto.NodeID) (*rpc.Client, error)

// ETLSTransportConfig defines a transport config with transport id and rpc service related config.
type ETLSTransportConfig struct {
	NodeID           proto.NodeID
	TransportID      string
	TransportService *ETLSTransportService
	ServiceName      string
	ClientBuilder    ETLSRPCClientBuilder
}

// ETLSTransport defines kayak transport using ETLS rpc as transport layer.
type ETLSTransport struct {
	*ETLSTransportConfig
	queue chan kayak.Request
}

// ETLSTransportService defines kayak rpc endpoint to be registered to rpc server.
type ETLSTransportService struct {
	serviceMap sync.Map
}

// ETLSTransportRequest defines kayak rpc request entity.
type ETLSTransportRequest struct {
	proto.Envelope
	TransportID   string
	NodeID        proto.NodeID
	Method        string
	Log           *kayak.Log
	Response      []byte
	Error         error
	respAvailable chan struct{}
	respInit      sync.Once
}

// ETLSTransportResponse defines kayak rpc response entity.
type ETLSTransportResponse struct {
	proto.Envelope
	Data []byte
}

// NewETLSTransport creates new transport and bind to transport service with specified transport id.
func NewETLSTransport(config *ETLSTransportConfig) (t *ETLSTransport, err error) {
	t = &ETLSTransport{
		ETLSTransportConfig: config,
		queue:               make(chan kayak.Request, 100),
	}

	t.TransportService.register(t)

	return
}

// Request implements kayak.Transport.Request.
func (e *ETLSTransport) Request(ctx context.Context,
	nodeID proto.NodeID, method string, log *kayak.Log) (response []byte, err error) {
	// send through rpc
	var client *rpc.Client

	if client, err = e.ClientBuilder(ctx, nodeID); err != nil {
		return
	}

	req := &ETLSTransportRequest{
		TransportID: e.TransportID,
		NodeID:      e.NodeID,
		Method:      method,
		Log:         log,
	}
	resp := &ETLSTransportResponse{}

	ch := client.Go(fmt.Sprintf("%s.Call", e.ServiceName), req, resp, make(chan *nrpc.Call, 1))

	select {
	case <-ctx.Done():
		err = ctx.Err()
	case call := <-ch.Done:
		return resp.Data, call.Error
	}

	return
}

// Process implements kayak.Transport.Process.
func (e *ETLSTransport) Process() <-chan kayak.Request {
	// get response from remote request
	return e.queue
}

func (e *ETLSTransport) enqueue(req *ETLSTransportRequest) {
	e.queue <- req
}

// GetPeerNodeID implements kayak.Request.GetPeerNodeID.
func (r *ETLSTransportRequest) GetPeerNodeID() proto.NodeID {
	return r.NodeID
}

// GetMethod implements kayak.Request.GetMethod.
func (r *ETLSTransportRequest) GetMethod() string {
	return r.Method
}

// GetLog implements kayak.Request.GetLog.
func (r *ETLSTransportRequest) GetLog() *kayak.Log {
	return r.Log
}

// SendResponse implements kayak.Request.SendResponse.
func (r *ETLSTransportRequest) SendResponse(resp []byte, err error) error {
	// send response with transport id
	// TODO(xq262144), too tricky
	r.respInit.Do(r.initChan)
	select {
	case <-r.respAvailable:
		return kayak.ErrInvalidRequest
	default:
		r.Response = resp
		r.Error = err
		close(r.respAvailable)
	}
	return nil
}

func (r *ETLSTransportRequest) initChan() {
	r.respAvailable = make(chan struct{})
}

func (r *ETLSTransportRequest) getResponse() ([]byte, error) {
	// TODO(xq262144), too tricky
	r.respInit.Do(r.initChan)
	select {
	case <-r.respAvailable:
	}
	return r.Response, r.Error
}

// Call is the rpc entry of ETLS transport.
func (s *ETLSTransportService) Call(req *ETLSTransportRequest, resp *ETLSTransportResponse) error {
	// verify
	// TODO(xq262144), unified NodeID types in project
	if req.Envelope.NodeID.String() != string(req.NodeID) {
		return kayak.ErrInvalidRequest
	}

	var t interface{}
	var trans *ETLSTransport
	var ok bool

	if t, ok = s.serviceMap.Load(req.TransportID); !ok {
		return kayak.ErrInvalidRequest
	}

	if trans, ok = t.(*ETLSTransport); !ok {
		return kayak.ErrInvalidRequest
	}

	trans.enqueue(req)
	obj, err := req.getResponse()

	if resp != nil {
		resp.Data = obj
	}

	return err
}

func (s *ETLSTransportService) register(t *ETLSTransport) {
	s.serviceMap.Store(t.TransportID, t)
}
