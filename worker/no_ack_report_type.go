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

	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
	"gitlab.com/thunderdb/ThunderDB/crypto/signature"
	"gitlab.com/thunderdb/ThunderDB/kayak"
	"gitlab.com/thunderdb/ThunderDB/proto"
)

// NoAckReportHeader defines worker issued client no ack report.
type NoAckReportHeader struct {
	NodeID    proto.NodeID // reporter node id
	Timestamp time.Time    // time in UTC zone
	Request   SignedRequestHeader
	Response  SignedResponseHeader
}

// SignedNoAckReportHeader defines worker worker issued/signed client no ack report.
type SignedNoAckReportHeader struct {
	NoAckReportHeader
	HeaderHash hash.Hash
	Signee     *signature.PublicKey
	Signature  *signature.Signature
}

// NoAckReport defines whole worker no client ack report.
type NoAckReport struct {
	Header SignedNoAckReportHeader
}

// AggrNoAckReportHeader defines worker leader aggregated client no ack report.
type AggrNoAckReportHeader struct {
	NodeID    proto.NodeID              // aggregated report node id
	Timestamp time.Time                 // time in UTC zone
	Request   SignedRequestHeader       // original request
	Reports   []SignedNoAckReportHeader // no-ack reports
	Peers     kayak.Peers               // serving peers during report
}

// SignedAggrNoAckReportHeader defines worker leader aggregated/signed client no ack report.
type SignedAggrNoAckReportHeader struct {
	AggrNoAckReportHeader
	HeaderHash hash.Hash
	Signee     *signature.PublicKey
	Signature  *signature.Signature
}

// AggrNoAckReport defines whole worker leader no client ack report.
type AggrNoAckReport struct {
	Header SignedAggrNoAckReportHeader
}

// Serialize structure to bytes.
func (h *NoAckReportHeader) Serialize() []byte {
	buf := new(bytes.Buffer)

	binary.Write(buf, binary.LittleEndian, uint64(len(h.NodeID)))
	buf.WriteString(string(h.NodeID))
	binary.Write(buf, binary.LittleEndian, int64(h.Timestamp.UnixNano()))
	buf.Write(h.Request.Serialize())
	buf.Write(h.Response.Serialize())

	return buf.Bytes()
}

// Serialize structure to bytes.
func (sh *SignedNoAckReportHeader) Serialize() []byte {
	buf := new(bytes.Buffer)

	buf.Write(sh.NoAckReportHeader.Serialize())
	buf.Write(sh.HeaderHash[:])
	buf.Write(sh.Signee.Serialize())
	buf.Write(sh.Signature.Serialize())

	return buf.Bytes()
}

// Verify checks hash and signature in signed no ack report header.
func (sh *SignedNoAckReportHeader) Verify() (err error) {
	// verify original request
	if err = sh.Request.Verify(); err != nil {
		return
	}
	// verify original response
	if err = sh.Response.Verify(); err != nil {
		return
	}
	// verify hash
	if err = verifyHash(&sh.NoAckReportHeader, &sh.HeaderHash); err != nil {
		return
	}
	// validate signature
	if !sh.Signature.Verify(sh.HeaderHash[:], sh.Signee) {
		return ErrSignVerification
	}
	return
}

// Serialize structure to bytes.
func (r *NoAckReport) Serialize() []byte {
	return r.Serialize()
}

// Verify checks hash and signature in whole no ack report.
func (r *NoAckReport) Verify() error {
	return r.Header.Verify()
}

// Serialize structure to bytes.
func (h *AggrNoAckReportHeader) Serialize() []byte {
	buf := new(bytes.Buffer)

	binary.Write(buf, binary.LittleEndian, uint64(len(h.NodeID)))
	buf.WriteString(string(h.NodeID))
	binary.Write(buf, binary.LittleEndian, int64(h.Timestamp.UnixNano()))
	buf.Write(h.Request.Serialize())
	binary.Write(buf, binary.LittleEndian, uint64(len(h.Reports)))
	for _, r := range h.Reports {
		buf.Write(r.Serialize())
	}
	buf.Write(h.Peers.Serialize())

	return buf.Bytes()
}

// Serialize structure to bytes.
func (sh *SignedAggrNoAckReportHeader) Serialize() []byte {
	buf := new(bytes.Buffer)

	buf.Write(sh.AggrNoAckReportHeader.Serialize())
	buf.Write(sh.HeaderHash[:])
	buf.Write(sh.Signee.Serialize())
	buf.Write(sh.Signature.Serialize())

	return buf.Bytes()
}

// Verify checks hash and signature in aggregated no ack report.
func (sh *SignedAggrNoAckReportHeader) Verify() (err error) {
	// verify original request
	if err = sh.AggrNoAckReportHeader.Request.Verify(); err != nil {
		return
	}
	// verify original reports
	for _, r := range sh.Reports {
		if err = r.Verify(); err != nil {
			return
		}
	}
	// verify hash
	if err = verifyHash(&sh.AggrNoAckReportHeader, &sh.HeaderHash); err != nil {
		return
	}
	// verify signature
	if !sh.Signature.Verify(sh.HeaderHash[:], sh.Signee) {
		return ErrSignVerification
	}
	return
}

// Serialize structure to bytes.
func (r *AggrNoAckReport) Serialize() ([]byte, error) {
	return r.Serialize()
}

// Verify the whole aggregation no ack report.
func (r *AggrNoAckReport) Verify() (err error) {
	return r.Header.Verify()
}
