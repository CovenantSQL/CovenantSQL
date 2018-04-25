package rpc

import (
	"net"
	"net/rpc"
	"github.com/hashicorp/yamux"
	log "github.com/sirupsen/logrus"
	"net/rpc/jsonrpc"
)

type Client struct {
	*rpc.Client
}

func NewClient() *Client {
	return &Client{}
}

func InitClient(addr string) (client *Client, err error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	client = NewClient()
	client.start(conn)
	return client,nil
}


func (c *Client) start(conn net.Conn) {
	sess, err := yamux.Client(conn, nil)
	if err != nil {
		log.Panic(err)
	}

	clientConn, err := sess.Open()
	if err != nil {
		log.Panic(err)
		return
	}
	c.Client = rpc.NewClientWithCodec(jsonrpc.NewClientCodec(clientConn))
}

func (c *Client) Close() {
	c.Client.Close()
}
