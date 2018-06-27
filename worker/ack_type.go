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

package worker

import (
	"bytes"
	"encoding/binary"
	"time"

	"gitlab.com/thunderdb/ThunderDB/crypto/asymmetric"
	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
	"gitlab.com/thunderdb/ThunderDB/proto"
	"gitlab.com/thunderdb/ThunderDB/utils"
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
	Signee     *asymmetric.PublicKey
	Signature  *asymmetric.Signature
}

// Ack defines a whole client ack request entity.
type Ack struct {
	Header SignedAckHeader
}

// Serialize structure to bytes.
func (h *AckHeader) Serialize() []byte {
	if h == nil {
		return []byte{'\000'}
	}

	buf := new(bytes.Buffer)

	buf.Write(h.Response.Serialize())
	binary.Write(buf, binary.LittleEndian, uint64(len(h.NodeID)))
	buf.WriteString(string(h.NodeID))
	binary.Write(buf, binary.LittleEndian, int64(h.Timestamp.UnixNano()))

	return buf.Bytes()
}

// Serialize structure to bytes.
func (sh *SignedAckHeader) Serialize() []byte {
	if sh == nil {
		return []byte{'\000'}
	}

	buf := new(bytes.Buffer)

	buf.Write(sh.AckHeader.Serialize())
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

// Sign the request.
func (sh *SignedAckHeader) Sign(signer *asymmetric.PrivateKey) (err error) {
	// check original header signature
	if err = sh.Response.Verify(); err != nil {
		return
	}

	// build hash
	buildHash(&sh.AckHeader, &sh.HeaderHash)

	// sign
	sh.Signature, err = signer.Sign(sh.HeaderHash[:])

	return
}

// Serialize structure to bytes.
func (a *Ack) Serialize() []byte {
	if a == nil {
		return []byte{'\000'}
	}

	return a.Header.Serialize()
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

// ResponseHeaderHash returns the deep shadowed Response HeaderHash field.
func (a *Ack) ResponseHeaderHash() hash.Hash {
	return a.Header.AckHeader.Response.HeaderHash
}

// SignedRequestHeader returns the deep shadowed Request reference.
func (a *Ack) SignedRequestHeader() *SignedRequestHeader {
	return &a.Header.AckHeader.Response.Request
}

// SignedResponseHeader returns the Response reference.
func (a *Ack) SignedResponseHeader() *SignedResponseHeader {
	return &a.Header.Response
}

// SignedAckHeader returns the Header reference.
func (a *Ack) SignedAckHeader() *SignedAckHeader {
	return &a.Header
}

// MarshalBinary implements BinaryMarshaler.
func (sh *SignedAckHeader) MarshalBinary() ([]byte, error) {
	buffer := bytes.NewBuffer(nil)

	if err := utils.WriteElements(buffer, binary.BigEndian,
		&sh.Response,
		&sh.NodeID,
		sh.Timestamp,
		&sh.HeaderHash,
		sh.Signee,
		sh.Signature,
	); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

// UnmarshalBinary implements BinaryUnmarshaler.
func (sh *SignedAckHeader) UnmarshalBinary(b []byte) error {
	reader := bytes.NewReader(b)
	return utils.ReadElements(reader, binary.BigEndian,
		&sh.Response,
		&sh.NodeID,
		&sh.Timestamp,
		&sh.HeaderHash,
		&sh.Signee,
		&sh.Signature,
	)
}
