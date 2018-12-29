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

// Package proto contains DHT RPC protocol struct
package proto

import (
	"context"
	"fmt"
	"time"

	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/pkg/errors"
)

//go:generate hsp
//hsp:shim time.Duration as:int64 using:int64/int64 mode:cast

// EnvelopeAPI defines envelope access functions for rpc Request/Response
type EnvelopeAPI interface {
	GetVersion() string
	GetTTL() time.Duration
	GetExpire() time.Duration
	GetNodeID() *RawNodeID
	GetContext() context.Context

	SetVersion(string)
	SetTTL(time.Duration)
	SetExpire(time.Duration)
	SetNodeID(*RawNodeID)
	SetContext(context.Context)
}

// Envelope is the protocol header
type Envelope struct {
	Version string        `json:"v"`
	TTL     time.Duration `json:"t"`
	Expire  time.Duration `json:"e"`
	NodeID  *RawNodeID    `json:"id"`
	_ctx    context.Context
}

// PingReq is Ping RPC request
type PingReq struct {
	Node Node
	Envelope
}

// PingResp is Ping RPC response, i.e. Pong
type PingResp struct {
	Msg string
	Envelope
}

// UploadMetricsReq is UploadMetrics RPC request
type UploadMetricsReq struct {
	// MetricFamily Bytes array
	MFBytes [][]byte
	Envelope
}

// UploadMetricsResp is UploadMetrics RPC response
type UploadMetricsResp struct {
	Msg string
	Envelope
}

// FindNeighborReq is FindNeighbor RPC request
type FindNeighborReq struct {
	ID    NodeID
	Roles []ServerRole
	Count int
	Envelope
}

// FindNeighborResp is FindNeighbor RPC response
type FindNeighborResp struct {
	Nodes []Node
	Msg   string
	Envelope
}

// FindNodeReq is FindNode RPC request
type FindNodeReq struct {
	ID NodeID
	Envelope
}

// FindNodeResp is FindNode RPC response
type FindNodeResp struct {
	Node *Node
	Msg  string
	Envelope
}

// Following are envelope methods implementing EnvelopeAPI interface

// GetVersion implements EnvelopeAPI.GetVersion
func (e *Envelope) GetVersion() string {
	return e.Version
}

// GetTTL implements EnvelopeAPI.GetTTL
func (e *Envelope) GetTTL() time.Duration {
	return e.TTL
}

// GetExpire implements EnvelopeAPI.GetExpire
func (e *Envelope) GetExpire() time.Duration {
	return e.Expire
}

// GetNodeID implements EnvelopeAPI.GetNodeID
func (e *Envelope) GetNodeID() *RawNodeID {
	return e.NodeID
}

// GetContext returns context from envelop which is set in server Accept
func (e *Envelope) GetContext() context.Context {
	if e._ctx == nil {
		return context.Background()
	}
	return e._ctx
}

// SetVersion implements EnvelopeAPI.SetVersion
func (e *Envelope) SetVersion(ver string) {
	e.Version = ver
}

// SetTTL implements EnvelopeAPI.SetTTL
func (e *Envelope) SetTTL(ttl time.Duration) {
	e.TTL = ttl
}

// SetExpire implements EnvelopeAPI.SetExpire
func (e *Envelope) SetExpire(exp time.Duration) {
	e.Expire = exp
}

// SetNodeID implements EnvelopeAPI.SetNodeID
func (e *Envelope) SetNodeID(nodeID *RawNodeID) {
	e.NodeID = nodeID
}

// SetContext set a ctx in envelope
func (e *Envelope) SetContext(ctx context.Context) {
	e._ctx = ctx
}

// DatabaseID is database name, will be generated from UUID
type DatabaseID string

// AccountAddress converts DatabaseID to AccountAddress.
func (d *DatabaseID) AccountAddress() (a AccountAddress, err error) {
	h, err := hash.NewHashFromStr(string(*d))
	if err != nil {
		err = errors.Wrap(err, "fail to convert string to hash")
		return
	}
	a = AccountAddress(*h)
	return
}

// FromAccountAndNonce generates databaseID from Account and its nonce.
func FromAccountAndNonce(accountAddress AccountAddress, nonce uint32) DatabaseID {
	addrAndNonce := fmt.Sprintf("%s%d", accountAddress.String(), nonce)
	rawID := hash.THashH([]byte(addrAndNonce))
	return DatabaseID(rawID.String())
}
