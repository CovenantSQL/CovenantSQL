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
	"time"

	pi "github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/crypto/verifier"
	"github.com/CovenantSQL/CovenantSQL/merkle"
	"github.com/CovenantSQL/CovenantSQL/proto"
)

//go:generate hsp

// BPHeader defines the main chain block header.
type BPHeader struct {
	Version    int32
	Producer   proto.AccountAddress
	MerkleRoot hash.Hash
	ParentHash hash.Hash
	Timestamp  time.Time
}

// BPSignedHeader defines the main chain header with the signature.
type BPSignedHeader struct {
	BPHeader
	verifier.DefaultHashSignVerifierImpl
}

func (s *BPSignedHeader) verifyHash() error {
	return s.DefaultHashSignVerifierImpl.VerifyHash(&s.BPHeader)
}

func (s *BPSignedHeader) verify() error {
	return s.DefaultHashSignVerifierImpl.Verify(&s.BPHeader)
}

func (s *BPSignedHeader) setHash() error {
	return s.DefaultHashSignVerifierImpl.SetHash(&s.BPHeader)
}

func (s *BPSignedHeader) sign(signer *asymmetric.PrivateKey) error {
	return s.DefaultHashSignVerifierImpl.Sign(&s.BPHeader, signer)
}

// BPBlock defines the main chain block.
type BPBlock struct {
	SignedHeader BPSignedHeader
	Transactions []pi.Transaction
}

// calcMerkleRoot commits to the complete transactions in the block.
// Transaction.Hash remains the header hash used by transaction pools and indexes.
func (b *BPBlock) calcMerkleRoot() (*hash.Hash, error) {
	hs := make([]*hash.Hash, len(b.Transactions))

	for i, v := range b.Transactions {
		if v == nil {
			return nil, ErrNilBlockMember
		}
		enc, err := v.MarshalHash()
		if err != nil {
			return nil, err
		}
		h := hash.THashH(enc)
		hs[i] = &h
	}
	return merkle.NewMerkle(hs).GetRoot(), nil
}

func (b *BPBlock) setMerkleRoot() error {
	merkleRoot, err := b.calcMerkleRoot()
	if err != nil {
		return err
	}
	b.SignedHeader.MerkleRoot = *merkleRoot
	return nil
}

func (b *BPBlock) verifyMerkleRoot() error {
	merkleRoot, err := b.calcMerkleRoot()
	if err != nil {
		return err
	}
	if !merkleRoot.IsEqual(&b.SignedHeader.MerkleRoot) {
		return ErrMerkleRootVerification
	}
	return nil
}

// SetHash sets the block header hash, including the merkle root of the packed transactions.
func (b *BPBlock) SetHash() error {
	if err := b.setMerkleRoot(); err != nil {
		return err
	}
	return b.SignedHeader.setHash()
}

// VerifyHash verifies the block header hash, including the merkle root of the packed transactions.
func (b *BPBlock) VerifyHash() error {
	if err := b.verifyMerkleRoot(); err != nil {
		return err
	}
	return b.SignedHeader.verifyHash()
}

// PackAndSignBlock computes block's hash and sign it.
func (b *BPBlock) PackAndSignBlock(signer *asymmetric.PrivateKey) error {
	if err := b.setMerkleRoot(); err != nil {
		return err
	}
	return b.SignedHeader.sign(signer)
}

// Verify verifies whether the block is valid.
func (b *BPBlock) Verify() error {
	if err := b.verifyMerkleRoot(); err != nil {
		return err
	}
	return b.SignedHeader.verify()
}

// Timestamp returns timestamp of block.
func (b *BPBlock) Timestamp() time.Time {
	return b.SignedHeader.Timestamp
}

// Producer returns the producer of block.
func (b *BPBlock) Producer() proto.AccountAddress {
	return b.SignedHeader.Producer
}

// ParentHash returns the parent hash field of the block header.
func (b *BPBlock) ParentHash() *hash.Hash {
	return &b.SignedHeader.ParentHash
}

// BlockHash returns the parent hash field of the block header.
func (b *BPBlock) BlockHash() *hash.Hash {
	return &b.SignedHeader.DataHash
}
