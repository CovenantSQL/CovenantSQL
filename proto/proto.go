package proto



type NodeId 	string
type NodeKey	uint64

type Node struct {
	Name      string
	Port      uint16
	Protocol  string
	Id        NodeId
	PublicKey string
}

