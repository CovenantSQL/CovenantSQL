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
	ct "github.com/CovenantSQL/CovenantSQL/sqlchain/types"
)

//go:generate hsp

// InitService defines worker service init request.
type InitService struct {
	proto.Envelope
}

// ResourceMeta defines single database resource meta.
type ResourceMeta struct {
	Node          uint16              // reserved node count
	Space         uint64              // reserved storage space in bytes
	Memory        uint64              // reserved memory in bytes
	LoadAvgPerCPU uint64              // max loadAvg15 per CPU
	EncryptionKey string `hspack:"-"` // encryption key for database instance
}

// ServiceInstance defines single instance to be initialized.
type ServiceInstance struct {
	DatabaseID   proto.DatabaseID
	Peers        *proto.Peers
	ResourceMeta ResourceMeta
	GenesisBlock *ct.Block
}

// InitServiceResponseHeader defines worker service init response header.
type InitServiceResponseHeader struct {
	Instances []ServiceInstance
}

// SignedInitServiceResponseHeader defines signed worker service init response header.
type SignedInitServiceResponseHeader struct {
	InitServiceResponseHeader
	Hash      hash.Hash
	Signee    *asymmetric.PublicKey
	Signature *asymmetric.Signature
}

// InitServiceResponse defines worker service init response.
type InitServiceResponse struct {
	Header SignedInitServiceResponseHeader
}

// Verify checks hash and signature in init service response header.
func (sh *SignedInitServiceResponseHeader) Verify() (err error) {
	// verify hash
	if err = verifyHash(&sh.InitServiceResponseHeader, &sh.Hash); err != nil {
		return
	}
	// verify sign
	if sh.Signee == nil || sh.Signature == nil || !sh.Signature.Verify(sh.Hash[:], sh.Signee) {
		return ErrSignVerification
	}
	return
}

// Sign the request.
func (sh *SignedInitServiceResponseHeader) Sign(signer *asymmetric.PrivateKey) (err error) {
	// build hash
	if err = buildHash(&sh.InitServiceResponseHeader, &sh.Hash); err != nil {
		return
	}

	// sign
	sh.Signature, err = signer.Sign(sh.Hash[:])
	sh.Signee = signer.PubKey()

	return
}

// Verify checks hash and signature in init service response header.
func (rs *InitServiceResponse) Verify() error {
	return rs.Header.Verify()
}

// Sign the request.
func (rs *InitServiceResponse) Sign(signer *asymmetric.PrivateKey) (err error) {
	// sign
	return rs.Header.Sign(signer)
}
