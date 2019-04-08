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
	"github.com/CovenantSQL/CovenantSQL/naconn"
	"github.com/CovenantSQL/CovenantSQL/proto"
)

// The following variables define a method set to Dial/Accept node-oriented connections for
// this RPC package.
//
// TODO(leventeliu): allow to config other node-oriented connection dialer/accepter.
var (
	Dial   = naconn.Dial
	DialEx = naconn.DialEx
	Accept = naconn.Accept
)

// NOClientPool defines the node-oriented client pool interface.
type NOClientPool interface {
	Get(remote proto.NodeID) (Client, error)
	GetEx(remote proto.NodeID, isAnonymous bool) (Client, error)
	Close() error
}

// DialToNodeWithPool ties use connection in pool, if fails then connects to the node with nodeID.
func DialToNodeWithPool(pool NOClientPool, nodeID proto.NodeID, isAnonymous bool) (Client, error) {
	if isAnonymous {
		return pool.GetEx(nodeID, true)
	}
	//log.WithField("poolSize", pool.Len()).Debug("session pool size")
	return pool.Get(nodeID)
}
