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
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/proto"
)

//go:generate hsp

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
	Hash      hash.Hash
	Signee    *asymmetric.PublicKey
	Signature *asymmetric.Signature
}

// UpdateService defines service update type.
type UpdateService struct {
	proto.Envelope
	Header SignedUpdateServiceHeader
}

// UpdateServiceResponse defines empty response entity.
type UpdateServiceResponse struct{}

// Verify checks hash and signature in update service header.
func (sh *SignedUpdateServiceHeader) Verify() (err error) {
	// verify hash
	if err = verifyHash(&sh.UpdateServiceHeader, &sh.Hash); err != nil {
		return
	}
	// verify sign
	if sh.Signee == nil || sh.Signature == nil || !sh.Signature.Verify(sh.Hash[:], sh.Signee) {
		return ErrSignVerification
	}
	return
}

// Sign the request.
func (sh *SignedUpdateServiceHeader) Sign(signer *asymmetric.PrivateKey) (err error) {
	// build hash
	if err = buildHash(&sh.UpdateServiceHeader, &sh.Hash); err != nil {
		return
	}

	// sign
	sh.Signature, err = signer.Sign(sh.Hash[:])
	sh.Signee = signer.PubKey()

	return
}

// Verify checks hash and signature in update service.
func (s *UpdateService) Verify() error {
	return s.Header.Verify()
}

// Sign the request.
func (s *UpdateService) Sign(signer *asymmetric.PrivateKey) (err error) {
	// sign
	return s.Header.Sign(signer)
}
