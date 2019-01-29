/*
 * Copyright 2019 The CovenantSQL Authors.
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

package rpc

import (
	"io"
	"net"
	"net/rpc"
	"strings"
	"sync"

	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/pkg/errors"
)

// ClientPoolCaller is a wrapper for session pooling and client pooling.
type ClientPoolCaller struct {
	sessPool   *SessionPool
	clientPool sync.Pool
	TargetID   proto.NodeID
}

// NewClientPoolCaller returns a client/session pool caller.
func NewClientPoolCaller(target proto.NodeID) (c *ClientPoolCaller) {
	c = &ClientPoolCaller{
		TargetID: target,
		sessPool: GetSessionPoolInstance(),
	}

	c.clientPool = sync.Pool{
		New: c.initClient,
	}

	return
}

func (c *ClientPoolCaller) initClient() interface{} {
	var (
		client *Client
		err    error
	)

	client, err = c.initClientEx(false)
	if err != nil {
		return err
	}

	return client
}

func (c *ClientPoolCaller) initClientEx(isAnonymous bool) (client *Client, err error) {
	var conn net.Conn
	conn, err = DialToNode(c.TargetID, c.sessPool, isAnonymous)
	if err != nil {
		err = errors.Wrap(err, "dial to node failed")
		return
	}
	client, err = InitClientConn(conn)
	if err != nil {
		err = errors.Wrap(err, "init RPC client failed")
		return
	}
	return
}

func (c *ClientPoolCaller) allocClient(isAnonymous bool) (client *Client, err error) {
	if isAnonymous {
		return c.initClientEx(true)
	}

	rawClient := c.clientPool.Get()

	if rawClient == nil {
		err = errors.New("no available client")
		return
	}

	switch v := rawClient.(type) {
	case *Client:
		client = v
	case error:
		err = v
	}

	return
}

func (c *ClientPoolCaller) Call(method string, args interface{}, reply interface{}) (err error) {
	var (
		isAnonymous = method == route.DHTPing.String()
		client      *Client
	)

	client, err = c.allocClient(isAnonymous)
	if err != nil {
		return
	}

	err = client.Call(method, args, reply)
	if !isAnonymous && (err == nil || (err != io.EOF &&
		err != io.ErrUnexpectedEOF &&
		err != io.ErrClosedPipe &&
		err != rpc.ErrShutdown &&
		!strings.Contains(strings.ToLower(err.Error()), "shut down") &&
		!strings.Contains(strings.ToLower(err.Error()), "broken pipe"))) {
		// put back connection
		c.clientPool.Put(client)
	} else {
		// close
		client.Close()
	}

	if err != nil {
		err = errors.Wrapf(err, "call %s failed", method)
	}

	return
}

func (c *ClientPoolCaller) Close() {
	// free pool
}
