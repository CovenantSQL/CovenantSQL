package rpc

import (
	"github.com/hashicorp/yamux"
	log "github.com/sirupsen/logrus"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
)

// Client is RPC client
type Client struct {
	*rpc.Client
}

// NewClient return a RPC client
func NewClient() *Client {
	return &Client{}
}

// InitClient init client with connection to given addr
func InitClient(addr string) (client *Client, err error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	client = NewClient()
	client.start(conn)
	return client, nil
}

// start init session and set RPC codec
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

// Close the client RPC connection
func (c *Client) Close() {
	c.Client.Close()
}
