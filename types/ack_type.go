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

	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/crypto/verifier"
	"github.com/CovenantSQL/CovenantSQL/proto"
)

//go:generate hsp

// AckHeader defines client ack entity.
type AckHeader struct {
	Response     ResponseHeader `json:"r"`
	ResponseHash hash.Hash      `json:"rh"`
	NodeID       proto.NodeID   `json:"i"` // ack node id
	Timestamp    time.Time      `json:"t"` // time in UTC zone
}

// GetQueryKey returns the request query key.
func (h *AckHeader) GetQueryKey() QueryKey {
	return h.Response.Request.GetQueryKey()
}

// GetRequestTimestamp returns the request timestamp.
func (h *AckHeader) GetRequestTimestamp() time.Time {
	return h.Response.GetRequestTimestamp()
}

// GetResponseTimestamp returns the response timestamp.
func (h *AckHeader) GetResponseTimestamp() time.Time {
	return h.Response.Timestamp
}

// GetRequestHash returns the request hash.
func (h *AckHeader) GetRequestHash() hash.Hash {
	return h.Response.GetRequestHash()
}

// GetResponseHash returns the response hash.
func (h *AckHeader) GetResponseHash() hash.Hash {
	return h.ResponseHash
}

// SignedAckHeader defines client signed ack entity.
type SignedAckHeader struct {
	AckHeader
	verifier.DefaultHashSignVerifierImpl
}

// Ack defines a whole client ack request entity.
type Ack struct {
	proto.Envelope
	Header SignedAckHeader `json:"h"`
}

// AckResponse defines client ack response entity.
type AckResponse struct{}

// Verify checks hash and signature in ack header.
func (sh *SignedAckHeader) Verify() (err error) {
	return sh.DefaultHashSignVerifierImpl.Verify(&sh.AckHeader)
}

// Sign the request.
func (sh *SignedAckHeader) Sign(signer *asymmetric.PrivateKey) (err error) {
	return sh.DefaultHashSignVerifierImpl.Sign(&sh.AckHeader, signer)
}

// Verify checks hash and signature in ack.
func (a *Ack) Verify() error {
	return a.Header.Verify()
}

// Sign the request.
func (a *Ack) Sign(signer *asymmetric.PrivateKey) (err error) {
	// sign
	return a.Header.Sign(signer)
}
