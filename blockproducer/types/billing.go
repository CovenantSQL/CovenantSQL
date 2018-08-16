/*
 * Copyright 2018 The ThunderDB Authors.
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
	"bytes"
	"encoding/binary"

	"gitlab.com/thunderdb/ThunderDB/crypto/asymmetric"
	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
	"gitlab.com/thunderdb/ThunderDB/proto"
	"gitlab.com/thunderdb/ThunderDB/utils"
)

// BillingRequestHeader includes contents that need to be signed. Billing blocks should be within
// height range [low, high] (inclusive).
type BillingRequestHeader struct {
	DatabaseID proto.DatabaseID
	// sqlchain block hash and its height
	LowBlock   hash.Hash
	LowHeight  int32
	HighBlock  hash.Hash
	HighHeight int32
	GasAmounts []*proto.AddrAndGas
}

// MarshalBinary implements BinaryMarshaler.
func (bh *BillingRequestHeader) MarshalBinary() ([]byte, error) {
	buffer := bytes.NewBuffer(nil)

	err := utils.WriteElements(buffer, binary.BigEndian,
		&bh.DatabaseID,
		&bh.LowBlock,
		&bh.LowHeight,
		&bh.HighBlock,
		&bh.HighHeight,
		&bh.GasAmounts,
	)

	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// UnmarshalBinary implements BinaryUnmarshaler.
func (bh *BillingRequestHeader) UnmarshalBinary(b []byte) error {
	reader := bytes.NewReader(b)

	err := utils.ReadElements(reader, binary.BigEndian,
		&bh.DatabaseID,
		&bh.LowBlock,
		&bh.LowHeight,
		&bh.HighBlock,
		&bh.HighHeight,
		&bh.GasAmounts,
	)
	if err != nil {
		return err
	}
	return nil
}

// BillingRequest defines periodically Billing sync
type BillingRequest struct {
	Header      BillingRequestHeader
	RequestHash hash.Hash
	Signees     []*asymmetric.PublicKey
	Signatures  []*asymmetric.Signature
}

// MarshalBinary implements BinaryMarshaler.
func (br *BillingRequest) MarshalBinary() ([]byte, error) {
	buffer := bytes.NewBuffer(nil)

	err := utils.WriteElements(buffer, binary.BigEndian,
		&br.Header,
		&br.RequestHash,
		&br.Signees,
		&br.Signatures,
	)

	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// UnmarshalBinary implements BinaryUnmarshaler.
func (br *BillingRequest) UnmarshalBinary(b []byte) error {
	reader := bytes.NewReader(b)

	err := utils.ReadElements(reader, binary.BigEndian,
		&br.Header,
		&br.RequestHash,
		&br.Signees,
		&br.Signatures,
	)
	if err != nil {
		return err
	}
	return nil
}

// PackRequestHeader computes the hash of header
func (br *BillingRequest) PackRequestHeader() (*hash.Hash, error) {
	b, err := br.Header.MarshalBinary()
	if err != nil {
		return nil, err
	}

	h := hash.THashH(b)
	return &h, nil
}

// SignRequestHeader first computes the hash of BillingRequestHeader, then signs the request
func (br *BillingRequest) SignRequestHeader(signee *asymmetric.PrivateKey) (*asymmetric.Signature, error) {
	signature, err := signee.Sign(br.RequestHash[:])
	if err != nil {
		return nil, err
	}
	return signature, nil
}

// BillingResponse defines the the response for BillingRequest
type BillingResponse struct {
	AccountAddress proto.AccountAddress
	RequestHash    hash.Hash
	Signee         *asymmetric.PublicKey
	Signature      *asymmetric.Signature
}

// MarshalBinary implements BinaryMarshaler.
func (br *BillingResponse) MarshalBinary() ([]byte, error) {
	buffer := bytes.NewBuffer(nil)

	err := utils.WriteElements(buffer, binary.BigEndian,
		&br.AccountAddress,
		&br.RequestHash,
		&br.Signee,
		&br.Signature,
	)

	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// UnmarshalBinary implements BinaryUnmarshaler.
func (br *BillingResponse) UnmarshalBinary(b []byte) error {
	reader := bytes.NewReader(b)

	err := utils.ReadElements(reader, binary.BigEndian,
		&br.AccountAddress,
		&br.RequestHash,
		&br.Signee,
		&br.Signature,
	)
	if err != nil {
		return err
	}
	return nil
}
