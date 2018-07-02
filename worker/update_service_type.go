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

	"gitlab.com/thunderdb/ThunderDB/crypto/asymmetric"
	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
	"gitlab.com/thunderdb/ThunderDB/proto"
)

// UpdateType defines service update type.
type UpdateType int32

const (
	// CreateDB indicates create database operation.
	CreateDB UpdateType = iota
	// UpdateDB indicates database peers update operation.
	UpdateDB
	// DropDB indicates drop database operation.
	DropDB
)

// UpdateServiceHeader defines service update header.
type UpdateServiceHeader struct {
	Op       UpdateType
	Instance ServiceInstance
}

// SignedUpdateServiceHeader defines signed service update header.
type SignedUpdateServiceHeader struct {
	UpdateServiceHeader
	HeaderHash hash.Hash
	Signee     *asymmetric.PublicKey
	Signature  *asymmetric.Signature
}

// UpdateService defines service update type.
type UpdateService struct {
	proto.Envelope
	Header SignedUpdateServiceHeader
}

// UpdateServiceResponse defines empty response entity.
type UpdateServiceResponse struct{}

// Serialize structure to bytes.
func (h *UpdateServiceHeader) Serialize() []byte {
	if h == nil {
		return []byte{'\000'}
	}

	buf := new(bytes.Buffer)

	binary.Write(buf, binary.LittleEndian, int32(h.Op))
	buf.Write(h.Instance.Serialize())

	return buf.Bytes()
}

// Serialize structure to bytes.
func (sh *SignedUpdateServiceHeader) Serialize() []byte {
	if sh == nil {
		return []byte{'\000'}
	}

	buf := new(bytes.Buffer)

	buf.Write(sh.UpdateServiceHeader.Serialize())
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

// Verify checks hash and signature in update service header.
func (sh *SignedUpdateServiceHeader) Verify() (err error) {
	// verify hash
	if err = verifyHash(&sh.UpdateServiceHeader, &sh.HeaderHash); err != nil {
		return
	}
	// verify sign
	if sh.Signee == nil || sh.Signature == nil || !sh.Signature.Verify(sh.HeaderHash[:], sh.Signee) {
		return ErrSignVerification
	}
	return
}

// Sign the request.
func (sh *SignedUpdateServiceHeader) Sign(signer *asymmetric.PrivateKey) (err error) {
	// build hash
	buildHash(&sh.UpdateServiceHeader, &sh.HeaderHash)

	// sign
	sh.Signature, err = signer.Sign(sh.HeaderHash[:])

	return
}

// Serialize structure to bytes.
func (s *UpdateService) Serialize() []byte {
	if s == nil {
		return []byte{'\000'}
	}

	return s.Header.Serialize()
}

// Verify checks hash and signature in update service.
func (r *UpdateService) Verify() error {
	return r.Header.Verify()
}

// Sign the request.
func (r *UpdateService) Sign(signer *asymmetric.PrivateKey) (err error) {
	// sign
	return r.Header.Sign(signer)
}
