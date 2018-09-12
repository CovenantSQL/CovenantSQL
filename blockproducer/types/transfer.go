/*
 * Copyright 2018 The CovenantSQL Authors.
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

	pi "github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils"
)

//go:generate hsp

// TransferHeader defines the transfer transaction header.
type TransferHeader struct {
	Sender, Receiver proto.AccountAddress
	Nonce            pi.AccountNonce
	Amount           uint64
}

// Transfer defines the transfer transaction.
type Transfer struct {
	TransferHeader
	HeaderHash hash.Hash
	Signee     *asymmetric.PublicKey
	Signature  *asymmetric.Signature
}

// Serialize serializes TxBilling using msgpack.
func (t *Transfer) Serialize() (b []byte, err error) {
	var enc *bytes.Buffer
	if enc, err = utils.EncodeMsgPack(t); err != nil {
		b = enc.Bytes()
	}
	return
}

// Deserialize desrializes TxBilling using msgpack.
func (t *Transfer) Deserialize(enc []byte) error {
	return utils.DecodeMsgPack(enc, t)
}

// GetAccountAddress implements interfaces/Transaction.GetAccountAddress.
func (t *Transfer) GetAccountAddress() proto.AccountAddress {
	return t.Sender
}

// GetAccountNonce implements interfaces/Transaction.GetAccountNonce.
func (t *Transfer) GetAccountNonce() pi.AccountNonce {
	return t.Nonce
}

// GetHash implements interfaces/Transaction.GetHash.
func (t *Transfer) GetHash() hash.Hash {
	return t.HeaderHash
}

// GetTransactionType implements interfaces/Transaction.GetTransactionType.
func (t *Transfer) GetTransactionType() pi.TransactionType {
	return pi.TransactionTypeTransfer
}

// Verify verifies the signature of TxBilling.
func (t *Transfer) Verify() (err error) {
	var enc []byte
	if enc, err = t.TransferHeader.MarshalHash(); err != nil {
		return
	} else if h := hash.THashH(enc); !t.HeaderHash.IsEqual(&h) {
		return
	} else if !t.Signature.Verify(h[:], t.Signee) {
		return
	}
	return
}
