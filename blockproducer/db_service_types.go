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

package blockproducer

import (
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/proto"
	wt "github.com/CovenantSQL/CovenantSQL/worker/types"
)

//go:generate hsp

// CreateDatabaseRequestHeader defines client create database rpc header.
type CreateDatabaseRequestHeader struct {
	ResourceMeta wt.ResourceMeta
}

// SignedCreateDatabaseRequestHeader defines signed client create database request header.
type SignedCreateDatabaseRequestHeader struct {
	CreateDatabaseRequestHeader
	Hash      hash.Hash
	Signee    *asymmetric.PublicKey
	Signature *asymmetric.Signature
}

// Verify checks hash and signature in create database request header.
func (sh *SignedCreateDatabaseRequestHeader) Verify() (err error) {
	// verify hash
	if err = verifyHash(&sh.CreateDatabaseRequestHeader, &sh.Hash); err != nil {
		return
	}
	// verify sign
	if sh.Signee == nil || sh.Signature == nil || !sh.Signature.Verify(sh.Hash[:], sh.Signee) {
		return wt.ErrSignVerification
	}
	return
}

// Sign the request.
func (sh *SignedCreateDatabaseRequestHeader) Sign(signer *asymmetric.PrivateKey) (err error) {
	// build hash
	if err = buildHash(&sh.CreateDatabaseRequestHeader, &sh.Hash); err != nil {
		return
	}

	// sign
	sh.Signature, err = signer.Sign(sh.Hash[:])
	sh.Signee = signer.PubKey()

	return
}

// CreateDatabaseRequest defines client create database rpc request entity.
type CreateDatabaseRequest struct {
	proto.Envelope
	Header SignedCreateDatabaseRequestHeader
}

// Verify checks hash and signature in request header.
func (r *CreateDatabaseRequest) Verify() error {
	return r.Header.Verify()
}

// Sign the request.
func (r *CreateDatabaseRequest) Sign(signer *asymmetric.PrivateKey) (err error) {
	// sign
	return r.Header.Sign(signer)
}

// CreateDatabaseResponseHeader defines client create database rpc response header.
type CreateDatabaseResponseHeader struct {
	InstanceMeta wt.ServiceInstance
}

// SignedCreateDatabaseResponseHeader defines signed client create database response header.
type SignedCreateDatabaseResponseHeader struct {
	CreateDatabaseResponseHeader
	Hash      hash.Hash
	Signee    *asymmetric.PublicKey
	Signature *asymmetric.Signature
}

// Verify checks hash and signature in create database response header.
func (sh *SignedCreateDatabaseResponseHeader) Verify() (err error) {
	// verify hash
	if err = verifyHash(&sh.CreateDatabaseResponseHeader, &sh.Hash); err != nil {
		return
	}
	// verify sign
	if sh.Signee == nil || sh.Signature == nil || !sh.Signature.Verify(sh.Hash[:], sh.Signee) {
		return wt.ErrSignVerification
	}
	return
}

// Sign the response.
func (sh *SignedCreateDatabaseResponseHeader) Sign(signer *asymmetric.PrivateKey) (err error) {
	// build hash
	if err = buildHash(&sh.CreateDatabaseResponseHeader, &sh.Hash); err != nil {
		return
	}

	// sign
	sh.Signature, err = signer.Sign(sh.Hash[:])
	sh.Signee = signer.PubKey()

	return
}

// CreateDatabaseResponse defines client create database rpc response entity.
type CreateDatabaseResponse struct {
	proto.Envelope
	Header SignedCreateDatabaseResponseHeader
}

// Verify checks hash and signature in response header.
func (r *CreateDatabaseResponse) Verify() error {
	return r.Header.Verify()
}

// Sign the response.
func (r *CreateDatabaseResponse) Sign(signer *asymmetric.PrivateKey) (err error) {
	// sign
	return r.Header.Sign(signer)
}

// DropDatabaseRequestHeader defines client drop database rpc request header.
type DropDatabaseRequestHeader struct {
	DatabaseID proto.DatabaseID
}

// SignedDropDatabaseRequestHeader defines signed client drop database rpc request header.
type SignedDropDatabaseRequestHeader struct {
	DropDatabaseRequestHeader
	Hash      hash.Hash
	Signee    *asymmetric.PublicKey
	Signature *asymmetric.Signature
}

// Verify checks hash and signature in request header.
func (sh *SignedDropDatabaseRequestHeader) Verify() (err error) {
	// verify hash
	if err = verifyHash(&sh.DropDatabaseRequestHeader, &sh.Hash); err != nil {
		return
	}
	// verify sign
	if sh.Signee == nil || sh.Signature == nil || !sh.Signature.Verify(sh.Hash[:], sh.Signee) {
		return wt.ErrSignVerification
	}
	return
}

// Sign the request.
func (sh *SignedDropDatabaseRequestHeader) Sign(signer *asymmetric.PrivateKey) (err error) {
	// build hash
	if err = buildHash(&sh.DropDatabaseRequestHeader, &sh.Hash); err != nil {
		return
	}

	// sign
	sh.Signature, err = signer.Sign(sh.Hash[:])
	sh.Signee = signer.PubKey()

	return
}

