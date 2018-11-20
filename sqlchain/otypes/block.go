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

package otypes

import (
	"reflect"
	"time"

	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/merkle"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

//go:generate hsp

// Header is a block header.
type Header struct {
	Version     int32
	Producer    proto.NodeID
	GenesisHash hash.Hash
	ParentHash  hash.Hash
	MerkleRoot  hash.Hash
	Timestamp   time.Time
}

// SignedHeader is block header along with its producer signature.
type SignedHeader struct {
	Header
	BlockHash hash.Hash
	Signee    *asymmetric.PublicKey
	Signature *asymmetric.Signature
}

// Verify verifies the signature of the signed header.
func (s *SignedHeader) Verify() error {
	if !s.Signature.Verify(s.BlockHash[:], s.Signee) {
		return ErrSignVerification
	}

	return nil
}

// VerifyAsGenesis verifies the signed header as a genesis block header.
func (s *SignedHeader) VerifyAsGenesis() (err error) {
	log.WithFields(log.Fields{
		"producer": s.Producer,
		"root":     s.GenesisHash.String(),
		"parent":   s.ParentHash.String(),
		"merkle":   s.MerkleRoot.String(),
		"block":    s.BlockHash.String(),
	}).Debug("verify genesis header")

	// Assume that we can fetch public key from kms after initialization.
	pk, err := kms.GetPublicKey(s.Producer)

	if err != nil {
		return
	}

	if !reflect.DeepEqual(pk, s.Signee) {
		return ErrNodePublicKeyNotMatch
	}

	return s.Verify()
}

// Block is a node of blockchain.
type Block struct {
	SignedHeader SignedHeader
	Queries      []*hash.Hash
}

// PackAndSignBlock generates the signature for the Block from the given PrivateKey.
func (b *Block) PackAndSignBlock(signer *asymmetric.PrivateKey) (err error) {
	// Calculate merkle root
	b.SignedHeader.MerkleRoot = *merkle.NewMerkle(b.Queries).GetRoot()
	buffer, err := b.SignedHeader.Header.MarshalHash()
	if err != nil {
		return
	}

	b.SignedHeader.Signee = signer.PubKey()
	b.SignedHeader.BlockHash = hash.THashH(buffer)
	b.SignedHeader.Signature, err = signer.Sign(b.SignedHeader.BlockHash[:])

	return
}

// PushAckedQuery pushes a acknowledged and verified query into the block.
func (b *Block) PushAckedQuery(h *hash.Hash) {
	if b.Queries == nil {
		// TODO(leventeliu): set appropriate capacity.
		b.Queries = make([]*hash.Hash, 0, 100)
	}

	b.Queries = append(b.Queries, h)
}

// Verify verifies the merkle root and header signature of the block.
func (b *Block) Verify() (err error) {
	// Verify merkle root
	if MerkleRoot := *merkle.NewMerkle(b.Queries).GetRoot(); !MerkleRoot.IsEqual(
		&b.SignedHeader.MerkleRoot,
	) {
		return ErrMerkleRootVerification
	}

	// Verify block hash
	buffer, err := b.SignedHeader.Header.MarshalHash()
	if err != nil {
		return
	}

	if h := hash.THashH(buffer); !h.IsEqual(&b.SignedHeader.BlockHash) {
		return ErrHashVerification
	}

	// Verify signature
	return b.SignedHeader.Verify()
}

// VerifyAsGenesis verifies the block as a genesis block.
func (b *Block) VerifyAsGenesis() (err error) {
	// Assume that we can fetch public key from kms after initialization.
	pk, err := kms.GetPublicKey(b.Producer())

	if err != nil {
		return
	}

	if !reflect.DeepEqual(pk, b.SignedHeader.Signee) {
		return ErrNodePublicKeyNotMatch
	}

	return b.Verify()
}

// Timestamp returns the timestamp field of the block header.
func (b *Block) Timestamp() time.Time {
	return b.SignedHeader.Timestamp
}

// Producer returns the producer field of the block header.
func (b *Block) Producer() proto.NodeID {
	return b.SignedHeader.Producer
}

// ParentHash returns the parent hash field of the block header.
func (b *Block) ParentHash() *hash.Hash {
	return &b.SignedHeader.ParentHash
}

// BlockHash returns the parent hash field of the block header.
func (b *Block) BlockHash() *hash.Hash {
	return &b.SignedHeader.BlockHash
}

// GenesisHash returns the parent hash field of the block header.
func (b *Block) GenesisHash() *hash.Hash {
	return &b.SignedHeader.GenesisHash
}

// Signee returns the signee field  of the block signed header.
func (b *Block) Signee() *asymmetric.PublicKey {
	return b.SignedHeader.Signee
}

// Blocks is Block (reference) array.
type Blocks []*Block
