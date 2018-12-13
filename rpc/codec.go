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

package rpc

import (
	"context"
	"net/rpc"

	"github.com/CovenantSQL/CovenantSQL/proto"
)

// NodeAwareServerCodec wraps normal rpc.ServerCodec and inject node id during request process
type NodeAwareServerCodec struct {
	rpc.ServerCodec
	NodeID *proto.RawNodeID
	Ctx    context.Context
}

// NewNodeAwareServerCodec returns new NodeAwareServerCodec with normal rpc.ServerCode and proto.RawNodeID
func NewNodeAwareServerCodec(ctx context.Context, codec rpc.ServerCodec, nodeID *proto.RawNodeID) *NodeAwareServerCodec {
	return &NodeAwareServerCodec{
		ServerCodec: codec,
		NodeID:      nodeID,
		Ctx:         ctx,
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

	if r, ok := body.(proto.EnvelopeAPI); ok {
		// inject node id to rpc envelope
		r.SetNodeID(nc.NodeID)
		// inject context
		r.SetContext(nc.Ctx)
	}

	return
}
