package route

import (
	log "github.com/sirupsen/logrus"
	"github.com/thunderdb/ThunderDB/proto"
	"github.com/thunderdb/ThunderDB/rpc"
	"github.com/thunderdb/ThunderDB/utils"
	"net"
	"testing"
)

func TestGetNeighbors(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	addr := "127.0.0.1:0"
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}

	dhtServer, err := InitDHTserver(l)
	if err != nil {
		log.Fatal(err)
	}

	go dhtServer.Serve()

	client, err := rpc.InitClient(l.Addr().String())
	if err != nil {
		log.Fatal(err)
	}

	reqA := &AddNodeReq{
		Node: proto.Node{
			ID: "node1",
		},
	}
	respA := new(AddNodeResp)
	err = client.Call("DHT.AddNode", reqA, respA)
	if err != nil {
		log.Fatal(err)
	}
	log.Debugf("respA: %v", respA)

	req := &GetNeighborsReq{
		nodeID: "123",
		Count:  2,
	}
	resp := new(GetNeighborsResp)
	err = client.Call("DHT.GetNeighbors", req, resp)
	if err != nil {
		log.Fatal(err)
	}
	log.Debugf("resp: %v", resp)
	utils.CheckStr(string(resp.Nodes[0].ID), "node1", t)

	client.Close()
	dhtServer.Stop()
}
