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

// ResponseRow defines single row of query response.
type ResponseRow struct {
	Values []interface{}
}

// ResponsePayload defines column names and rows of query response.
type ResponsePayload struct {
	Columns   []string      `json:"c"`
	DeclTypes []string      `json:"t"`
	Rows      []ResponseRow `json:"r"`
}

// ResponseHeader defines a query response header.
type ResponseHeader struct {
	Request      RequestHeader `json:"r"`
	RequestHash  hash.Hash     `json:"rh"`
	NodeID       proto.NodeID  `json:"id"` // response node id
	Timestamp    time.Time     `json:"t"`  // time in UTC zone
	RowCount     uint64        `json:"c"`  // response row count of payload
	LogOffset    uint64        `json:"o"`  // request log offset
	LastInsertID int64         `json:"l"`  // insert insert id
	AffectedRows int64         `json:"a"`  // affected rows
	PayloadHash  hash.Hash     `json:"dh"` // hash of query response payload
}

// GetRequestHash returns the request hash.
func (h *ResponseHeader) GetRequestHash() hash.Hash {
	return h.RequestHash
}

// GetRequestTimestamp returns the request timestamp.
func (h *ResponseHeader) GetRequestTimestamp() time.Time {
	return h.Request.Timestamp
}

// SignedResponseHeader defines a signed query response header.
type SignedResponseHeader struct {
	ResponseHeader
	verifier.DefaultHashSignVerifierImpl
}

// Response defines a complete query response.
type Response struct {
	Header   SignedResponseHeader `json:"h"`
	Payload  ResponsePayload      `json:"p"`
	callback func(res *Response)
}

// Verify checks hash and signature in response header.
func (sh *SignedResponseHeader) Verify() (err error) {
	return sh.DefaultHashSignVerifierImpl.Verify(&sh.ResponseHeader)
}

// Sign the request.
func (sh *SignedResponseHeader) Sign(signer *asymmetric.PrivateKey) (err error) {
	return sh.DefaultHashSignVerifierImpl.Sign(&sh.ResponseHeader, signer)
}

// Verify checks hash and signature in whole response.
func (r *Response) Verify() (err error) {
	// verify data hash in header
	if err = verifyHash(&r.Payload, &r.Header.PayloadHash); err != nil {
		return
	}

	return r.Header.Verify()
}

// Sign the response.
func (r *Response) Sign(signer *asymmetric.PrivateKey) (err error) {
	if err = r.BuildHash(); err != nil {
		return
	}

	return r.SignHash(signer)
}

// SignHash computes the signature of the response through existing hash.
func (r *Response) SignHash(signer *asymmetric.PrivateKey) (err error) {
	return r.Header.SignHash(signer)
}

// BuildHash computes the hash of the response.
func (r *Response) BuildHash() (err error) {
	// set rows count
	r.Header.RowCount = uint64(len(r.Payload.Rows))

	// build hash in header
	if err = buildHash(&r.Payload, &r.Header.PayloadHash); err != nil {
		return
	}

	// compute header hash
	return r.Header.SetHash(&r.Header.ResponseHeader)
}

// SetResponseCallback stores callback function to process after response processed.
func (r *Response) SetResponseCallback(cb func(res *Response)) {
	r.callback = cb
}

// TriggerResponseCallback async executes callback.
func (r *Response) TriggerResponseCallback() {
	if r.callback != nil {
		go r.callback(r)
	}
}
