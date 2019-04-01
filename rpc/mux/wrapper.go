package mux

import (
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

type ServiceMap rpc.ServiceMap

// Server is the RPC server struct.
type Server struct {
	*rpc.Server
}

// NewServer return a new Server.
func NewServer() *Server {
	return &Server{
		Server: rpc.NewServerWithServeFunc(ServeMux),
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
	return
}

// Caller is a wrapper for session pooling and RPC calling.
type Caller struct {
	*rpc.Caller
}

// NewCaller returns a new RPCCaller.
func NewCaller() *Caller {
	return &Caller{
		Caller: rpc.NewCallerWithPool(defaultPool),
	}
}

// PersistentCaller is a wrapper for session pooling and RPC calling.
type PersistentCaller struct {
	*rpc.PersistentCaller
}

// NewPersistentCaller returns a persistent RPCCaller.
//  IMPORTANT: If a PersistentCaller is firstly used by a DHT.Ping, which is an anonymous
//  ETLS connection. It should not be used by any other RPC except DHT.Ping.
func NewPersistentCaller(target proto.NodeID) *PersistentCaller {
	return &PersistentCaller{
		PersistentCaller: rpc.NewPersistentCallerWithPool(defaultPool, target),
	}
}
