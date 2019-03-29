package mux

import (
	"github.com/CovenantSQL/CovenantSQL/proto"
	rrpc "github.com/CovenantSQL/CovenantSQL/rpc"
)

// PersistentCaller is a wrapper for session pooling and RPC calling.
type PersistentCaller struct {
	*rrpc.PersistentCaller
}

// NewPersistentCaller returns a persistent RPCCaller.
//  IMPORTANT: If a PersistentCaller is firstly used by a DHT.Ping, which is an anonymous
//  ETLS connection. It should not be used by any other RPC except DHT.Ping.
func NewPersistentCaller(target proto.NodeID) *PersistentCaller {
	return &PersistentCaller{
		PersistentCaller: rrpc.NewPersistentCaller(GetSessionPoolInstance(), target),
	}
}
