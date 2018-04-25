package route

import (
	"fmt"
	"github.com/opentracing/opentracing-go/log"
	"github.com/thunderdb/ThunderDB/consistent"
	"github.com/thunderdb/ThunderDB/proto"
)

type DhtService struct {
	thisNode proto.Node
	hashRing *consistent.Consistent
}

func NewDhtService() *DhtService {
	return &DhtService{
		hashRing: consistent.New(),
	}
}

type GetNeighborsReq struct {
	NodeId  proto.NodeId
	Count   int
	Version string
}

type GetNeighborsResp struct {
	Nodes   []proto.Node
	ErrMsg  string
	Version string
}

func (dht *DhtService) GetNeighbors(req *GetNeighborsReq, resp *GetNeighborsResp) (err error) {
	resp.Nodes, err = dht.hashRing.GetN(string(req.NodeId), req.Count)
	if err != nil {
		log.Error(err)
		resp.ErrMsg = fmt.Sprint(err)
	}
	return
}

type AddNodeReq struct {
	Node    proto.Node
	Version string
}

type AddNodeResp struct {
	ErrMsg  string
	Version string
}

func (dht *DhtService) AddNode(req *AddNodeReq, resp *AddNodeResp) (err error) {
	dht.hashRing.Add(req.Node)
	return
}
