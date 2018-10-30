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
	"bytes"
	"encoding/binary"
	"time"

	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils"
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
	DataHash     hash.Hash           `json:"dh"` // hash of query response
}

// SignedResponseHeader defines a signed query response header.
type SignedResponseHeader struct {
	ResponseHeader
	HeaderHash hash.Hash             `json:"h"`
	Signee     *asymmetric.PublicKey `json:"e"`
	Signature  *asymmetric.Signature `json:"s"`
}

// Response defines a complete query response.
type Response struct {
	Header  SignedResponseHeader `json:"h"`
	Payload ResponsePayload      `json:"p"`
}

// Serialize structure to bytes.
func (r *ResponseRow) Serialize() []byte {
	// HACK(xq262144), currently use idiomatic serialization for hash generation
	buf, _ := utils.EncodeMsgPack(r)

	return buf.Bytes()
}

// Serialize structure to bytes.
func (r *ResponsePayload) Serialize() []byte {
	if r == nil {
		return []byte{'\000'}
	}

	buf := new(bytes.Buffer)

	binary.Write(buf, binary.LittleEndian, uint64(len(r.Columns)))
	for _, c := range r.Columns {
		buf.WriteString(c)
	}

	binary.Write(buf, binary.LittleEndian, uint64(len(r.DeclTypes)))
	for _, t := range r.DeclTypes {
		buf.WriteString(t)
	}

	binary.Write(buf, binary.LittleEndian, uint64(len(r.Rows)))
	for _, row := range r.Rows {
		buf.Write(row.Serialize())
	}

	return buf.Bytes()
}

// Serialize structure to bytes.
func (h *ResponseHeader) Serialize() []byte {
	if h == nil {
		return []byte{'\000'}
	}

	buf := new(bytes.Buffer)

	buf.Write(h.Request.Serialize())
	binary.Write(buf, binary.LittleEndian, uint64(len(h.NodeID)))
	buf.WriteString(string(h.NodeID))
	binary.Write(buf, binary.LittleEndian, int64(h.Timestamp.UnixNano()))
	binary.Write(buf, binary.LittleEndian, h.RowCount)
	binary.Write(buf, binary.LittleEndian, h.LogOffset)
	buf.Write(h.DataHash[:])

	return buf.Bytes()
}

// Serialize structure to bytes.
func (sh *SignedResponseHeader) Serialize() []byte {
	if sh == nil {
		return []byte{'\000'}
	}

	buf := new(bytes.Buffer)

	buf.Write(sh.ResponseHeader.Serialize())
	buf.Write(sh.HeaderHash[:])
	if sh.Signee != nil {
		buf.Write(sh.Signee.Serialize())
	} else {
		buf.WriteRune('\000')
	}
	if sh.Signature != nil {
		buf.Write(sh.Signature.Serialize())
	} else {
		buf.WriteRune('\000')
	}

	return buf.Bytes()
}

// Verify checks hash and signature in response header.
func (sh *SignedResponseHeader) Verify() (err error) {
	// verify original request header
	if err = sh.Request.Verify(); err != nil {
		return
	}
	// verify hash
	if err = verifyHash(&sh.ResponseHeader, &sh.HeaderHash); err != nil {
		return
	}
	// verify signature
	if sh.Signee == nil || sh.Signature == nil || !sh.Signature.Verify(sh.HeaderHash[:], sh.Signee) {
		return ErrSignVerification
	}

	return nil
}

// Sign the request.
func (sh *SignedResponseHeader) Sign(signer *asymmetric.PrivateKey) (err error) {
	// make sure original header is signed
	if err = sh.Request.Verify(); err != nil {
		err = errors.Wrapf(err, "SignedResponseHeader %v", sh)
		return
	}

	// build our hash
	buildHash(&sh.ResponseHeader, &sh.HeaderHash)

	// sign
	sh.Signature, err = signer.Sign(sh.HeaderHash[:])
	sh.Signee = signer.PubKey()

	return
}

// Serialize structure to bytes.
func (sh *Response) Serialize() []byte {
	if sh == nil {
		return []byte{'\000'}
	}

	buf := new(bytes.Buffer)

	buf.Write(sh.Header.Serialize())
	buf.Write(sh.Payload.Serialize())

	return buf.Bytes()
}

// Verify checks hash and signature in whole response.
func (sh *Response) Verify() (err error) {
	// verify data hash in header
	if err = verifyHash(&sh.Payload, &sh.Header.DataHash); err != nil {
		return
	}

	return sh.Header.Verify()
}

// Sign the request.
func (sh *Response) Sign(signer *asymmetric.PrivateKey) (err error) {
	// set rows count
	sh.Header.RowCount = uint64(len(sh.Payload.Rows))

	// build hash in header
	buildHash(&sh.Payload, &sh.Header.DataHash)

	// sign the request
	return sh.Header.Sign(signer)
}
