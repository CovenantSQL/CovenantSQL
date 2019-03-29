package mux

import rrpc "github.com/CovenantSQL/CovenantSQL/rpc"

// Caller is a wrapper for session pooling and RPC calling.
type Caller struct {
	*rrpc.Caller
}

// NewCaller returns a new RPCCaller.
func NewCaller() *Caller {
	return &Caller{
		Caller: rrpc.NewCaller(GetSessionPoolInstance()),
	}
}
