package rpc

import (
	"net"

	"github.com/CovenantSQL/CovenantSQL/noconn"
	"github.com/CovenantSQL/CovenantSQL/proto"
)

// TODO(leventeliu): allow to config other node-oriented connection dialer/accepter.
var (
	Dial   = noconn.Dial
	DialEx = noconn.DialEx
	Accept = noconn.Accept
)

type NodeConnPool interface {
	Get(remote proto.NodeID) (net.Conn, error)
	GetEx(remote proto.NodeID, isAnonymous bool) (net.Conn, error)
	Close() error
}

// DialToNode ties use connection in pool, if fails then connects to the node with nodeID.
func DialToNodeWithPool(pool NodeConnPool, nodeID proto.NodeID, isAnonymous bool) (conn net.Conn, err error) {
	if isAnonymous {
		return pool.GetEx(nodeID, true)
	}
	//log.WithField("poolSize", pool.Len()).Debug("session pool size")
	return pool.Get(nodeID)
}
