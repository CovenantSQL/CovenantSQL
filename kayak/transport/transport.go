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

// Request is the request object hand off inter node request
type Request struct {
	NodeID        proto.NodeID
	Method        string
	Payload       interface{}
	Response      interface{}
	Error         error
	respAvailable chan struct{}
	respInit      sync.Once
}

// ClientCodecBuilder is the client codec builder
type ClientCodecBuilder func(io.ReadWriteCloser) rpc.ClientCodec

// ServerCodecBuilder is the server codec builder
type ServerCodecBuilder func(closer io.ReadWriteCloser) rpc.ServerCodec

// Response is the response object hand off inter node response
type Response struct {
	Response interface{}
}

// Transport support customized stream layer integration with kayak transport
type Transport struct {
	config     *Config
	shutdownCh chan struct{}
	queue      chan kayak.Request
}

// Config defines Transport config object
type Config struct {
	NodeID      proto.NodeID
	StreamLayer StreamLayer

	ClientCodec ClientCodecBuilder
	ServerCodec ServerCodecBuilder
}

// RequestProxy defines a rpc proxy method exported to golang net/rpc
type RequestProxy struct {
	transport *Transport
	conn      ConnWithPeerNodeID
	server    *rpc.Server
}

// NewConfig returns new transport config
func NewConfig(nodeID proto.NodeID, streamLayer StreamLayer) (c *Config) {
	return NewConfigWithCodec(nodeID, streamLayer, jsonrpc.NewClientCodec, jsonrpc.NewServerCodec)
}

// NewConfigWithCodec returns new transport config with custom codec
func NewConfigWithCodec(nodeID proto.NodeID, streamLayer StreamLayer,
	clientCodec ClientCodecBuilder, serverCodec ServerCodecBuilder) (c *Config) {
	return &Config{
		NodeID:      nodeID,
		StreamLayer: streamLayer,
		ClientCodec: clientCodec,
		ServerCodec: serverCodec,
	}
}

// NewRequest returns new request entity
func NewRequest(nodeID proto.NodeID, method string, payload interface{}) (r *Request) {
	return &Request{
		NodeID:  nodeID,
		Method:  method,
		Payload: payload,
	}
}

// NewResponse returns response returns new response entity
func NewResponse() (r *Response) {
	return &Response{}
}

// NewTransport returns new network transport
func NewTransport(config *Config) (t *Transport) {
	t = &Transport{
		config:     config,
		shutdownCh: make(chan struct{}),
		queue:      make(chan kayak.Request, 100),
	}

	go t.run()

	return
}

// NewRequestProxy returns request proxy object hand-off golang net/rpc
func NewRequestProxy(transport *Transport, conn ConnWithPeerNodeID) (rp *RequestProxy) {
	rp = &RequestProxy{
		transport: transport,
		conn:      conn,
		server:    rpc.NewServer(),
	}

	rp.server.RegisterName("Service", rp)

	return
}

// GetNodeID implements kayak.Request.GetNodeID
func (r *Request) GetNodeID() proto.NodeID {
	return r.NodeID
}

// GetMethod implements kayak.Request.GetMethod
func (r *Request) GetMethod() string {
	return r.Method
}

// GetRequest implements kayak.Request.GetRequest
func (r *Request) GetRequest() interface{} {
	return r.Payload
}

// SendResponse implements kayak.Request.SendResponse
func (r *Request) SendResponse(resp interface{}, err error) error {
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

func (r *Request) getResponse() (interface{}, error) {
	// TODO(xq262144), too tricky
	r.respInit.Do(r.initChan)
	select {
	case <-r.respAvailable:
	}
	return r.Response, r.Error
}

func (r *Request) initChan() {
	r.respAvailable = make(chan struct{})
}

func (r *Response) set(v interface{}) {
	r.Response = v
}

func (r *Response) get() interface{} {
	return r.Response
}

// Request implements Transport.Request method
func (t *Transport) Request(ctx context.Context, nodeID proto.NodeID,
	method string, args interface{}) (response interface{}, err error) {
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
	req := NewRequest(t.config.NodeID, method, args)
	res := NewResponse()
	// TODO(xq262144), too tricky
	err = client.Call("Service.Call", req, res)

	return res.get(), err
}

// Process implements Transport.Process method
func (t *Transport) Process() <-chan kayak.Request {
	return t.queue
}

func (t *Transport) enqueue(req *Request) {
	t.queue <- req
}

// Call hand-off request from remote rpc server
func (p *RequestProxy) Call(req *Request, res *Response) error {
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

func (p *RequestProxy) serve() {
	p.server.ServeCodec(p.transport.config.ServerCodec(p.conn))
}

// Close shutdown transport hand-off
func (t *Transport) Close() {
	select {
	case <-t.shutdownCh:
	default:
		close(t.shutdownCh)
	}
}

func (t *Transport) run() {
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

func (t *Transport) handleConn(conn ConnWithPeerNodeID) {
	NewRequestProxy(t, conn).serve()
}
