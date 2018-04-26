package proto

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
