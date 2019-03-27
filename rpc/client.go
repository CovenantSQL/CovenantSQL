/*
 * Copyright 2018 The CovenantSQL Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package mux provides RPC Client/Server functions.
package rpc

import (
	"net"
	"net/rpc"

	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils"
)

// Client is RPC client.
type Client struct {
	*rpc.Client
	RemoteAddr string
	Conn       net.Conn
}

var (
	// DefaultNodeDialer holds the default dialer of SessionPool
	DefaultNodeDialer func(nodeID proto.NodeID) (conn net.Conn, err error)
)

func init() {
	DefaultNodeDialer = DialETLS
}

// NewClient returns a RPC client.
func NewClient() *Client {
	return &Client{}
}

func NewRPCClient(conn net.Conn) (client *Client) {
	return &Client{
		Conn:       conn,
		Client:     rpc.NewClientWithCodec(utils.GetMsgPackClientCodec(conn)),
		RemoteAddr: conn.RemoteAddr().String(),
	}
}

// Close the client RPC connection.
func (c *Client) Close() {
	//log.WithField("addr", c.RemoteAddr).Debug("closing client")
	_ = c.Client.Close()
}
