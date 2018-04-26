package route

import (
	"fmt"
	"github.com/opentracing/opentracing-go/log"
	"github.com/thunderdb/ThunderDB/consistent"
	"github.com/thunderdb/ThunderDB/proto"
)

// DHTService is server side RPC implementation
type DHTService struct {
	thisNode proto.Node
	hashRing *consistent.Consistent
}

// NewDHTService will return a new DHTService
func NewDHTService() *DHTService {
	return &DHTService{
		hashRing: consistent.New(),
	}
}

// GetNeighborsReq is GetNeighbors RPC request
type GetNeighborsReq struct {
	nodeID  proto.NodeID
	Count   int
	Version string
}

// GetNeighborsResp is GetNeighbors RPC response
type GetNeighborsResp struct {
	Nodes   []proto.Node
	ErrMsg  string
	Version string
}

// GetNeighbors RPC returns GetNeighborsReq.Count closest node from consistent hash ring
func (DHT *DHTService) GetNeighbors(req *GetNeighborsReq, resp *GetNeighborsResp) (err error) {
	resp.Nodes, err = DHT.hashRing.GetN(string(req.nodeID), req.Count)
	if err != nil {
		log.Error(err)
		resp.ErrMsg = fmt.Sprint(err)
	}
	return
}

// AddNodeReq is AddNode RPC request
type AddNodeReq struct {
	Node    proto.Node
	Version string
}

// AddNodeResp is AddNode RPC response
type AddNodeResp struct {
	ErrMsg  string
	Version string
}

// AddNode RPC add AddNodeReq.Node to consistent hash ring
func (DHT *DHTService) AddNode(req *AddNodeReq, resp *AddNodeResp) (err error) {
	DHT.hashRing.Add(req.Node)
	return
}
