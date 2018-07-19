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
	"gitlab.com/thunderdb/ThunderDB/proto"
	"gitlab.com/thunderdb/ThunderDB/utils"
)

// BillingRequestHeader includes contents that need to be signed
type BillingRequestHeader struct {
	DatabaseID  proto.DatabaseID
	BlockHash   hash.Hash
	BlockHeight int32
	GasAmounts  []*proto.AddrAndGas
}

// MarshalBinary implements BinaryMarshaler.
func (bh *BillingRequestHeader) MarshalBinary() ([]byte, error) {
	buffer := bytes.NewBuffer(nil)

	err := utils.WriteElements(buffer, binary.BigEndian,
		&bh.DatabaseID,
		&bh.BlockHash,
		&bh.BlockHeight,
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
		&bh.BlockHash,
		&bh.BlockHeight,
		&bh.GasAmounts,
	)
	if err != nil {
		return err
	}
	return nil
}

// PackAndSignRequestHeader first computes the hash of BillingRequestHeader, then signs the request
func (bh *BillingRequestHeader) PackAndSignRequestHeader(signee *asymmetric.PrivateKey) (*hash.Hash,
	*asymmetric.Signature,
	error) {
	b, err := bh.MarshalBinary()
	if err != nil {
		return nil, nil, err
	}

	h := hash.THashH(b)
	signature, err := signee.Sign(h[:])
	if err != nil {
		return nil, nil, err
	}
	return &h, signature, nil
}

// BillingRequest defines periodically Billing sync
type BillingRequest struct {
	Header      BillingRequestHeader
	RequestHash hash.Hash
	Signees     []*asymmetric.PublicKey
	Signatures  []*asymmetric.Signature
}

// BillingResponse definies the the response for BillingRequest
type BillingResponse struct {
	RequestHash hash.Hash
	Signee      *asymmetric.PublicKey
	Signature   *asymmetric.Signature
}
