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
	"github.com/CovenantSQL/CovenantSQL/proto"
)

// LogType defines the log type.
type LogType uint16

const (
	// LogPrepare defines the prepare phase of a commit.
	LogPrepare LogType = iota
	// LogRollback defines the rollback phase of a commit.
	LogRollback
	// LogCommit defines the commit phase of a commit.
	LogCommit
	// LogCheckpoint defines the checkpoint log (created/virtually created by block production or log truncation).
	LogCheckpoint
	// LogBarrier defines barrier log, all open windows should be waiting this operations to complete.
	LogBarrier
	// LogNoop defines noop log.
	LogNoop
)

func (t LogType) String() (s string) {
	switch t {
	case LogPrepare:
		return "LogPrepare"
	case LogRollback:
		return "LogRollback"
	case LogCommit:
		return "LogCommit"
	case LogCheckpoint:
		return "LogCheckpoint"
	case LogBarrier:
		return "LogBarrier"
	case LogNoop:
		return "LogNoop"
	default:
		return "Unknown"
	}
}

// LogHeader defines the checksum header structure.
type LogHeader struct {
	Index      uint64       // log index
	Version    uint64       // log version
	Type       LogType      // log type
	Producer   proto.NodeID // producer node
	DataLength uint64       // data length
}

// Log defines the log data structure.
type Log struct {
	LogHeader
	// Data could be detected and handle decode properly by log layer
	Data []byte
}
