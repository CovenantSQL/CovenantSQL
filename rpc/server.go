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
	"net"
	"net/rpc"

	"github.com/CovenantSQL/CovenantSQL/crypto/etls"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/pow/cpuminer"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

// ServiceMap maps service name to service instance
type ServiceMap map[string]interface{}

// Server is the RPC server struct
type Server struct {
	rpcServer  *rpc.Server
	stopCh     chan interface{}
	serviceMap ServiceMap
	Listener   net.Listener
}

// NewServer return a new Server
func NewServer() *Server {
	return &Server{
		rpcServer:  rpc.NewServer(),
		stopCh:     make(chan interface{}),
		serviceMap: make(ServiceMap),
	}
}

// InitRPCServer load the private key, init the crypto transfer layer and register RPC
// services.
// IF ANY ERROR returned, please raise a FATAL
func (s *Server) InitRPCServer(
	addr string,
	privateKeyPath string,
	masterKey []byte,
) (err error) {
	//route.InitResolver()

	err = kms.InitLocalKeyPair(privateKeyPath, masterKey)
	if err != nil {
		log.Errorf("init local key pair failed: %s", err)
		return
	}

	l, err := etls.NewCryptoListener("tcp", addr, handleCipher)
	if err != nil {
		log.Errorf("create crypto listener failed: %s", err)
		return
	}

	s.SetListener(l)

	return
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
	s.Listener = l
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
			conn, err := s.Listener.Accept()
			if err != nil {
				continue
			}
			go s.handleConn(conn)
		}
	}
}

// handleConn do all the work
func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()

	// remote remoteNodeID connection awareness
	var remoteNodeID *proto.RawNodeID

	if c, ok := conn.(*etls.CryptoConn); ok {
		// set node id
		remoteNodeID = c.NodeID
	}

	log.Debugf("session accepted for %v", remoteNodeID)
	nodeAwareCodec := NewNodeAwareServerCodec(utils.GetMsgPackServerCodec(conn), remoteNodeID)
	s.rpcServer.ServeCodec(nodeAwareCodec)

	log.Debugf("Server.handleConn finished for %s %s", remoteNodeID, conn.RemoteAddr())
}

// RegisterService with a Service name, used by Client RPC
func (s *Server) RegisterService(name string, service interface{}) error {
	return s.rpcServer.RegisterName(name, service)
}

// Stop Server main loop
func (s *Server) Stop() {
	if s.Listener != nil {
		s.Listener.Close()
	}
	close(s.stopCh)
}

func handleCipher(conn net.Conn) (cryptoConn *etls.CryptoConn, err error) {
	// NodeID + Uint256 Nonce
	headerBuf := make([]byte, hash.HashBSize+32)
	rCount, err := conn.Read(headerBuf)
	if err != nil || rCount != hash.HashBSize+32 {
		log.Errorf("read node header error: %s", err)
		return
	}

	// headerBuf len is hash.HashBSize, so there won't be any error
	idHash, _ := hash.NewHash(headerBuf[:hash.HashBSize])
	rawNodeID := &proto.RawNodeID{Hash: *idHash}
	// TODO(auxten): compute the nonce and check difficulty
	cpuminer.Uint256FromBytes(headerBuf[hash.HashBSize:])

	symmetricKey, err := GetSharedSecretWith(
		rawNodeID,
		rawNodeID.IsEqual(&kms.AnonymousRawNodeID.Hash),
	)
	if err != nil {
		log.Errorf("get shared secret for %s failed: %s", rawNodeID.ToNodeID(), err)
		return
	}
	cipher := etls.NewCipher(symmetricKey)
	cryptoConn = etls.NewConn(conn, cipher, rawNodeID)

	return
}
