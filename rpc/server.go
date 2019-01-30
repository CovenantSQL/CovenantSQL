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
	"io"
	"net"
	"net/rpc"

	"github.com/CovenantSQL/CovenantSQL/crypto/etls"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/pow/cpuminer"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/pkg/errors"
	mux "github.com/xtaci/smux"
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
		err = errors.Wrap(err, "init local key pair failed")
		return
	}

	l, err := etls.NewCryptoListener("tcp", addr, handleCipher)
	if err != nil {
		err = errors.Wrap(err, "create crypto listener failed")
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

// handleConn do all the work
func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()

	// remote remoteNodeID connection awareness
	var remoteNodeID *proto.RawNodeID
	var err error

	if c, ok := conn.(*etls.CryptoConn); ok {
		conn, err = s.Listener.(*etls.CryptoListener).CHandler(c.Conn)
		if err != nil {
			err = errors.Wrap(err, "handle ETLS handler failed")
			return
		}
		// set node id
		remoteNodeID = conn.(*etls.CryptoConn).NodeID
	}

	sess, err := mux.Server(conn, MuxConfig)
	if err != nil {
		err = errors.Wrap(err, "create mux server failed")
		return
	}
	defer sess.Close()

sessionLoop:
	for {
		select {
		case <-s.stopCh:
			log.Info("stopping Session Loop")
			break sessionLoop
		default:
			muxConn, err := sess.AcceptStream()
			if err != nil {
				if err == io.EOF {
					//log.WithField("remote", remoteNodeID).Debug("session connection closed")
				} else {
					err = errors.Wrapf(err, "session accept failed, remote: %s", remoteNodeID)
				}
				break sessionLoop
			}
			ctx, cancelFunc := context.WithCancel(context.Background())
			go func() {
				<-muxConn.GetDieCh()
				cancelFunc()
			}()
			nodeAwareCodec := NewNodeAwareServerCodec(ctx, utils.GetMsgPackServerCodec(muxConn), remoteNodeID)
			go s.rpcServer.ServeCodec(nodeAwareCodec)
		}
	}
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
	headerBuf := make([]byte, ETLSHeaderSize)
	rCount, err := conn.Read(headerBuf)
	if err != nil {
		err = errors.Wrap(err, "read node header error")
		return
	}

	if rCount != ETLSHeaderSize {
		err = errors.New("invalid ETLS header size")
		return
	}

	if headerBuf[0] != etls.ETLSMagicBytes[0] || headerBuf[1] != etls.ETLSMagicBytes[1] {
		err = errors.New("bad ETLS header")
		return
	}

	// headerBuf len is hash.HashBSize, so there won't be any error
	idHash, _ := hash.NewHash(headerBuf[2 : 2+hash.HashBSize])
	rawNodeID := &proto.RawNodeID{Hash: *idHash}
	// TODO(auxten): compute the nonce and check difficulty
	cpuminer.Uint256FromBytes(headerBuf[2+hash.HashBSize:])

	symmetricKey, err := GetSharedSecretWith(
		rawNodeID,
		rawNodeID.IsEqual(&kms.AnonymousRawNodeID.Hash),
	)
	if err != nil {
		err = errors.Wrapf(err, "get shared secret, target: %s", rawNodeID.String())
		return
	}
	cipher := etls.NewCipher(symmetricKey)
	cryptoConn = etls.NewConn(conn, cipher, rawNodeID)

	return
}
