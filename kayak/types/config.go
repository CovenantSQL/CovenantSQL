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

package types

import (
	"time"

	"github.com/CovenantSQL/CovenantSQL/proto"
)

// RuntimeConfig defines the runtime config of kayak.
type RuntimeConfig struct {
	// underlying handler.
	Handler Handler
	// minimum rpc success node percent requirement for prepare operation.
	PrepareThreshold float64
	// minimum rpc success node percent requirement for commit operation.
	CommitThreshold float64
	// maximum allowed time for prepare operation.
	PrepareTimeout time.Duration
	// maximum allowed time for commit operation.
	CommitTimeout time.Duration
	// timeout for follower to actively fetch logs from leader or maybe other nodes.
	LogFetchTimeout time.Duration
	// init peers of node.
	Peers *proto.Peers
	// wal for kayak.
	Wal Wal
	// current node id.
	NodeID proto.NodeID
	// current instance id.
	InstanceID string
	// mux service name.
	ServiceName string
	// mux service method.
	MethodName string
}
