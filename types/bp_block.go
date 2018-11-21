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
	BlockHash hash.Hash
	Signee    *asymmetric.PublicKey
	Signature *asymmetric.Signature
}

// Verify verifies the signature.
func (s *BPSignedHeader) Verify() error {
	if !s.Signature.Verify(s.BlockHash[:], s.Signee) {
		return ErrSignVerification
	}

	return nil
}

// BPBlock defines the main chain block.
type BPBlock struct {
	SignedHeader BPSignedHeader
	Transactions []pi.Transaction
}

// GetTxHashes returns all hashes of tx in block.{Billings, ...}
func (b *BPBlock) GetTxHashes() []*hash.Hash {
	// TODO(lambda): when you add new tx type, you need to put new tx's hash in the slice
	// get hashes in block.Transactions
	hs := make([]*hash.Hash, len(b.Transactions))

	for i, v := range b.Transactions {
		h := v.Hash()
		hs[i] = &h
	}
	return hs
}

// PackAndSignBlock computes block's hash and sign it.
func (b *BPBlock) PackAndSignBlock(signer *asymmetric.PrivateKey) error {
	hs := b.GetTxHashes()

	b.SignedHeader.MerkleRoot = *merkle.NewMerkle(hs).GetRoot()
	enc, err := b.SignedHeader.BPHeader.MarshalHash()

	if err != nil {
		return err
	}

	b.SignedHeader.BlockHash = hash.THashH(enc)
	b.SignedHeader.Signature, err = signer.Sign(b.SignedHeader.BlockHash[:])
	b.SignedHeader.Signee = signer.PubKey()

	if err != nil {
		return err
	}

	return nil
}

// Verify verifies whether the block is valid.
func (b *BPBlock) Verify() error {
	hs := b.GetTxHashes()
	merkleRoot := *merkle.NewMerkle(hs).GetRoot()
	if !merkleRoot.IsEqual(&b.SignedHeader.MerkleRoot) {
		return ErrMerkleRootVerification
	}

	enc, err := b.SignedHeader.BPHeader.MarshalHash()
	if err != nil {
		return err
	}

	h := hash.THashH(enc)
	if !h.IsEqual(&b.SignedHeader.BlockHash) {
		return ErrHashVerification
	}

	return b.SignedHeader.Verify()
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
	return &b.SignedHeader.BlockHash
}
