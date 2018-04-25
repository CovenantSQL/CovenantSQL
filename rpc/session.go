package rpc

import (
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
)

type Session struct {
	*rpc.Client
}

func NewSession(conn net.Conn) *Session {
	return &Session{
		Client: rpc.NewClientWithCodec(jsonrpc.NewClientCodec(conn)),
	}
}

func (s *Session) Close() {
	s.Client.Close()
}
