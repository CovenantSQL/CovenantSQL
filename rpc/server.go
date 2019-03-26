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
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

// ServiceMap maps service name to service instance.
type ServiceMap map[string]interface{}

// Server is the RPC server struct.
type Server struct {
	rpcServer  *rpc.Server
	stopCh     chan interface{}
	serviceMap ServiceMap
	Listener   net.Listener
}

// NewServer return a new Server.
func NewServer() *Server {
	return &Server{
		rpcServer:  rpc.NewServer(),
		stopCh:     make(chan interface{}),
		serviceMap: make(ServiceMap),
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

// NewServerWithService also return a new Server, and also register the Server.ServiceMap.
func NewServerWithService(serviceMap ServiceMap) (server *Server, err error) {
	server = NewServer()
	for k, v := range serviceMap {
		err = server.RegisterService(k, v)
		if err != nil {
			log.Fatal(err)
			return nil, err
		}
	}
	return server, nil
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
		case <-s.stopCh:
			log.Info("stopping Server Loop")
			break serverLoop
		default:
			conn, err := s.Listener.Accept()
			if err != nil {
				continue
			}
			log.WithField("remote", conn.RemoteAddr().String()).Info("accept")
			go s.handleConn(conn)
		}
	}
}

// handleConn do all the work.
func (s *Server) handleConn(conn net.Conn) {
	etlsconn := NewServerConn(conn)
	defer etlsconn.Close() // take ownership of conn

	// TODO(leventeliu): consider to do handshake in the first Read/Write like tls.
	var err error
	if err = etlsconn.Handshake(); err != nil {
		return
	}

	// Serve
	var remoteNodeID = etlsconn.RemoteNodeID()
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	nodeAwareCodec := NewNodeAwareServerCodec(ctx, utils.GetMsgPackServerCodec(conn), &remoteNodeID)
	s.rpcServer.ServeCodec(nodeAwareCodec)
}

// RegisterService with a Service name, used by Client RPC.
func (s *Server) RegisterService(name string, service interface{}) error {
	return s.rpcServer.RegisterName(name, service)
}

// Stop Server main loop.
func (s *Server) Stop() {
	if s.Listener != nil {
		s.Listener.Close()
	}
	close(s.stopCh)
}
