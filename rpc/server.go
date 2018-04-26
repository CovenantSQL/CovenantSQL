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

type ServiceMap map[string]interface{}

type Server struct {
	node       *proto.Node
	sessions   sync.Map // map[id]*Session
	rpcServer  *rpc.Server
	stopCh     chan interface{}
	serviceMap ServiceMap
	listener   net.Listener
}

func NewServer() *Server {
	return &Server{
		rpcServer:  rpc.NewServer(),
		stopCh:     make(chan interface{}),
		serviceMap: make(ServiceMap),
	}
}

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

func (s *Server) SetListener(l net.Listener) {
	s.listener = l
	return
}

func (s *Server) serveRPC(sess *yamux.Session) {
	conn, err := sess.Accept()
	if err != nil {
		log.Error(err)
		return
	}
	s.rpcServer.ServeCodec(jsonrpc.NewServerCodec(conn))
}

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

func (s *Server) RegisterService(name string, service interface{}) error {
	return s.rpcServer.RegisterName(name, service)
}

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

func (s *Server) Close() {
	close(s.stopCh)
}
