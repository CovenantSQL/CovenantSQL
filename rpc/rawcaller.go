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

	"github.com/pkg/errors"
)

// PCaller defines generic interface shared with PersistentCaller and RawCaller.
type PCaller interface {
	Call(method string, request interface{}, reply interface{}) (err error)
	Close()
	Target() string
	New() PCaller // returns new instance of current caller
}

// RawCaller defines a raw rpc caller without any encryption.
type RawCaller struct {
	targetAddr string
	client     *Client
	sync.RWMutex
}

// NewRawCaller creates the raw rpc caller to target node.
func NewRawCaller(targetAddr string) *RawCaller {
	return &RawCaller{
		targetAddr: targetAddr,
	}
}

func (c *RawCaller) isClientValid() bool {
	c.RLock()
	defer c.RUnlock()

	return c.client != nil
}

func (c *RawCaller) resetClient() (err error) {
	c.Lock()
	defer c.Unlock()

	if c.client != nil {
		c.client.Close()
		c.client = nil
	}

	var conn net.Conn
	if conn, err = net.Dial("tcp", c.targetAddr); err != nil {
		err = errors.Wrapf(err, "dial to target %s failed", c.targetAddr)
		return
	}

	if c.client, err = InitClientConn(conn); err != nil {
		c.client = nil
		err = errors.Wrapf(err, "init client to target %s failed", c.targetAddr)
		return
	}

	return
}

// Call issues client rpc call.
func (c *RawCaller) Call(method string, args interface{}, reply interface{}) (err error) {
	if !c.isClientValid() {
		if err = c.resetClient(); err != nil {
			return
		}
	}

	c.RLock()
	err = c.client.Call(method, args, reply)
	c.RUnlock()

	if err != nil {
		if err == io.EOF ||
			err == io.ErrUnexpectedEOF ||
			err == io.ErrClosedPipe ||
			err == rpc.ErrShutdown ||
			strings.Contains(strings.ToLower(err.Error()), "shut down") ||
			strings.Contains(strings.ToLower(err.Error()), "broken pipe") {
			// if got EOF, retry once
			reconnectErr := c.resetClient()
			if reconnectErr != nil {
				err = errors.Wrap(reconnectErr, "reconnect failed")
			}
		}
		err = errors.Wrapf(err, "call %s failed", method)
	}
	return
}

// Close release underlying connection resources.
func (c *RawCaller) Close() {
	c.Lock()
	defer c.Unlock()
	if c.client != nil {
		c.client.Close()
		c.client = nil
	}
}

// Target returns the request target for logging purpose.
func (c *RawCaller) Target() string {
	return c.targetAddr
}

// New returns brand new caller.
func (c *RawCaller) New() PCaller {
	return NewRawCaller(c.targetAddr)
}
