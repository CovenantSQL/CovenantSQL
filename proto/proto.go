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

// Envelope is the protocol
type Envelope struct {
	Version string
	TTL     time.Duration
	Expire  time.Duration
}

// PingReq is Ping RPC request
type PingReq struct {
	Node NodeBytes
	Envelope
}

// PingResp is Ping RPC response, i.e. Pong
type PingResp struct {
	Msg string
	Envelope
}

// FindValueReq is FindValue RPC request
type FindValueReq struct {
	NodeID NodeID
	Count  int
	Envelope
}

// FindValueResp is FindValue RPC response
type FindValueResp struct {
	Nodes []NodeBytes
	Msg   string
	Envelope
}
