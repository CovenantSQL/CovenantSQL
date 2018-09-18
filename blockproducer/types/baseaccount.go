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

type BaseAccount struct {
	Account
	AccountHash hash.Hash
	Signee      *asymmetric.PublicKey
	Signature   *asymmetric.Signature
}

// Serialize serializes TxBilling using msgpack.
func (b *BaseAccount) Serialize() (s []byte, err error) {
	var enc *bytes.Buffer
	if enc, err = utils.EncodeMsgPack(b); err != nil {
		return
	}
	s = enc.Bytes()
	return
}

// Deserialize desrializes TxBilling using msgpack.
func (b *BaseAccount) Deserialize(enc []byte) error {
	return utils.DecodeMsgPack(enc, b)
}

// GetAccountAddress implements interfaces/Transaction.GetAccountAddress.
func (b *BaseAccount) GetAccountAddress() proto.AccountAddress {
	return b.Address
}

// GetAccountNonce implements interfaces/Transaction.GetAccountNonce.
func (b *BaseAccount) GetAccountNonce() pi.AccountNonce {
	// BaseAccount nonce is not counted, always return 0.
	return pi.AccountNonce(0)
}

// GetHash implements interfaces/Transaction.GetHash.
func (b *BaseAccount) GetHash() hash.Hash {
	return b.AccountHash
}

// GetTransactionType implements interfaces/Transaction.GetTransactionType.
func (b *BaseAccount) GetTransactionType() pi.TransactionType {
	return pi.TransactionTypeBaseAccount
}

// Sign implements interfaces/Transaction.Sign.
func (b *BaseAccount) Sign(signer *asymmetric.PrivateKey) (err error) {
	var enc []byte
	if enc, err = b.Account.MarshalHash(); err != nil {
		return
	}
	var h = hash.THashH(enc)
	if b.Signature, err = signer.Sign(h[:]); err != nil {
		return
	}
	b.AccountHash = h
	b.Signee = signer.PubKey()
	return
}

// Verify implements interfaces/Transaction.Verify.
func (b *BaseAccount) Verify() (err error) {
	var enc []byte
	if enc, err = b.Account.MarshalHash(); err != nil {
		return
	} else if h := hash.THashH(enc); !b.AccountHash.IsEqual(&h) {
		err = ErrSignVerification
		return
	} else if !b.Signature.Verify(h[:], b.Signee) {
		err = ErrSignVerification
		return
	}
	return
}
