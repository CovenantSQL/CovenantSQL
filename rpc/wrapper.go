package rpc

import (
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

// NewServer return a new Server.
func NewServer() *Server {
	return NewServerWithServeFunc(ServeDirect)
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

// NewCaller returns a new RPCCaller.
func NewCaller() *Caller {
	return NewCallerWithPool(defaultPool)
}

// NewPersistentCaller returns a persistent RPCCaller.
//  IMPORTANT: If a PersistentCaller is firstly used by a DHT.Ping, which is an anonymous
//  ETLS connection. It should not be used by any other RPC except DHT.Ping.
func NewPersistentCaller(target proto.NodeID) *PersistentCaller {
	return NewPersistentCallerWithPool(defaultPool, target)
}
