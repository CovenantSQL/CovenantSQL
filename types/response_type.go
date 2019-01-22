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
	"context"
	"time"

	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/crypto/verifier"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils/trace"
	"github.com/pkg/errors"
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
	Request      SignedRequestHeader `json:"r"`
	NodeID       proto.NodeID        `json:"id"` // response node id
	Timestamp    time.Time           `json:"t"`  // time in UTC zone
	RowCount     uint64              `json:"c"`  // response row count of payload
	LogOffset    uint64              `json:"o"`  // request log offset
	LastInsertID int64               `json:"l"`  // insert insert id
	AffectedRows int64               `json:"a"`  // affected rows
	PayloadHash  hash.Hash           `json:"dh"` // hash of query response payload
}

// SignedResponseHeader defines a signed query response header.
type SignedResponseHeader struct {
	ResponseHeader
	verifier.DefaultHashSignVerifierImpl
}

// Response defines a complete query response.
type Response struct {
	Header  SignedResponseHeader `json:"h"`
	Payload ResponsePayload      `json:"p"`
}

// Verify checks hash and signature in response header.
func (sh *SignedResponseHeader) Verify() (err error) {
	// verify original request header
	if err = sh.Request.Verify(); err != nil {
		return
	}

	return sh.DefaultHashSignVerifierImpl.Verify(&sh.ResponseHeader)
}

// Sign the request.
func (sh *SignedResponseHeader) Sign(signer *asymmetric.PrivateKey) (err error) {
	// make sure original header is signed
	if err = sh.Request.Verify(); err != nil {
		err = errors.Wrapf(err, "SignedResponseHeader %v", sh)
		return
	}

	return sh.DefaultHashSignVerifierImpl.Sign(&sh.ResponseHeader, signer)
}

// Verify checks hash and signature in whole response.
func (sh *Response) Verify() (err error) {
	_, task := trace.NewTask(context.Background(), "ResponseVerify")
	defer task.End()

	// verify data hash in header
	if err = verifyHash(&sh.Payload, &sh.Header.PayloadHash); err != nil {
		return
	}

	return sh.Header.Verify()
}

// Sign the request.
func (sh *Response) Sign(signer *asymmetric.PrivateKey) (err error) {
	_, task := trace.NewTask(context.Background(), "ResponseSign")
	defer task.End()

	// set rows count
	sh.Header.RowCount = uint64(len(sh.Payload.Rows))

	// build hash in header
	if err = buildHash(&sh.Payload, &sh.Header.PayloadHash); err != nil {
		return
	}

	// sign the request
	return sh.Header.Sign(signer)
}
