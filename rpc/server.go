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

package rpc

import (
	"context"
	"net"
	"net/rpc"

	"github.com/pkg/errors"

	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

// ServiceMap maps service name to service instance.
type ServiceMap map[string]interface{}

// ServeStream defines the stream serving function.
type ServeStream func(ctx context.Context, server *rpc.Server, conn net.Conn, remote proto.RawNodeID)

// Server is the RPC server struct.
type Server struct {
	ctx         context.Context
	cancel      context.CancelFunc
	rpcServer   *rpc.Server
	serviceMap  ServiceMap
	serveStream ServeStream
	Listener    net.Listener
}

// NewServerWithServeFunc return a new Server.
func NewServerWithServeFunc(f ServeStream) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		ctx:         ctx,
		cancel:      cancel,
		rpcServer:   rpc.NewServer(),
		serviceMap:  make(ServiceMap),
		serveStream: f,
	}
}

// InitRPCServer load the private key, init the crypto transfer layer and register RPC
// services.
// IF ANY ERROR returned, please raise a FATAL.
func (s *Server) InitRPCServer(
	addr string,
	privateKeyPath string,
	masterKey []byte,
) (err error) {
	//route.InitResolver()

	err = kms.InitLocalKeyPair(privateKeyPath, masterKey)
	if err != nil {
		err = errors.Wrap(err, "init local key pair failed")
		return
	}

	l, err := net.Listen("tcp", addr)
	if err != nil {
		err = errors.Wrap(err, "create crypto listener failed")
		return
	}

	s.SetListener(l)

	return
}

// SetListener set the service loop listener, used by func Serve main loop.
func (s *Server) SetListener(l net.Listener) {
	s.Listener = l
}

// Serve start the Server main loop,.
func (s *Server) Serve() {
serverLoop:
	for {
		select {
		case <-s.ctx.Done():
			log.Info("stopping Server Loop")
			break serverLoop
		default:
			conn, err := s.Listener.Accept()
			if err != nil {
				continue
			}
			log.WithField("remote", conn.RemoteAddr().String()).Info("accept")
			go s.serveConn(conn)
		}
	}
}

func (s *Server) serveConn(conn net.Conn) {
	noconn, err := Accept(conn)
	if err != nil {
		return
	}
	defer func() { _ = noconn.Close() }()

	remote := noconn.Remote()
	log.WithFields(log.Fields{
		"remote_addr": noconn.RemoteAddr(),
		"remote_node": remote,
	}).Debug("handshake success")

	s.serveStream(s.ctx, s.rpcServer, noconn, remote)
}

// RegisterService registers service with a Service name, used by Client RPC.
func (s *Server) RegisterService(name string, service interface{}) error {
	return s.rpcServer.RegisterName(name, service)
}

// Stop Server main loop.
func (s *Server) Stop() {
	if s.Listener != nil {
		_ = s.Listener.Close()
	}
	s.cancel()
}
