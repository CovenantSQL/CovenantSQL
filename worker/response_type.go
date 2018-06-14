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

package worker

import (
	"bytes"
	"encoding/binary"
	"time"

	"github.com/hashicorp/go-msgpack/codec"
	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
	"gitlab.com/thunderdb/ThunderDB/crypto/signature"
	"gitlab.com/thunderdb/ThunderDB/proto"
)

// ResponseRow defines single row of query response.
type ResponseRow struct {
	// TODO(xq262144), currently use as-is type from golang sqlite3 driver
	Values []interface{}
}

// ResponsePayload defines column names and rows of query response.
type ResponsePayload struct {
	Columns   []string
	DeclTypes []string
	Rows      []ResponseRow
}

// ResponseHeader defines a query response header.
type ResponseHeader struct {
	Request           SignedRequestHeader
	NodeID            proto.NodeID // response node id
	Timestamp         time.Time    // time in UTC zone
	RowCount          uint64       // response row count of payload
	AffectedRowsCount uint64       // response affected rows count
	DataHash          hash.Hash    // hash of query response
}

// SignedResponseHeader defines a signed query response header.
type SignedResponseHeader struct {
	ResponseHeader
	HeaderHash hash.Hash
	Signee     *signature.PublicKey
	Signature  *signature.Signature
}

// Response defines a complete query response.
type Response struct {
	Header  SignedResponseHeader
	Payload ResponsePayload
}

// Serialize structure to bytes.
func (r *ResponseRow) Serialize() []byte {
	// FIXME(xq262144), currently use idiomatic serialization for hash generation
	buf := new(bytes.Buffer)
	hd := codec.MsgpackHandle{}
	enc := codec.NewEncoder(buf, &hd)
	err := enc.Encode(r)
	// FIXME(xq262144), panic of failure, should never happened
	if err != nil {
		panic(err)
	}
	return buf.Bytes()
}

// Serialize structure to bytes.
func (r *ResponsePayload) Serialize() []byte {
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
	buf := new(bytes.Buffer)

	buf.Write(h.Request.Serialize())
	binary.Write(buf, binary.LittleEndian, uint64(len(h.NodeID)))
	buf.WriteString(string(h.NodeID))
	binary.Write(buf, binary.LittleEndian, int64(h.Timestamp.UnixNano()))
	binary.Write(buf, binary.LittleEndian, h.RowCount)
	binary.Write(buf, binary.LittleEndian, h.AffectedRowsCount)
	buf.Write(h.DataHash[:])

	return buf.Bytes()
}

// Serialize structure to bytes.
func (sh *SignedResponseHeader) Serialize() []byte {
	buf := new(bytes.Buffer)

	buf.Write(sh.ResponseHeader.Serialize())
	buf.Write(sh.HeaderHash[:])
	buf.Write(sh.Signee.Serialize())
	buf.Write(sh.Signature.Serialize())

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
	if !sh.Signature.Verify(sh.HeaderHash[:], sh.Signee) {
		return ErrSignVerification
	}

	return nil
}
