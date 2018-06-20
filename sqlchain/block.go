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
	"bytes"
	"encoding/binary"
	"reflect"
	"time"

	log "github.com/sirupsen/logrus"
	"gitlab.com/thunderdb/ThunderDB/crypto/asymmetric"
	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
	"gitlab.com/thunderdb/ThunderDB/crypto/kms"
	"gitlab.com/thunderdb/ThunderDB/proto"
	"gitlab.com/thunderdb/ThunderDB/utils"
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
	buffer := bytes.NewBuffer(nil)

	if err := utils.WriteElements(buffer, binary.BigEndian,
		h.Version,
		h.Producer,
		&h.RootHash,
		&h.ParentHash,
		&h.MerkleRoot,
		h.Timestamp,
	); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

// SignedHeader is block header along with its producer signature.
type SignedHeader struct {
	Header

	BlockHash hash.Hash
	Signee    *asymmetric.PublicKey
	Signature *asymmetric.Signature
}

func (s *SignedHeader) marshal() ([]byte, error) {
	buffer := bytes.NewBuffer(nil)

	if err := utils.WriteElements(buffer, binary.BigEndian,
		s.Version,
		s.Producer,
		&s.RootHash,
		&s.ParentHash,
		&s.MerkleRoot,
		s.Timestamp,
		&s.BlockHash,
		s.Signee,
		s.Signature,
	); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func (s *SignedHeader) unmarshal(b []byte) error {
	reader := bytes.NewReader(b)
	return utils.ReadElements(reader, binary.BigEndian,
		&s.Version,
		&s.Producer,
		&s.RootHash,
		&s.ParentHash,
		&s.MerkleRoot,
		&s.Timestamp,
		&s.BlockHash,
		&s.Signee,
		&s.Signature,
	)
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
	log.Debugf("verify genesis header: producer = %s, root = %s, parent = %s, merkle = %s,"+
		" block = %s",
		string(s.Producer[:]),
		s.RootHash.String(),
		s.ParentHash.String(),
		s.MerkleRoot.String(),
		s.BlockHash.String(),
	)

	// Assume that we can fetch public key from kms after initialization.
	pk, err := kms.GetPublicKey(proto.NodeID(s.Header.Producer[:]))

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

// Verify verifies the merkle root and header signature of the block.
func (b *Block) Verify() (err error) {
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

// VerifyAsGenesis verifies the block as a genesis block.
func (b *Block) VerifyAsGenesis() (err error) {
	if b.SignedHeader == nil {
		return ErrNilValue
	}

	// Assume that we can fetch public key from kms after initialization.
	pk, err := kms.GetPublicKey(proto.NodeID(b.SignedHeader.Header.Producer[:]))

	if err != nil {
		return
	}

	if !reflect.DeepEqual(pk, b.SignedHeader.Signee) {
		return ErrNodePublicKeyNotMatch
	}

	return b.Verify()
}

// Blocks is Block (reference) array.
type Blocks []*Block
