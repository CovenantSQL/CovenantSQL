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
	"fmt"
	"math/big"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/golang/protobuf/proto"
	"github.com/thunderdb/ThunderDB/common"
	"github.com/thunderdb/ThunderDB/crypto/hash"
	"github.com/thunderdb/ThunderDB/crypto/signature"
	"github.com/thunderdb/ThunderDB/sqlchain/pbtypes"
)

// Header is a block header.
type Header struct {
	Version    int32
	Producer   common.Address
	RootHash   hash.Hash
	ParentHash hash.Hash
	MerkleRoot hash.Hash
	Timestamp  time.Time
}

func (h *Header) marshal() ([]byte, error) {
	buffer, err := h.Timestamp.MarshalJSON()

	if err != nil {
		return nil, err
	}

	return proto.Marshal(&pbtypes.Header{
		Version:    h.Version,
		Producer:   &pbtypes.Address{Address: h.Producer[:]},
		Root:       &pbtypes.Hash{Hash: h.RootHash[:]},
		Parent:     &pbtypes.Hash{Hash: h.ParentHash[:]},
		MerkleRoot: &pbtypes.Hash{Hash: h.MerkleRoot[:]},
		Timestamp:  buffer,
	})
}

// SignedHeader is block header along with its producer signature.
type SignedHeader struct {
	Header

	BlockHash hash.Hash
	Signee    *signature.PublicKey
	Signature *signature.Signature
}

func (s *SignedHeader) marshal() ([]byte, error) {
	buffer, err := s.Timestamp.MarshalJSON()

	if err != nil {
		return nil, err
	}

	return proto.Marshal(&pbtypes.SignedHeader{
		Header: &pbtypes.Header{
			Version:    s.Version,
			Producer:   &pbtypes.Address{Address: s.Producer[:]},
			Root:       &pbtypes.Hash{Hash: s.RootHash[:]},
			Parent:     &pbtypes.Hash{Hash: s.ParentHash[:]},
			MerkleRoot: &pbtypes.Hash{Hash: s.MerkleRoot[:]},
			Timestamp:  buffer,
		},
		BlockHash: &pbtypes.Hash{Hash: s.BlockHash[:]},
		Signee:    &pbtypes.PublicKey{PublicKey: s.Signee.Serialize()},
		Signature: func(s *signature.Signature) *pbtypes.Signature {
			if s == nil {
				return nil
			}
			return &pbtypes.Signature{
				S: s.S.String(),
				R: s.R.String(),
			}
		}(s.Signature),
	})
}

func (s *SignedHeader) unmarshal(buffer []byte) (err error) {
	pbSignedHeader := &pbtypes.SignedHeader{}
	err = proto.Unmarshal(buffer, pbSignedHeader)

	if err != nil {
		return err
	}

	pr := new(big.Int)
	ps := new(big.Int)

	pr, ok := pr.SetString(pbSignedHeader.GetSignature().GetR(), 10)

	if !ok {
		return fmt.Errorf("sqlchain: unexpected big int")
	}

	ps, ok = ps.SetString(pbSignedHeader.GetSignature().GetR(), 10)

	if !ok {
		return fmt.Errorf("sqlchain: unexpected big int")
	}

	if len(pbSignedHeader.GetHeader().GetProducer().GetAddress()) != common.AddressLength ||
		len(pbSignedHeader.GetHeader().GetRoot().GetHash()) != hash.HashSize ||
		len(pbSignedHeader.GetHeader().GetParent().GetHash()) != hash.HashSize ||
		len(pbSignedHeader.GetHeader().GetMerkleRoot().GetHash()) != hash.HashSize ||
		len(pbSignedHeader.GetBlockHash().GetHash()) != hash.HashSize {
		return fmt.Errorf("sqlchain: unexpected hash length")
	}

	t := time.Time{}
	err = t.UnmarshalJSON(pbSignedHeader.GetHeader().GetTimestamp())

	if err != nil {
		return err
	}

	pk, err := signature.ParsePubKey(pbSignedHeader.GetSignee().GetPublicKey(), btcec.S256())

	if err != nil {
		return err
	}

	// Copy fields
	s.Version = pbSignedHeader.Header.GetVersion()
	copy(s.Producer[:], pbSignedHeader.GetHeader().GetProducer().GetAddress())
	copy(s.RootHash[:], pbSignedHeader.GetHeader().GetRoot().GetHash())
	copy(s.ParentHash[:], pbSignedHeader.GetHeader().GetParent().GetHash())
	copy(s.MerkleRoot[:], pbSignedHeader.GetHeader().GetMerkleRoot().GetHash())
	copy(s.BlockHash[:], pbSignedHeader.GetBlockHash().GetHash())
	s.Timestamp = t
	s.Signature.R = pr
	s.Signature.S = ps
	s.Signee = pk

	return err
}

// Block is a node of blockchain.
type Block struct {
	SignedHeader *SignedHeader
	Queries      []*Query
}

// Blocks is Block (reference) array.
type Blocks []*Block

// SignHeader generates the signature for the Block from the given PrivateKey.
func (b *Block) SignHeader(signer *signature.PrivateKey) (err error) {
	buffer, err := b.SignedHeader.marshal()

	if err != nil {
		return err
	}

	b.SignedHeader.BlockHash = hash.DoubleHashH(buffer)
	b.SignedHeader.Signature, err = signer.Sign(b.SignedHeader.BlockHash[:])

	return err
}

// VerifyHeader verifies the signature of the Block.
func (b *Block) VerifyHeader() bool {
	return b.SignedHeader.Signature.Verify(b.SignedHeader.BlockHash[:], b.SignedHeader.Signee)
}
