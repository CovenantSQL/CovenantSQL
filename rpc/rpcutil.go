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
	"gitlab.com/thunderdb/ThunderDB/proto"
	"gitlab.com/thunderdb/ThunderDB/route"
	"gitlab.com/thunderdb/ThunderDB/utils/log"
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

	//defer func() {
	//	// call the yamux stream Close explicitly
	//	//TODO(auxten) maybe a rpc client pool will gain much more performance
	//	stream, ok := conn.(*yamux.Stream)
	//	if ok {
	//		stream.Close()
	//	}
	//}()
	return client.Call(method, args, reply)
}

// GetNodeAddr tries best to get node addr
func GetNodeAddr(id *proto.RawNodeID) (addr string, err error) {
	addr, err = route.GetNodeAddrCache(id)
	if err != nil {
		log.Infof("get node \"%s\" addr failed: %s", addr, err)
		if err == route.ErrUnknownNodeID {
			BPs := route.GetBPs()
			if len(BPs) == 0 {
				log.Errorf("no available BP")
				return
			}
			client := NewCaller()
			reqFN := &proto.FindNodeReq{
				NodeID: proto.NodeID(id.String()),
			}
			respFN := new(proto.FindNodeResp)

			// TODO(auxten) add some random here for bp selection
			for _, bp := range BPs {
				method := "DHT.FindNode"
				err = client.CallNode(bp, method, reqFN, respFN)
				if err != nil {
					log.Errorf("call %s %s failed: %s", bp, method, err)
					continue
				}
				break
			}
			if err == nil {
				route.SetNodeAddrCache(id, respFN.Node.Addr)
				addr = respFN.Node.Addr
			}
		}
	}
	return
}
