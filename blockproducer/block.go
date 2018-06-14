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

package blockproducer

import (
	"time"

	"math/big"

	"github.com/golang/protobuf/proto"
	"github.com/thunderdb/ThunderDB/crypto/asymmetric"
	"github.com/thunderdb/ThunderDB/crypto/hash"
	proto2 "github.com/thunderdb/ThunderDB/proto"
	"github.com/thunderdb/ThunderDB/types"
)

// Header is the header structure that will be signed
type Header struct {
	Version    int32
	Producer   proto2.AccountAddress
	Root       hash.Hash
	Parent     hash.Hash
	MerkleRoot hash.Hash
	Timestamp  time.Time
}

func (h *Header) marshal() ([]byte, error) {
	return proto.Marshal(&types.BPHeader{
		Version:    h.Version,
		Producer:   &types.AccountAddress{AccountAddress: string(h.Producer)},
		Root:       &types.Hash{Hash: h.Root[:]},
		Parent:     &types.Hash{Hash: h.Parent[:]},
		MerkleRoot: &types.Hash{Hash: h.MerkleRoot[:]},
		Timestamp:  h.Timestamp.UnixNano(),
	})
}

func (h *Header) fromBPHeader(bpHeader *types.BPHeader) error {
	rootHash, err := hash.NewHash(bpHeader.Root.Hash)
	if err != nil {
		return err
	}

	parentHash, err := hash.NewHash(bpHeader.Parent.Hash)
	if err != nil {
		return err
	}

	MerkleRootHash, err := hash.NewHash(bpHeader.MerkleRoot.Hash)
	if err != nil {
		return err
	}

	h.Version = bpHeader.Version
	h.Producer = proto2.AccountAddress(bpHeader.Producer.AccountAddress)
	h.Root = *rootHash
	h.Parent = *parentHash
	h.MerkleRoot = *MerkleRootHash
	h.Timestamp = time.Unix(0, bpHeader.Timestamp).UTC()

	return nil
}

// SignedHeader is the full header structure including
// Header and signature information
type SignedHeader struct {
	Header
	BlockHash hash.Hash
	PublicKey *asymmetric.PublicKey
	Signature *asymmetric.Signature
}

func (h *SignedHeader) toBPSignedHeader() *types.BPSignedHeader {
	return &types.BPSignedHeader{
		Header: &types.BPHeader{
			Version:    h.Version,
			Producer:   &types.AccountAddress{AccountAddress: string(h.Producer)},
			Root:       &types.Hash{Hash: h.Root[:]},
			Parent:     &types.Hash{Hash: h.Parent[:]},
			MerkleRoot: &types.Hash{Hash: h.MerkleRoot[:]},
			Timestamp:  h.Timestamp.UnixNano(),
		},
		BlockHash: &types.Hash{Hash: h.BlockHash[:]},
		Signee: &types.PublicKey{
			PublicKey: h.PublicKey.Serialize(),
		},
		Signature: &types.Signature{
			R: h.Signature.R.String(),
			S: h.Signature.S.String(),
		},
	}
}

func (h *SignedHeader) fromBPSignedHeader(bpSignedHeader *types.BPSignedHeader) error {
	blockHash, err := hash.NewHash(bpSignedHeader.BlockHash.Hash)
	if err != nil {
		return err
	}

	publicKey, err := asymmetric.ParsePubKey(bpSignedHeader.Signee.PublicKey)
	if err != nil {
		return err
	}

	r := new(big.Int)
	r.SetString(bpSignedHeader.Signature.R, 10)
	s := new(big.Int)
	s.SetString(bpSignedHeader.Signature.S, 10)

	err = h.fromBPHeader(bpSignedHeader.Header)
	if err != nil {
		return err
	}

	h.BlockHash = *blockHash
	h.PublicKey = publicKey
	h.Signature = &asymmetric.Signature{
		R: r,
		S: s,
	}
	return nil
}

func (h *SignedHeader) marshal() ([]byte, error) {
	return proto.Marshal(h.toBPSignedHeader())
}

