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
	"time"

	"github.com/pkg/errors"

	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
)

// PersistentCaller is a wrapper for session pooling and RPC calling.
type PersistentCaller struct {
	pool       NOConnPool
	client     *rpc.Client
	TargetAddr string
	TargetID   proto.NodeID
	sync.Mutex
}

// NewPersistentCallerWithPool returns a persistent RPCCaller.
//  IMPORTANT: If a PersistentCaller is firstly used by a DHT.Ping, which is an anonymous
//  ETLS connection. It should not be used by any other RPC except DHT.Ping.
func NewPersistentCallerWithPool(pool NOConnPool, target proto.NodeID) *PersistentCaller {
	return &PersistentCaller{
		pool:     pool,
		TargetID: target,
	}
}

func (c *PersistentCaller) initClient(isAnonymous bool) (err error) {
	c.Lock()
	defer c.Unlock()
	if c.client == nil {
		var conn net.Conn
		conn, err = DialToNodeWithPool(c.pool, c.TargetID, isAnonymous)
		if err != nil {
			err = errors.Wrap(err, "dial to node failed")
			return
		}
		//conn.SetDeadline(time.Time{})
		c.client = NewClient(conn)
		c.TargetAddr = conn.RemoteAddr().String()
	}
	return
}

// Call invokes the named function, waits for it to complete, and returns its error status.
func (c *PersistentCaller) Call(method string, args interface{}, reply interface{}) (err error) {
	startTime := time.Now()
	defer func() {
		recordRPCCost(startTime, method, err)
	}()

	err = c.initClient(method == route.DHTPing.String())
	if err != nil {
		err = errors.Wrap(err, "init PersistentCaller client failed")
		return
	}
	err = c.client.Call(method, args, reply)
	if err != nil {
		if err == io.EOF ||
			err == io.ErrUnexpectedEOF ||
			err == io.ErrClosedPipe ||
			err == rpc.ErrShutdown ||
			strings.Contains(strings.ToLower(err.Error()), "shut down") ||
			strings.Contains(strings.ToLower(err.Error()), "broken pipe") {
			// if got EOF, retry once
			reconnectErr := c.ResetClient()
			if reconnectErr != nil {
				err = errors.Wrap(reconnectErr, "reconnect failed")
			}
		}
		err = errors.Wrapf(err, "call %s failed", method)
		return
	}
	return
}

// ResetClient resets client.
func (c *PersistentCaller) ResetClient() (err error) {
	c.Lock()
	c.Close()
	c.client = nil
	c.Unlock()
	return
}

// Close closes the stream and RPC client.
func (c *PersistentCaller) Close() {
	_ = c.client.Close()
}
