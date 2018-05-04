/*
 * MIT License
 *
 * Copyright (c) 2016-2018. ThunderDB
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
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
