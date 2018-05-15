/*
 * Copyright 2018 The ThunderDB Authors.
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

// Package rpc provides RPC Client/Server functions
package rpc

import (
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"

	"github.com/hashicorp/yamux"
	log "github.com/sirupsen/logrus"
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