// DropDatabaseRequest defines client drop database rpc request entity.
type DropDatabaseRequest struct {
	proto.Envelope
	Header SignedDropDatabaseRequestHeader
}

// Verify checks hash and signature in request header.
func (r *DropDatabaseRequest) Verify() error {
	return r.Header.Verify()
}

// Sign the request.
func (r *DropDatabaseRequest) Sign(signer *asymmetric.PrivateKey) error {
	return r.Header.Sign(signer)
}

// DropDatabaseResponse defines client drop database rpc response entity.
type DropDatabaseResponse struct{}

// GetDatabaseRequestHeader defines client get database rpc request header entity.
type GetDatabaseRequestHeader struct {
	DatabaseID proto.DatabaseID
}

// SignedGetDatabaseRequestHeader defines signed client get database rpc request header entity.
type SignedGetDatabaseRequestHeader struct {
	GetDatabaseRequestHeader
	Hash      hash.Hash
	Signee    *asymmetric.PublicKey
	Signature *asymmetric.Signature
}

// Verify checks hash and signature in request header.
func (sh *SignedGetDatabaseRequestHeader) Verify() (err error) {
	// verify hash
	if err = verifyHash(&sh.GetDatabaseRequestHeader, &sh.Hash); err != nil {
		return
	}
	// verify sign
	if sh.Signee == nil || sh.Signature == nil || !sh.Signature.Verify(sh.Hash[:], sh.Signee) {
		return wt.ErrSignVerification
	}
	return
}

// Sign the request.
func (sh *SignedGetDatabaseRequestHeader) Sign(signer *asymmetric.PrivateKey) (err error) {
	// build hash
	if err = buildHash(&sh.GetDatabaseRequestHeader, &sh.Hash); err != nil {
		return
	}

	// sign
	sh.Signature, err = signer.Sign(sh.Hash[:])
	sh.Signee = signer.PubKey()

	return
}

// GetDatabaseRequest defines client get database rpc request entity.
type GetDatabaseRequest struct {
	proto.Envelope
	Header SignedGetDatabaseRequestHeader
}

// Verify checks hash and signature in request header.
func (r *GetDatabaseRequest) Verify() error {
	return r.Header.Verify()
}

// Sign the request.
func (r *GetDatabaseRequest) Sign(signer *asymmetric.PrivateKey) error {
	return r.Header.Sign(signer)
}

// GetDatabaseResponseHeader defines client get database rpc response header entity.
type GetDatabaseResponseHeader struct {
	InstanceMeta wt.ServiceInstance
}

// SignedGetDatabaseResponseHeader defines client get database rpc response header entity.
type SignedGetDatabaseResponseHeader struct {
	GetDatabaseResponseHeader
	Hash      hash.Hash
	Signee    *asymmetric.PublicKey
	Signature *asymmetric.Signature
}

// Verify checks hash and signature in response header.
func (sh *SignedGetDatabaseResponseHeader) Verify() (err error) {
	// verify hash
	if err = verifyHash(&sh.GetDatabaseResponseHeader, &sh.Hash); err != nil {
		return
	}
	// verify sign
	if sh.Signee == nil || sh.Signature == nil || !sh.Signature.Verify(sh.Hash[:], sh.Signee) {
		return wt.ErrSignVerification
	}
	return
}

// Sign the request.
func (sh *SignedGetDatabaseResponseHeader) Sign(signer *asymmetric.PrivateKey) (err error) {
	// build hash
	if err = buildHash(&sh.GetDatabaseResponseHeader, &sh.Hash); err != nil {
		return
	}

	// sign
	sh.Signature, err = signer.Sign(sh.Hash[:])
	sh.Signee = signer.PubKey()

	return
}

// GetDatabaseResponse defines client get database rpc response entity.
type GetDatabaseResponse struct {
	proto.Envelope
	Header SignedGetDatabaseResponseHeader
}

// Verify checks hash and signature in response header.
func (r *GetDatabaseResponse) Verify() (err error) {
	return r.Header.Verify()
}

// Sign the request.
func (r *GetDatabaseResponse) Sign(signer *asymmetric.PrivateKey) (err error) {
	return r.Header.Sign(signer)
}

// FIXME(xq262144) remove duplicated interface in utils package.
type canMarshalHash interface {
	MarshalHash() ([]byte, error)
}

func verifyHash(data canMarshalHash, h *hash.Hash) (err error) {
	var newHash hash.Hash
	if err = buildHash(data, &newHash); err != nil {
		return
	}
	if !newHash.IsEqual(h) {
		return ErrSignVerification
	}
	return
}

func buildHash(data canMarshalHash, h *hash.Hash) (err error) {
	var hashBytes []byte
	if hashBytes, err = data.MarshalHash(); err != nil {
		return
	}
	newHash := hash.THashH(hashBytes)
	copy(h[:], newHash[:])
	return
}
