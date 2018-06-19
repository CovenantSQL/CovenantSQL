/*
 * Copyright 2018 The ThunderDB Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the “License”);
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an “AS IS” BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package rpc

import (
	"net/rpc"

	"github.com/thunderdb/ThunderDB/proto"
)

// NodeAwareServerCodec wraps normal rpc.ServerCodec and inject node id during request process
type NodeAwareServerCodec struct {
	rpc.ServerCodec
	NodeID *proto.RawNodeID
}

// NewNodeAwareServerCodec returns new NodeAwareServerCodec with normal rpc.ServerCode and proto.RawNodeID
func NewNodeAwareServerCodec(codec rpc.ServerCodec, nodeID *proto.RawNodeID) *NodeAwareServerCodec {
	return &NodeAwareServerCodec{
		ServerCodec: codec,
		NodeID:      nodeID,
	}
}

// ReadRequestBody override default rpc.ServerCodec behaviour and inject remote node id into request
func (nc *NodeAwareServerCodec) ReadRequestBody(body interface{}) (err error) {
	err = nc.ServerCodec.ReadRequestBody(body)
	if err != nil {
		return
	}

	// test if request contains rpc envelope
	if body == nil {
		return
	}

	if _, ok := body.(proto.EnvelopeAPI); !ok {
		return
	}

	// inject node id to rpc envelope
	r := body.(proto.EnvelopeAPI)
	r.SetNodeID(nc.NodeID)

	return
}
