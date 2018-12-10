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

import "github.com/CovenantSQL/CovenantSQL/proto"

// ApplyRequest defines the kayak apply RPC request entity.
type ApplyRequest struct {
	proto.Envelope
	Instance string
	Log      *Log
}

// State defines the kayak runtime state attributes.
type State struct {
	Instance      string           // instance id
	IsRecovering  bool             // the runtime recovery state
	Role          proto.ServerRole // the runtime application role
	LastTruncated uint64           // the last truncated wal index
	LastCommitted uint64           // the last commit type log index
	LastOffset    uint64           // the last log entry index (log end offset, not high watermark)
}

// FetchRequest defines the kayak log fetch RPC request entity.
type FetchRequest struct {
	proto.Envelope
	Instance string
	Index    uint64
}

// FetchResponse defines the kayak log fetch RPC response entity.
type FetchResponse struct {
	proto.Envelope
	Instance string
	Log      *Log
}
