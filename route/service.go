package route

import (
	"fmt"
	log "github.com/sirupsen/logrus"
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

// FindValue RPC returns FindValueReq.Count closest node from DHT
func (DHT *DHTService) FindValue(req *proto.FindValueReq, resp *proto.FindValueResp) (err error) {
	resp.Nodes, err = DHT.hashRing.GetN(string(req.NodeID), req.Count)
	if err != nil {
		log.Error(err)
		resp.Msg = fmt.Sprint(err)
	}
	return
}

// Ping RPC add PingReq.Node to DHT
func (DHT *DHTService) Ping(req *proto.PingReq, resp *proto.PingResp) (err error) {
	DHT.hashRing.Add(req.Node)
	resp = new(proto.PingResp)
	resp.Msg = "Pong"
	return
}
