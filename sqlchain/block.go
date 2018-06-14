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

package sqlchain

import (
	"math/big"
	"time"

	"github.com/btcsuite/btcd/btcec"
	pb "github.com/golang/protobuf/proto"
	"github.com/thunderdb/ThunderDB/common"
	"github.com/thunderdb/ThunderDB/crypto/asymmetric"
	"github.com/thunderdb/ThunderDB/crypto/hash"
	"github.com/thunderdb/ThunderDB/proto"
	"github.com/thunderdb/ThunderDB/types"
)

// Header is a block header.
type Header struct {
	Version    int32
	Producer   proto.NodeID
	RootHash   hash.Hash
	ParentHash hash.Hash
	MerkleRoot hash.Hash
	Timestamp  time.Time
}

func (h *Header) marshal() ([]byte, error) {
	return pb.Marshal(&types.Header{
		Version:    h.Version,
		Producer:   &types.NodeID{NodeID: string(h.Producer)},
		Root:       &types.Hash{Hash: h.RootHash[:]},
		Parent:     &types.Hash{Hash: h.ParentHash[:]},
		MerkleRoot: &types.Hash{Hash: h.MerkleRoot[:]},
		Timestamp:  h.Timestamp.UnixNano(),
	})
}

// SignedHeader is block header along with its producer signature.
type SignedHeader struct {
	Header

	BlockHash hash.Hash
	Signee    *asymmetric.PublicKey
	Signature *asymmetric.Signature
}

func (s *SignedHeader) marshal() ([]byte, error) {
	return pb.Marshal(&types.SignedHeader{
		Header: &types.Header{
			Version:    s.Version,
			Producer:   &types.NodeID{NodeID: string(s.Producer)},
			Root:       &types.Hash{Hash: s.RootHash[:]},
			Parent:     &types.Hash{Hash: s.ParentHash[:]},
			MerkleRoot: &types.Hash{Hash: s.MerkleRoot[:]},
			Timestamp:  s.Timestamp.UnixNano(),
		},
		BlockHash: &types.Hash{Hash: s.BlockHash[:]},
		Signee:    &types.PublicKey{PublicKey: s.Signee.Serialize()},
		Signature: func(s *asymmetric.Signature) *types.Signature {
			if s == nil {
				return nil
			}
			return &types.Signature{
				S: s.S.String(),
				R: s.R.String(),
			}
		}(s.Signature),
	})
}

func (s *SignedHeader) unmarshal(buffer []byte) (err error) {
	pbSignedHeader := &types.SignedHeader{}
	err = pb.Unmarshal(buffer, pbSignedHeader)

	if err != nil {
		return
	}

	pr := new(big.Int)
	ps := new(big.Int)

	pr, ok := pr.SetString(pbSignedHeader.GetSignature().GetR(), 10)

	if !ok {
		return ErrFieldConversion
	}

	ps, ok = ps.SetString(pbSignedHeader.GetSignature().GetS(), 10)

	if !ok {
		return ErrFieldConversion
	}

	if len(pbSignedHeader.GetHeader().GetProducer().GetNodeID()) != common.AddressLength ||
		len(pbSignedHeader.GetHeader().GetRoot().GetHash()) != hash.HashSize ||
		len(pbSignedHeader.GetHeader().GetParent().GetHash()) != hash.HashSize ||
		len(pbSignedHeader.GetHeader().GetMerkleRoot().GetHash()) != hash.HashSize ||
		len(pbSignedHeader.GetBlockHash().GetHash()) != hash.HashSize {
		return ErrFieldLength
	}

	pk, err := asymmetric.ParsePubKey(pbSignedHeader.GetSignee().GetPublicKey(), btcec.S256())

	if err != nil {
		return
	}

	// Copy fields
	s.Version = pbSignedHeader.Header.GetVersion()
	s.Producer = proto.NodeID(pbSignedHeader.GetHeader().GetProducer().GetNodeID())
	copy(s.RootHash[:], pbSignedHeader.GetHeader().GetRoot().GetHash())
	copy(s.ParentHash[:], pbSignedHeader.GetHeader().GetParent().GetHash())
	copy(s.MerkleRoot[:], pbSignedHeader.GetHeader().GetMerkleRoot().GetHash())
	copy(s.BlockHash[:], pbSignedHeader.GetBlockHash().GetHash())
	s.Timestamp = time.Unix(0, pbSignedHeader.GetHeader().GetTimestamp()).UTC()
	s.Signature = &asymmetric.Signature{
		R: pr,
		S: ps,
	}
	s.Signee = pk

	return
}

// Verify verifies the signature of the SignedHeader.
func (s *SignedHeader) Verify() error {
	if !s.Signature.Verify(s.BlockHash[:], s.Signee) {
		return ErrSignVerification
	}

	return nil
}

// Block is a node of blockchain.
type Block struct {
	SignedHeader *SignedHeader
	Queries      []*Query
}

// SignHeader generates the signature for the Block from the given PrivateKey.
func (b *Block) SignHeader(signer *asymmetric.PrivateKey) (err error) {
	buffer, err := b.SignedHeader.Header.marshal()

	if err != nil {
		return
	}

	b.SignedHeader.BlockHash = hash.THashH(buffer)
	b.SignedHeader.Signature, err = signer.Sign(b.SignedHeader.BlockHash[:])

	return
}

// VerifyHeader verifies the signature of the Block.
func (b *Block) VerifyHeader() (err error) {
	// TODO(leventeliu): verify merkle root of queries
	// ...

	// Verify block hash
	buffer, err := b.SignedHeader.Header.marshal()

	if err != nil {
		return
	}

	h := hash.THashH(buffer)

	if !h.IsEqual(&b.SignedHeader.BlockHash) {
		return ErrHashVerification
	}

	// Verify signature
	return b.SignedHeader.Verify()
}

// Blocks is Block (reference) array.
type Blocks []*Block
