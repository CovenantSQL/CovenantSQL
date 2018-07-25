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
	"bytes"
	"encoding/binary"

	"gitlab.com/thunderdb/ThunderDB/crypto/asymmetric"

	"gitlab.com/thunderdb/ThunderDB/crypto/hash"

	"gitlab.com/thunderdb/ThunderDB/utils"

	"gitlab.com/thunderdb/ThunderDB/proto"
)

// TxContent defines the customer's billing and block rewards in transaction
type TxContent struct {
	SequenceID     uint64
	TxType         byte
	BillingRequest BillingRequest
	Receivers      []*proto.AccountAddress
	// Fee paid by stable coin
	Fees []uint64
	// Reward is share coin
	Rewards         []uint64
	BillingResponse BillingResponse
}

// MarshalBinary implements BinaryMarshaler.
func (tb *TxContent) MarshalBinary() ([]byte, error) {
	buffer := bytes.NewBuffer(nil)

	err := utils.WriteElements(buffer, binary.BigEndian,
		tb.SequenceID,
		tb.TxType,
		&tb.BillingRequest,
		tb.Receivers,
		tb.Fees,
		tb.Rewards,
		&tb.BillingResponse,
	)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// UnmarshalBinary implements BinaryUnmarshaler.
func (tb *TxContent) UnmarshalBinary(b []byte) error {
	reader := bytes.NewReader(b)

	err := utils.ReadElements(reader, binary.BigEndian,
		&tb.SequenceID,
		&tb.TxType,
		&tb.BillingRequest,
		&tb.Receivers,
		&tb.Fees,
		&tb.Rewards,
		&tb.BillingResponse,
	)
	if err != nil {
		return err
	}
	return nil
}

// GetHash returns the hash of transaction
func (tb *TxContent) GetHash() (*hash.Hash, error) {
	b, err := tb.MarshalBinary()
	if err != nil {
		return nil, err
	}
	h := hash.THashH(b)
	return &h, nil
}

// GetType returns the type of transaction
func (tb *TxContent) GetType() byte {
	return tb.TxType
}

// TxBilling is a type of tx, that is used to record sql chain billing and block rewards
type TxBilling struct {
	TxContent      TxContent
	AccountAddress *proto.AccountAddress
	TxHash         *hash.Hash
	Signee         *asymmetric.PublicKey
	Signature      *asymmetric.Signature
	SignedBlock    *hash.Hash
}

// Serialize serializes TxBilling using msgpack
func (tb *TxBilling) Serialize() ([]byte, error) {
	b, err := utils.EncodeMsgPack(tb)
	if err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

// Deserialize desrializes TxBilling using msgpack
func (tb *TxBilling) Deserialize(enc []byte) error {
	err := utils.DecodeMsgPack(enc, tb)
	return err
}

// PackAndSignTx computes tx of TxContent and signs it
func (tb *TxBilling) PackAndSignTx(signer *asymmetric.PrivateKey) error {
	enc, err := tb.TxContent.MarshalBinary()
	if err != nil {
		return err
	}
	h := hash.THashH(enc)
	tb.TxHash = &h

	pub := asymmetric.PublicKey(signer.PublicKey)
	tb.Signee = &pub

	signature, err := signer.Sign(h[:])
	if err != nil {
		return err
	}
	tb.Signature = signature

	return nil
}

// Verify verifies the signature of TxBilling
func (tb *TxBilling) Verify() (err error) {
	if !tb.Signature.Verify(tb.TxHash[:], tb.Signee) {
		err = ErrSignVerification
	}
	return
}
