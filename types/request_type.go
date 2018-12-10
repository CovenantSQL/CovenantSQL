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
	"fmt"
	"time"

	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/crypto/verifier"
	"github.com/CovenantSQL/CovenantSQL/proto"
)

//go:generate hsp

// QueryType enumerates available query type, currently read/write.
type QueryType int32

const (
	// ReadQuery defines a read query type.
	ReadQuery QueryType = iota
	// WriteQuery defines a write query type.
	WriteQuery
)

// NamedArg defines the named argument structure for database.
type NamedArg struct {
	Name  string
	Value interface{}
}

// Query defines single query.
type Query struct {
	Pattern string
	Args    []NamedArg
}

// RequestPayload defines a queries payload.
type RequestPayload struct {
	Queries []Query `json:"qs"`
}

// RequestHeader defines a query request header.
type RequestHeader struct {
	QueryType    QueryType        `json:"qt"`
	NodeID       proto.NodeID     `json:"id"`   // request node id
	DatabaseID   proto.DatabaseID `json:"dbid"` // request database id
	ConnectionID uint64           `json:"cid"`
	SeqNo        uint64           `json:"seq"`
	Timestamp    time.Time        `json:"t"`  // time in UTC zone
	BatchCount   uint64           `json:"bc"` // query count in this request
	QueriesHash  hash.Hash        `json:"qh"` // hash of query payload
	LogOffset    uint64           `hspack:"-"`
}

// QueryKey defines an unique query key of a request.
type QueryKey struct {
	NodeID       proto.NodeID `json:"id"`
	ConnectionID uint64       `json:"cid"`
	SeqNo        uint64       `json:"seq"`
}

// String implements fmt.Stringer for logging purpose.
func (k *QueryKey) String() string {
	return fmt.Sprintf("%s#%016x#%016x", string(k.NodeID[:8]), k.ConnectionID, k.SeqNo)
}

// SignedRequestHeader defines a signed query request header.
type SignedRequestHeader struct {
	RequestHeader
	verifier.DefaultHashSignVerifierImpl
}

// Request defines a complete query request.
type Request struct {
	proto.Envelope
	Header  SignedRequestHeader `json:"h"`
	Payload RequestPayload      `json:"p"`
}

// String implements fmt.Stringer for logging purpose.
func (t QueryType) String() string {
	switch t {
	case ReadQuery:
		return "read"
	case WriteQuery:
		return "write"
	default:
		return "unknown"
	}
}

// Verify checks hash and signature in request header.
func (sh *SignedRequestHeader) Verify() (err error) {
	return sh.DefaultHashSignVerifierImpl.Verify(&sh.RequestHeader)
}

// Sign the request.
func (sh *SignedRequestHeader) Sign(signer *asymmetric.PrivateKey) (err error) {
	return sh.DefaultHashSignVerifierImpl.Sign(&sh.RequestHeader, signer)
}

// Verify checks hash and signature in whole request.
func (r *Request) Verify() (err error) {
	// verify payload hash in signed header
	if err = verifyHash(&r.Payload, &r.Header.QueriesHash); err != nil {
		return
	}
	// verify header sign
	return r.Header.Verify()
}

// Sign the request.
func (r *Request) Sign(signer *asymmetric.PrivateKey) (err error) {
	// set query count
	r.Header.BatchCount = uint64(len(r.Payload.Queries))

	// compute payload hash
	if err = buildHash(&r.Payload, &r.Header.QueriesHash); err != nil {
		return
	}

	return r.Header.Sign(signer)
}

// GetQueryKey returns a unique query key of this request.
func (sh *SignedRequestHeader) GetQueryKey() QueryKey {
	return QueryKey{
		NodeID:       sh.NodeID,
		ConnectionID: sh.ConnectionID,
		SeqNo:        sh.SeqNo,
	}
}
