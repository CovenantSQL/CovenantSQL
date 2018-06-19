/*
 * Copyright 2018 The ThunderDB Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the “License”);
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an “AS IS” BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package transport

import (
	"context"
	"io"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"sync"

	"github.com/thunderdb/ThunderDB/kayak"
	"github.com/thunderdb/ThunderDB/proto"
)

// ConnWithPeerNodeID defines interface support getting remote peer ID
type ConnWithPeerNodeID interface {
	net.Conn

	GetPeerNodeID() proto.NodeID
}

// StreamLayer is the underlying network connection layer
type StreamLayer interface {
	Accept() (ConnWithPeerNodeID, error)
	Dial(context.Context, proto.NodeID) (ConnWithPeerNodeID, error)
}

// NetworkRequest is the request object hand off inter node request
type NetworkRequest struct {
	NodeID        proto.NodeID
	Method        string
	Log           *kayak.Log
	Response      interface{}
	Error         error
	respAvailable chan struct{}
	respInit      sync.Once
}

// ClientCodecBuilder is the client codec builder
type ClientCodecBuilder func(io.ReadWriteCloser) rpc.ClientCodec

// ServerCodecBuilder is the server codec builder
type ServerCodecBuilder func(closer io.ReadWriteCloser) rpc.ServerCodec

// NetworkResponse is the response object hand off inter node response
type NetworkResponse struct {
	Response interface{}
}

// NetworkTransport support customized stream layer integration with kayak transport
type NetworkTransport struct {
	config     *NetworkTransportConfig
	shutdownCh chan struct{}
	queue      chan kayak.Request
}

// NetworkTransportConfig defines NetworkTransport config object
type NetworkTransportConfig struct {
	NodeID      proto.NodeID
	StreamLayer StreamLayer

	ClientCodec ClientCodecBuilder
	ServerCodec ServerCodecBuilder
}

// NetworkTransportRequestProxy defines a rpc proxy method exported to golang net/rpc
type NetworkTransportRequestProxy struct {
	transport *NetworkTransport
	conn      ConnWithPeerNodeID
	server    *rpc.Server
}

// NewConfig returns new transport config
func NewConfig(nodeID proto.NodeID, streamLayer StreamLayer) (c *NetworkTransportConfig) {
	return NewConfigWithCodec(nodeID, streamLayer, jsonrpc.NewClientCodec, jsonrpc.NewServerCodec)
}

// NewConfigWithCodec returns new transport config with custom codec
func NewConfigWithCodec(nodeID proto.NodeID, streamLayer StreamLayer,
	clientCodec ClientCodecBuilder, serverCodec ServerCodecBuilder) (c *NetworkTransportConfig) {
	return &NetworkTransportConfig{
		NodeID:      nodeID,
		StreamLayer: streamLayer,
		ClientCodec: clientCodec,
		ServerCodec: serverCodec,
	}
}

// NewRequest returns new request entity
func NewRequest(nodeID proto.NodeID, method string, log *kayak.Log) (r *NetworkRequest) {
	return &NetworkRequest{
		NodeID: nodeID,
		Method: method,
		Log:    log,
	}
}

// NewResponse returns response returns new response entity
func NewResponse() (r *NetworkResponse) {
	return &NetworkResponse{}
}

// NewTransport returns new network transport
func NewTransport(config *NetworkTransportConfig) (t *NetworkTransport) {
	t = &NetworkTransport{
		config:     config,
		shutdownCh: make(chan struct{}),
		queue:      make(chan kayak.Request, 100),
	}

	go t.run()

	return
}

// NewRequestProxy returns request proxy object hand-off golang net/rpc
func NewRequestProxy(transport *NetworkTransport, conn ConnWithPeerNodeID) (rp *NetworkTransportRequestProxy) {
	rp = &NetworkTransportRequestProxy{
		transport: transport,
		conn:      conn,
		server:    rpc.NewServer(),
	}

	rp.server.RegisterName("Service", rp)

	return
}

// GetNodeID implements kayak.Request.GetNodeID
func (r *NetworkRequest) GetNodeID() proto.NodeID {
	return r.NodeID
}

// GetMethod implements kayak.Request.GetMethod
func (r *NetworkRequest) GetMethod() string {
	return r.Method
}

// GetLog implements kayak.Request.GetLog
func (r *NetworkRequest) GetLog() *kayak.Log {
	return r.Log
}

// SendResponse implements kayak.Request.SendResponse
func (r *NetworkRequest) SendResponse(resp interface{}, err error) error {
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

func (r *NetworkRequest) getResponse() (interface{}, error) {
	// TODO(xq262144), too tricky
	r.respInit.Do(r.initChan)
	select {
	case <-r.respAvailable:
	}
	return r.Response, r.Error
}

func (r *NetworkRequest) initChan() {
	r.respAvailable = make(chan struct{})
}

func (r *NetworkResponse) set(v interface{}) {
	r.Response = v
}

func (r *NetworkResponse) get() interface{} {
	return r.Response
}

// Request implements Transport.Request method
func (t *NetworkTransport) Request(ctx context.Context, nodeID proto.NodeID,
	method string, log *kayak.Log) (response interface{}, err error) {
	conn, err := t.config.StreamLayer.Dial(ctx, nodeID)

	if err != nil {
		return
	}

	// check node id
	if conn.GetPeerNodeID() != nodeID {
		// err creating connection
		return nil, kayak.ErrInvalidRequest
	}

	client := rpc.NewClientWithCodec(t.config.ClientCodec(conn))
	req := NewRequest(t.config.NodeID, method, log)
	res := NewResponse()
	// TODO(xq262144), too tricky
	err = client.Call("Service.Call", req, res)

	return res.get(), err
}

// Process implements Transport.Process method
func (t *NetworkTransport) Process() <-chan kayak.Request {
	return t.queue
}

func (t *NetworkTransport) enqueue(req *NetworkRequest) {
	t.queue <- req
}

// Call hand-off request from remote rpc server
func (p *NetworkTransportRequestProxy) Call(req *NetworkRequest, res *NetworkResponse) error {
	// TODO(xq262144), too tricky
	// verify node id
	if p.conn.GetPeerNodeID() != req.NodeID {
		return kayak.ErrInvalidRequest
	}

	p.transport.enqueue(req)
	obj, err := req.getResponse()
	res.set(obj)
	return err
}

func (p *NetworkTransportRequestProxy) serve() {
	p.server.ServeCodec(p.transport.config.ServerCodec(p.conn))
}

// Close shutdown transport hand-off
func (t *NetworkTransport) Close() {
	select {
	case <-t.shutdownCh:
	default:
		close(t.shutdownCh)
	}
}

func (t *NetworkTransport) run() {
	for {
		select {
		case <-t.shutdownCh:
			return
		default:
			conn, err := t.config.StreamLayer.Accept()
			if err != nil {
				// TODO, log
				continue
			}

			go t.handleConn(conn)
		}
	}
}

func (t *NetworkTransport) handleConn(conn ConnWithPeerNodeID) {
	NewRequestProxy(t, conn).serve()
}
