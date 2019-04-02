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
	"context"
	"io"
	"net/rpc"

	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils"
)

// ServeDirect serves data stream directly.
func ServeDirect(
	ctx context.Context, server *rpc.Server, stream io.ReadWriteCloser, remote *proto.RawNodeID,
) {
	subctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()
	nodeAwareCodec := NewNodeAwareServerCodec(subctx, utils.GetMsgPackServerCodec(stream), remote)
	server.ServeCodec(nodeAwareCodec)
}