func (h *SignedHeader) unmarshal(buff []byte) error {
	pbBPSignedHeader := types.BPSignedHeader{}
	err := proto.Unmarshal(buff, &pbBPSignedHeader)
	if err != nil {
		return err
	}

	err = h.fromBPSignedHeader(&pbBPSignedHeader)
	if err != nil {
		return err
	}

	return nil
}

// TxData is the details of a tx
type TxData struct {
	AccountNonce uint64
	Recipient    *types.AccountAddress
	Amount       *big.Int
	Payload      []byte

	Signature *asymmetric.Signature
	PublicKey *asymmetric.PublicKey
}

func (t *TxData) toBPTxData() *types.BPTxData {
	return &types.BPTxData{
		AccountNonce: t.AccountNonce,
		Recipient:    t.Recipient,
		Amount:       t.Amount.Bytes(),
		Payload:      t.Payload,
		Signature: &types.Signature{
			R: t.Signature.R.String(),
			S: t.Signature.S.String(),
		},
		Signee: &types.PublicKey{
			PublicKey: t.PublicKey.Serialize(),
		},
	}
}

func (t *TxData) marshal() ([]byte, error) {
	return proto.Marshal(t.toBPTxData())
}

func (t *TxData) fromBPTxData(bpTxData *types.BPTxData) error {
	publicKey, err := asymmetric.ParsePubKey(bpTxData.Signee.PublicKey)
	if err != nil {
		return err
	}

	amount := new(big.Int)
	amount.SetBytes(bpTxData.Amount)

	r := new(big.Int)
	r.SetString(bpTxData.Signature.R, 10)
	s := new(big.Int)
	s.SetString(bpTxData.Signature.S, 10)

	t.AccountNonce = bpTxData.AccountNonce
	t.Recipient = bpTxData.Recipient
	t.Amount = amount
	t.Payload = bpTxData.Payload
	t.Signature = &asymmetric.Signature{
		R: r,
		S: s,
	}
	t.PublicKey = publicKey

	return nil
}

// Tx includes TxData and TxData's hash
type Tx struct {
	TxHash hash.Hash
	TxData TxData
}

func (t *Tx) fromBPTx(BPTx *types.BPTx) error {
	txHash, err := hash.NewHash(BPTx.TxHash.Hash)
	if err != nil {
		return err
	}

	err = t.TxData.fromBPTxData(BPTx.TxData)
	if err != nil {
		return err
	}

	t.TxHash = *txHash

	return nil
}

func (t *Tx) toBPTx() *types.BPTx {
	return &types.BPTx{
		// lambda: warning!!!! it will not pass the test case if i use t.TxHash[:] and i do not know why
		TxHash: &types.Hash{Hash: t.TxHash[:]},
		TxData: t.TxData.toBPTxData(),
	}
}

func (t *Tx) marshal() ([]byte, error) {
	return proto.Marshal(t.toBPTx())
}

func (t *Tx) unmarshal(buff []byte) error {
	bpTx := &types.BPTx{}
	err := proto.Unmarshal(buff, bpTx)
	if err != nil {
		return err
	}

	err = t.fromBPTx(bpTx)
	return err
}

// Block is generated by Block Producer
type Block struct {
	Header *SignedHeader
	Tx     []*Tx
}

func (b *Block) marshal() ([]byte, error) {
	bpTx := make([]*types.BPTx, len(b.Tx))
	size := len(b.Tx)

	for i := 0; i < size; i++ {
		bpTx[i] = b.Tx[i].toBPTx()
	}

	return proto.Marshal(&types.BPBlock{
		Header: b.Header.toBPSignedHeader(),
		Tx:     bpTx[:],
	})
}

func (b *Block) unmarshal(buff []byte) error {
	block := &types.BPBlock{}
	err := proto.Unmarshal(buff, block)
	if err != nil {
		return err
	}

	header := SignedHeader{}
	err = header.fromBPSignedHeader(block.Header)
	if err != nil {
		return err
	}

	b.Header = &header

	txes := make([]*Tx, len(block.Tx))

	size := len(block.Tx)
	for i := 0; i < size; i++ {
		txes[i] = &Tx{}
		err = txes[i].fromBPTx(block.Tx[i])
		if err != nil {
			return err
		}
	}

	b.Tx = txes
	return nil
}
