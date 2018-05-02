package route

import (
	log "github.com/sirupsen/logrus"
	. "github.com/smartystreets/goconvey/convey"
	. "github.com/thunderdb/ThunderDB/proto"
	"github.com/thunderdb/ThunderDB/rpc"
	"net"
	"testing"
)

func TestPingFindValue(t *testing.T) {
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

	reqA := &PingReq{
		Node: Node{
			ID: "node1",
		},
	}
	respA := new(PingResp)
	err = client.Call("DHT.Ping", reqA, respA)
	if err != nil {
		log.Fatal(err)
	}
	log.Debugf("respA: %v", respA)

	reqB := &PingReq{
		Node: Node{
			ID: "node2",
		},
	}
	respB := new(PingResp)
	err = client.Call("DHT.Ping", reqB, respB)
	if err != nil {
		log.Fatal(err)
	}
	log.Debugf("respA: %v", respB)

	req := &FindValueReq{
		NodeID: "123",
		Count:  2,
	}
	resp := new(FindValueResp)
	err = client.Call("DHT.FindValue", req, resp)
	if err != nil {
		log.Fatal(err)
	}
	log.Debugf("resp: %v", resp)
	nodeIDList := []string{
		string(resp.Nodes[0].ID),
		string(resp.Nodes[1].ID),
	}
	Convey("test FindValue", t, func() {
		So(nodeIDList, ShouldContain, "node1")
		So(nodeIDList, ShouldContain, "node2")
	})

	client.Close()
	dhtServer.Stop()
}
