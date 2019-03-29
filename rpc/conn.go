package rpc

import (
	"net"

	"github.com/CovenantSQL/CovenantSQL/csconn"
	"github.com/CovenantSQL/CovenantSQL/proto"
)

// TODO(leventeliu): allow to config other node connection dialer/accepter.
var (
	Dial   = csconn.DialETLS
	DialEx = csconn.DialETLSEx
	Accept = csconn.AcceptETLS
)

type NodeConnPool interface {
	Get(remote proto.NodeID) (net.Conn, error)
	GetEx(remote proto.NodeID, isAnonymous bool) (net.Conn, error)
	Close()
}

// DialToNode ties use connection in pool, if fails then connects to the node with nodeID.
func DialToNodeWithPool(pool NodeConnPool, nodeID proto.NodeID, isAnonymous bool) (conn net.Conn, err error) {
	if isAnonymous {
		return pool.GetEx(nodeID, true)
	}
	//log.WithField("poolSize", pool.Len()).Debug("session pool size")
	return pool.Get(nodeID)
}
