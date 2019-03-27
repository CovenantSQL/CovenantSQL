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
package mux

import (
	"net"
	"net/rpc"

	"github.com/pkg/errors"
	mux "github.com/xtaci/smux"

	"github.com/CovenantSQL/CovenantSQL/proto"
	rrpc "github.com/CovenantSQL/CovenantSQL/rpc"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

// Client is RPC client.
type Client struct {
	*rpc.Client
	RemoteAddr string
	Conn       net.Conn
}

var (
	// MuxConfig holds the default mux config
	MuxConfig *mux.Config
)

func init() {
	MuxConfig = mux.DefaultConfig()
}

// DialToNodeWithPool ties use connection in pool, if fails then connects to the node with nodeID.
func DialToNodeWithPool(pool rrpc.NodeConnPool, nodeID proto.NodeID, isAnonymous bool) (conn net.Conn, err error) {
	if pool == nil || isAnonymous {
		var etlsconn net.Conn
		var sess *mux.Session
		etlsconn, err = rrpc.DialETLSEx(nodeID, isAnonymous)
		if err != nil {
			return
		}
		sess, err = mux.Client(etlsconn, MuxConfig)
		if err != nil {
			err = errors.Wrapf(err, "init yamux client to %s failed", nodeID)
			return
		}
		conn, err = sess.OpenStream()
		log.Debug("open new stream")
		if err != nil {
			err = errors.Wrapf(err, "open new session to %s failed", nodeID)
		}
		return
	}
	//log.WithField("poolSize", pool.Len()).Debug("session pool size")
	conn, err = pool.Get(nodeID)
	return
}

// NewClient returns a RPC client.
func NewClient() *Client {
	return &Client{}
}

// initClient initializes client with connection to given addr.
func initClient(addr string) (client *Client, err error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	etlsconn := rrpc.NewClientConn(conn)
	if err = etlsconn.Handshake(); err != nil {
		return
	}
	return InitClientConn(etlsconn)
}

// InitClientConn initializes client with connection to given addr.
func InitClientConn(conn net.Conn) (client *Client, err error) {
	client = NewClient()
	var muxConn *mux.Stream
	muxConn, ok := conn.(*mux.Stream)
	if !ok {
		var sess *mux.Session
		sess, err = mux.Client(conn, MuxConfig)
		if err != nil {
			err = errors.Wrap(err, "init mux client failed")
			return
		}

		muxConn, err = sess.OpenStream()
		if err != nil {
			err = errors.Wrap(err, "open stream failed")
			return
		}
	}
	client.Conn = muxConn
	client.Client = rpc.NewClientWithCodec(utils.GetMsgPackClientCodec(muxConn))
	client.RemoteAddr = conn.RemoteAddr().String()

	return client, nil
}

// Close the client RPC connection.
func (c *Client) Close() {
	//log.WithField("addr", c.RemoteAddr).Debug("closing client")
	_ = c.Client.Close()
}
