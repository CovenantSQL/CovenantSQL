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
