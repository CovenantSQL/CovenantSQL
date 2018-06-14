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

package rpc

import (
	"net"
	"net/rpc"
	"sync"

	"github.com/hashicorp/yamux"
	log "github.com/sirupsen/logrus"
	"github.com/thunderdb/ThunderDB/conf"
	"github.com/thunderdb/ThunderDB/crypto/asymmetric"
	"github.com/thunderdb/ThunderDB/crypto/etls"
	"github.com/thunderdb/ThunderDB/crypto/hash"
	"github.com/thunderdb/ThunderDB/crypto/kms"
	"github.com/thunderdb/ThunderDB/proto"
	"github.com/thunderdb/ThunderDB/route"
	"github.com/ugorji/go/codec"
)

// ServiceMap maps service name to service instance
type ServiceMap map[string]interface{}

// Server is the RPC server struct
type Server struct {
	publicKeyStore *kms.PublicKeyStore
	node           *proto.Node
	sessions       sync.Map // map[id]*Session
	rpcServer      *rpc.Server
	stopCh         chan interface{}
	serviceMap     ServiceMap
	Listener       net.Listener
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
	route.InitResolver()

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
			conn, err := s.Listener.Accept()
			if err != nil {
				log.Error(err)
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
	msgpackCodec := codec.MsgpackSpecRpc.ServerCodec(conn, &codec.MsgpackHandle{})
	s.rpcServer.ServeCodec(msgpackCodec)
}

// RegisterService with a Service name, used by Client RPC
func (s *Server) RegisterService(name string, service interface{}) error {
	return s.rpcServer.RegisterName(name, service)
}

// Stop Server main loop
func (s *Server) Stop() {
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
	nodeID := proto.NodeID(idHash.String())
	// TODO(auxten): compute the nonce and check difficulty
	// cpuminer.FromBytes(headerBuf[hash.HashBSize:])

	publicKey, err := kms.GetPublicKey(nodeID)
	if err != nil {
		if conf.Role[0] == 'M' && err == kms.ErrKeyNotFound {
			// TODO(auxten): if Miner running and key not found, ask BlockProducer
		}
		log.Errorf("get public key failed, node id: %s, err: %s", nodeID, err)
		return
	}
	privateKey, err := kms.GetLocalPrivateKey()
	if err != nil {
		log.Errorf("get local private key failed: %s", err)
		return
	}

	symmetricKey := asymmetric.GenECDHSharedSecret(privateKey, publicKey)
	cipher := etls.NewCipher(symmetricKey)
	cryptoConn = etls.NewConn(conn, cipher)

	return
}
