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

package types

import (
	"gitlab.com/thunderdb/ThunderDB/proto"
	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
	"time"
	"bytes"
	"gitlab.com/thunderdb/ThunderDB/utils"
	"encoding/binary"
	"gitlab.com/thunderdb/ThunderDB/crypto/asymmetric"
	"gitlab.com/thunderdb/ThunderDB/merkle"
)

type Header struct {
	Version int32
	Producer proto.AccountAddress
	MerkleRoot hash.Hash
	ParentHash hash.Hash
	Timestamp time.Time
}

func (h *Header) MarshalBinary() ([]byte, error) {
	buffer := bytes.NewBuffer(nil)

	err := utils.WriteElements(buffer, binary.BigEndian,
		h.Version,
		&h.Producer,
		&h.MerkleRoot,
		&h.ParentHash,
		h.Timestamp,
	)

	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func (h *Header) UnmarshalBinary(b []byte) error {
	reader := bytes.NewReader(b)

	return utils.ReadElements(reader, binary.BigEndian,
		&h.Version,
		&h.Producer,
		&h.MerkleRoot,
		&h.ParentHash,
		&h.Timestamp,
	)
}

type SignedHeader struct {
	Header
	BlockHash hash.Hash
	Signee *asymmetric.PublicKey
	Signature *asymmetric.Signature
}

func (s *SignedHeader) MarshalBinary() ([]byte, error) {
	buffer := bytes.NewBuffer(nil)

	err := utils.WriteElements(buffer, binary.BigEndian,
		s.Version,
		&s.Producer,
		&s.MerkleRoot,
		&s.ParentHash,
		s.Timestamp,
		&s.BlockHash,
		s.Signee,
		s.Signature,
	)

	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func (s *SignedHeader) UnmarshalBinary(b []byte) error {
	reader := bytes.NewReader(b)

	return utils.ReadElements(reader, binary.BigEndian,
		&s.Version,
		&s.Producer,
		&s.MerkleRoot,
		&s.ParentHash,
		&s.Timestamp,
		&s.BlockHash,
		&s.Signee,
		&s.Signature,
	)
}

func (s *SignedHeader) Verify() error {
	if !s.Signature.Verify(s.BlockHash[:], s.Signee) {
		return ErrSignVerification
	}

	return nil
}

type Block struct {
	SignedHeader SignedHeader
	Transactions []*hash.Hash
}

func (b *Block) PackAndSignBlock(signer *asymmetric.PrivateKey) error {
	b.SignedHeader.MerkleRoot = *merkle.NewMerkle(b.Transactions).GetRoot()
	enc, err := b.SignedHeader.Header.MarshalBinary()

	if err != nil {
		return err
	}

	b.SignedHeader.BlockHash = hash.THashH(enc)
	b.SignedHeader.Signature, err = signer.Sign(b.SignedHeader.BlockHash[:])

	if err != nil {
		return err
	}

	return nil
}

func (b *Block) MarshalBinary() ([]byte, error) {
	buffer := bytes.NewBuffer(nil)

	err := utils.WriteElements(buffer, binary.BigEndian,
		&b.SignedHeader,
		b.Transactions,
	)

	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func (b *Block) UnmarshalBinary(buf []byte) error {
	reader := bytes.NewReader(buf)

	return utils.ReadElements(reader, binary.BigEndian,
		&b.SignedHeader,
		&b.Transactions,
	)
}

func (b *Block) PushTx(tx *hash.Hash) {
	if b.Transactions != nil {
		// TODO(lambda): set appropriate capacity.
		b.Transactions = make([]*hash.Hash, 0, 100)
	}

	b.Transactions = append(b.Transactions, tx)
}

func (b *Block) Verify() error {
	merkleRoot := *merkle.NewMerkle(b.Transactions).GetRoot()
	if !merkleRoot.IsEqual(&b.SignedHeader.MerkleRoot) {
		return ErrMerkleRootVerification
	}

	enc, err := b.SignedHeader.Header.MarshalBinary()
	if err != nil {
		return err
	}

	h := hash.THashH(enc)
	if !h.IsEqual(&b.SignedHeader.BlockHash) {
		return ErrHashVerification
	}

	return b.SignedHeader.Verify()
}