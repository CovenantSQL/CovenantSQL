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

// Package sqlchain provides a blockchain implementation for database state tracking.
package sqlchain

import (
	"bytes"
	"encoding/gob"
	"time"

	"github.com/thunderdb/ThunderDB/common"
	"github.com/thunderdb/ThunderDB/crypto/hash"
	"github.com/thunderdb/ThunderDB/crypto/sign"
)

// Header is a block header
type Header struct {
	Version    int32
	Producer   common.Address
	RootHash   hash.Hash
	ParentHash hash.Hash
	MerkleRoot hash.Hash
	TimeStamp  time.Time
}

// SignedHeader is block header along with its producer signature
type SignedHeader struct {
	Header

	BlockHash hash.Hash
	Signee    *sign.PublicKey
	Signature *sign.Signature
}

// Block is a node of blockchain
type Block struct {
	SignedHeader *SignedHeader
	Queries      []*Query
}

// Blocks is Block (reference) array
type Blocks []*Block

// SignHeader generates the signature for the Block from the given PrivateKey
func (b *Block) SignHeader(signer *sign.PrivateKey) (err error) {
	var buffer bytes.Buffer
	enc := gob.NewEncoder(&buffer)
	err = enc.Encode(b.SignedHeader.Header)

	if err != nil {
		return err
	}

	b.SignedHeader.BlockHash = hash.DoubleHashH(buffer.Bytes())
	b.SignedHeader.Signature, err = signer.Sign(b.SignedHeader.BlockHash[:])

	return err
}

// VerifyHeader verifies the signature of the Block
func (b *Block) VerifyHeader() bool {
	return b.SignedHeader.Signature.Verify(b.SignedHeader.BlockHash[:], b.SignedHeader.Signee)
}
