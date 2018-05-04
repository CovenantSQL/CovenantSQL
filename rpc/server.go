/*
 * MIT License
 *
 * Copyright (c) 2016-2018. ThunderDB
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package rpc

import (
	"github.com/hashicorp/yamux"
	log "github.com/sirupsen/logrus"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"sync"

	"github.com/thunderdb/ThunderDB/proto"
)

// ServiceMap map service name to service instance
type ServiceMap map[string]interface{}

// Server is the RPC server struct
type Server struct {
	node       *proto.Node
	sessions   sync.Map // map[id]*Session
	rpcServer  *rpc.Server
	stopCh     chan interface{}
	serviceMap ServiceMap
	listener   net.Listener
}

// NewServer return a new Server
func NewServer() *Server {
	return &Server{
		rpcServer:  rpc.NewServer(),
		stopCh:     make(chan interface{}),
		serviceMap: make(ServiceMap),
	}
}

// NewServerWithService also return a new Server, and also register the Server.ServiceMap
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

// SetListener set the service loop listener, used by func Serve main loop
func (s *Server) SetListener(l net.Listener) {
	s.listener = l
	return
}

// Serve start the Server main loop,
func (s *Server) Serve() {
serverLoop:
	for {
		select {
		case <-s.stopCh:
			log.Info("Stopping Server Loop")
			break serverLoop
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				log.Debug(err)
				continue
			}
			go s.handleConn(conn)
		}
	}
}

// handleConn do all the work
func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()

	sess, err := yamux.Server(conn, nil)
	if err != nil {
		log.Error(err)
		return
	}

	s.serveRPC(sess)
	log.Debugf("%s closed connection", conn.RemoteAddr())
}

// serveRPC install the JSON RPC codec
func (s *Server) serveRPC(sess *yamux.Session) {
	conn, err := sess.Accept()
	if err != nil {
		log.Error(err)
		return
	}
	s.rpcServer.ServeCodec(jsonrpc.NewServerCodec(conn))
}

// RegisterService with a Service name, used by Client RPC
func (s *Server) RegisterService(name string, service interface{}) error {
	return s.rpcServer.RegisterName(name, service)
}

// Stop Server main loop
func (s *Server) Stop() {
	close(s.stopCh)
}
