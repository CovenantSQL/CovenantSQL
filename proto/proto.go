package proto

import (
	"time"
)

// NodeID is node name, will be generated from Hash(nodePublicKey)
type NodeID string

// NodeKey is node key on consistent hash ring, generate from Hash(NodeID)
type NodeKey uint64

// Node is all node info struct
type Node struct {
	Name      string
	Port      uint16
	Protocol  string
	ID        NodeID
	PublicKey string
}

// Envelope is the protocol
type Envelope struct {
	Version string
	TTL     time.Duration
	Expire  time.Duration
}

// PingReq is Ping RPC request
type PingReq struct {
	Node    Node
	Version string
	Envelope
}

// PingResp is Ping RPC response, i.e. Pong
type PingResp struct {
	Msg     string
	Version string
	Envelope
}

// FindValueReq is FindValue RPC request
type FindValueReq struct {
	NodeID  NodeID
	Count   int
	Version string
	Envelope
}

// FindValueResp is FindValue RPC response
type FindValueResp struct {
	Nodes   []Node
	Msg     string
	Version string
	Envelope
}
