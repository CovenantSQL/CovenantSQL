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

package sqlchain

import (
	"gitlab.com/thunderdb/ThunderDB/proto"
	"gitlab.com/thunderdb/ThunderDB/rpc"
)

// TODO(leventeliu): move this file to rpc package for common use.

// rpcCaller is a wrapper for session pooling and RPC calling.
type rpcCaller struct {
	pool *rpc.SessionPool
}

// newRPCCaller returns a new RPCCaller.
func newRPCCaller() *rpcCaller {
	return &rpcCaller{
		pool: rpc.GetSessionPoolInstance(),
	}
}

// call invokes the named function, waits for it to complete, and returns its error status.
func (c *rpcCaller) call(
	node proto.NodeID, method string, args interface{}, reply interface{}) (err error) {
	conn, err := rpc.DialToNode(node, c.pool)

	if err != nil {
		return
	}

	cli, err := rpc.InitClientConn(conn)

	if err != nil {
		return
	}

	defer cli.Close()
	return cli.Call(method, args, reply)
}
