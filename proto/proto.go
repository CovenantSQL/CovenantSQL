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

// Package proto contains DHT RPC protocol struct
package proto

import (
	"time"
)

// EnvelopeAPI defines envelope access functions for rpc Request/Response
type EnvelopeAPI interface {
	GetVersion() string
	GetTTL() time.Duration
	GetExpire() time.Duration
	GetNodeID() *RawNodeID

	SetVersion(string)
	SetTTL(time.Duration)
	SetExpire(time.Duration)
	SetNodeID(*RawNodeID)
}

// Envelope is the protocol header
type Envelope struct {
	Version string
	TTL     time.Duration
	Expire  time.Duration
	NodeID  *RawNodeID
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
	NodeID NodeID
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
	NodeID NodeID
	Count  int
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
	NodeID NodeID
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

// DatabaseID is database name, will be generated from UUID
type DatabaseID string
