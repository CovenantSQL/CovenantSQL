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

package rpc

import (
	"github.com/hashicorp/yamux"
	log "github.com/sirupsen/logrus"
	"gitlab.com/thunderdb/ThunderDB/proto"
	"gitlab.com/thunderdb/ThunderDB/route"
)

// Caller is a wrapper for session pooling and RPC calling.
type Caller struct {
	pool *SessionPool
}

// NewCaller returns a new RPCCaller.
func NewCaller() *Caller {
	return &Caller{
		pool: GetSessionPoolInstance(),
	}
}

// CallNode invokes the named function, waits for it to complete, and returns its error status.
func (c *Caller) CallNode(
	node proto.NodeID, method string, args interface{}, reply interface{}) (err error) {
	conn, err := DialToNode(node, c.pool)
	if err != nil {
		log.Errorf("dialing to node: %s failed: %s", node, err)
		return
	}

	client, err := InitClientConn(conn)
	if err != nil {
		log.Errorf("init RPC client failed: %s", err)
		return
	}

	defer func() {
		// call the yamux stream Close explicitly
		//TODO(auxten) maybe a rpc client pool will gain much more performance
		stream, ok := conn.(*yamux.Stream)
		if ok {
			stream.Close()
		}
	}()
	return client.Call(method, args, reply)
}

// GetNodeAddr tries best to get node addr
func GetNodeAddr(id *proto.RawNodeID) (addr string, err error) {
	addr, err = route.GetNodeAddrCache(id)
	if err != nil {
		log.Infof("get node: %s addr failed: %s", addr, err)
		if err == route.ErrUnknownNodeID {

		}
	}
	return
}
