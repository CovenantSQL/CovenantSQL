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

	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/merkle"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/types"
)

//go:generate hsp

// BlockHeader defines a block header.
type BlockHeader struct {
	Version     int32
	Producer    proto.NodeID
	GenesisHash hash.Hash
	ParentHash  hash.Hash
	MerkleRoot  hash.Hash
	Timestamp   time.Time
}

// SignedBlockHeader defines a block along with its hasher, signer and verifier.
type SignedBlockHeader struct {
	BlockHeader
	DefaultHashSignVerifierImpl
}

// Sign signs the block header.
func (h *SignedBlockHeader) Sign(signer *asymmetric.PrivateKey) error {
	return h.DefaultHashSignVerifierImpl.Sign(&h.BlockHeader, signer)
}

// Verify verifies the block header.
func (h *SignedBlockHeader) Verify() error {
	return h.DefaultHashSignVerifierImpl.Verify(&h.BlockHeader)
}

// Block defines a block including a signed block header and its query list.
type Block struct {
	SignedBlockHeader
	ReadQueries  []*types.Ack
	WriteQueries []*types.Ack
}

// Sign signs the block.
func (b *Block) Sign(signer *asymmetric.PrivateKey) (err error) {
	// Update header fields: generate merkle root from queries
	var hashes []*hash.Hash
	for _, v := range b.ReadQueries {
		h := v.Header.Hash()
		hashes = append(hashes, &h)
	}
	for _, v := range b.WriteQueries {
		h := v.Header.Hash()
		hashes = append(hashes, &h)
	}
	if err = b.MerkleRoot.SetBytes(merkle.NewMerkle(hashes).GetRoot()[:]); err != nil {
		return
	}
	// Sign block header
	return b.SignedBlockHeader.Sign(signer)
}

// Verify verifies the block.
func (b *Block) Verify() error {
	// Verify header fields: compare merkle root from queries
	var hashes []*hash.Hash
	for _, v := range b.ReadQueries {
		h := v.Header.Hash()
		hashes = append(hashes, &h)
	}
	for _, v := range b.WriteQueries {
		h := v.Header.Hash()
		hashes = append(hashes, &h)
	}
	if mroot := merkle.NewMerkle(hashes).GetRoot(); !mroot.IsEqual(
		&b.SignedBlockHeader.MerkleRoot,
	) {
		return ErrMerkleRootNotMatch
	}
	// Verify block header signature
	return b.SignedBlockHeader.Verify()
}
