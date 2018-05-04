/*
 * MIT License
 *
 * Copyright (c) 2016-2018. ThunderDB
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

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
