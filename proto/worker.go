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

package proto

import "time"

// ServiceUpdateType database service update operation type
type ServiceUpdateType uint

// database service update operation type values
const (
	ServiceUpdateTypeCreate ServiceUpdateType = iota
	ServiceUpdateTypeUpdate
	ServiceUpdateTypeDrop
)

// QueryID query id type for database query
type QueryID string

// DatabaseID database id for identify globally unique database
type DatabaseID string

// LogOffset is database replication log offset
type LogOffset uint64

// PrepareQueryReq prepare query request entity
type PrepareQueryReq struct {
	Node       Node          // issuing client node info
	DatabaseID DatabaseID    // database id
	QueryID    QueryID       // query id
	Query      string        // query content
	Timeout    time.Duration // query expire time duration
}

// PrepareQueryResp prepare query response entity
type PrepareQueryResp struct {
	Success bool
	Msg     string // response status text
	Envelope
}

// CommitQueryReq execute query request entity
type CommitQueryReq struct {
	Node       Node
	DatabaseID DatabaseID
	QueryID    QueryID
	Envelope
}

// CommitQueryResp execute query response entity
type CommitQueryResp struct {
	Success bool
	Msg     string // response status text
	Envelope
}

// RollbackQueryReq confirm query request entity
type RollbackQueryReq struct {
	Node       Node
	DatabaseID DatabaseID
	QueryID    QueryID
	Envelope
}

// RollbackQueryResp confirm query response entity
type RollbackQueryResp struct {
	Success bool
	Msg     string // response status text
	Envelope
}

// PrepareServiceUpdateReq update service request entity
type PrepareServiceUpdateReq struct {
	Node               Node              // requiring block producer
	DatabaseID         DatabaseID        // database identifier
	ReservedSpace      uint64            // database reserved max space in bytes
	Peers              []Node            // database service nodes (if op == add/update, current node itself is included)
	OpType             ServiceUpdateType // service update operation type
	LeaderNode         Node              // leader node address
	IsLeader           bool              // service role in database quorum
	ReplicationTimeout time.Duration     // dynamic controlled replication timeout
	ApplyTimeout       time.Duration     // dynamic controlled commit log apply timeout
	LeaderLeaseTimeout time.Duration     // dynamic controlled leader lease timeout
	Envelope
}

// PrepareServiceUpdateResp update service response entity
type PrepareServiceUpdateResp struct {
	Success bool
	Msg     string // response status text
	Envelope
}

// CommitServiceUpdateReq update service request entity
type CommitServiceUpdateReq struct {
	Node       Node       // requiring block producer
	DatabaseID DatabaseID // database identifier
	Envelope
}

// CommitServiceUpdateResp update service response entity
type CommitServiceUpdateResp struct {
	Success bool
	Msg     string // response status text
	Envelope
}

// RollbackServiceUpdateReq update service request entity
type RollbackServiceUpdateReq struct {
	Node       Node       // requiring block producer
	DatabaseID DatabaseID // database identifier
	Envelope
}

// RollbackServiceUpdateResp update service response entity
type RollbackServiceUpdateResp struct {
	Success bool
	Msg     string // response status text
	Envelope
}

// GetStatusReq get raft status request entity
type GetStatusReq struct {
	Node       Node
	DatabaseID DatabaseID
	Envelope
}

// GetStatusResp get raft status response entity
type GetStatusResp struct {
	State string   // node state
	Peers []string // peers nodes
	Msg   string
	Envelope
}

// CheckpointReq get database checkpoint request entity
type CheckpointReq struct {
	Node       Node
	DatabaseID DatabaseID
}

// CheckpointResp get database checkpoint response entity
type CheckpointResp struct {
	Success    bool
	Msg        string    // response status text
	LogOffset  LogOffset // checkpoint log offset
	DataLength uint64    // checkpoint data length
	Data       []byte    // checkpoint data
	Envelope
}

// SubscribeReq commit log subscribe request entity
type SubscribeReq struct {
	Node       Node // make sure the Node is a peer or client
	DatabaseID DatabaseID
	LogOffset  LogOffset
	Envelope
}

// SubscribeResp commit log subscribe response entity
type SubscribeResp struct {
	Success bool
	Msg     string // response status text
	Envelope
}

// UpdateSubscriptionReq update subscription request entity
type UpdateSubscriptionReq struct {
	Node                 Node
	DatabaseID           DatabaseID
	ContinueSubscription bool      // should continue subscription
	LogOffset            LogOffset // reset log offset, 0 means no effect
	Envelope
}

// UpdateSubscriptionResp update subscription response entity
type UpdateSubscriptionResp struct {
	Success bool
	Msg     string // response status text
	Envelope
}
