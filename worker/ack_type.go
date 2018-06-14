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
	"gitlab.com/thunderdb/ThunderDB/proto"
)

// AckHeader defines client ack entity.
type AckHeader struct {
	Response  SignedResponseHeader
	NodeID    proto.NodeID // ack node id
	Timestamp time.Time    // time in UTC zone
}

// SignedAckHeader defines client signed ack entity.
type SignedAckHeader struct {
	AckHeader
	HeaderHash hash.Hash
	Signee     *signature.PublicKey
	Signature  *signature.Signature
}

// Ack defines a whole client ack request entity.
type Ack struct {
	Header SignedAckHeader
}

// Serialize structure to bytes.
func (h *AckHeader) Serialize() []byte {
	buf := new(bytes.Buffer)

	buf.Write(h.Response.Serialize())
	binary.Write(buf, binary.LittleEndian, uint64(len(h.NodeID)))
	buf.WriteString(string(h.NodeID))
	binary.Write(buf, binary.LittleEndian, int64(h.Timestamp.UnixNano()))

	return buf.Bytes()
}

// Serialize structure to bytes.
func (sh *SignedAckHeader) Serialize() []byte {
	buf := new(bytes.Buffer)

	buf.Write(sh.AckHeader.Serialize())
	buf.Write(sh.HeaderHash[:])
	buf.Write(sh.Signee.Serialize())
	buf.Write(sh.Signature.Serialize())

	return buf.Bytes()
}

// Verify checks hash and signature in ack header.
func (sh *SignedAckHeader) Verify() (err error) {
	// verify response
	if err = sh.Response.Verify(); err != nil {
		return
	}
	if err = verifyHash(&sh.AckHeader, &sh.HeaderHash); err != nil {
		return
	}
	// verify sign
	if !sh.Signature.Verify(sh.HeaderHash[:], sh.Signee) {
		return ErrSignVerification
	}
	return
}

// Serialize structure to bytes.
func (a *Ack) Serialize() []byte {
	return a.Header.Serialize()
}

// Verify checks hash and signature in ack.
func (a *Ack) Verify() error {
	return a.Header.Verify()
}
