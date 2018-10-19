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

// TxContent defines the customer's billing and block rewards in transaction.
type TxContent struct {
	// Transaction nonce
	Nonce          pi.AccountNonce
	BillingRequest BillingRequest
	// Bill producer
	Producer proto.AccountAddress
	// Bill receivers
	Receivers []*proto.AccountAddress
	// Fee paid by stable coin
	Fees []uint64
	// Reward is share coin
	Rewards []uint64
}

// NewTxContent generates new TxContent.
func NewTxContent(seqID pi.AccountNonce, bReq *BillingRequest, producer proto.AccountAddress, receivers []*proto.AccountAddress,
	fees []uint64, rewards []uint64) *TxContent {
	return &TxContent{
		Nonce:          seqID,
		BillingRequest: *bReq,
		Producer:       producer,
		Receivers:      receivers,
		Fees:           fees,
		Rewards:        rewards,
	}
}

// GetHash returns the hash of transaction.
func (tb *TxContent) GetHash() (*hash.Hash, error) {
	b, err := tb.MarshalHash()
	if err != nil {
		return nil, err
	}
	h := hash.THashH(b)
	return &h, nil
}

// TxBilling is a type of tx, that is used to record sql chain billing and block rewards.
type TxBilling struct {
	TxContent TxContent
	TxHash    *hash.Hash
	Signee    *asymmetric.PublicKey
	Signature *asymmetric.Signature
}

// NewTxBilling generates a new TxBilling.
func NewTxBilling(txContent *TxContent) *TxBilling {
	return &TxBilling{
		TxContent: *txContent,
	}
}

// Serialize serializes TxBilling using msgpack.
func (tb *TxBilling) Serialize() (b []byte, err error) {
	var enc *bytes.Buffer
	if enc, err = utils.EncodeMsgPack(tb); err != nil {
		return
	}
	b = enc.Bytes()
	return
}

// Deserialize desrializes TxBilling using msgpack.
func (tb *TxBilling) Deserialize(enc []byte) error {
	return utils.DecodeMsgPack(enc, tb)
}

// GetAccountAddress implements interfaces/Transaction.GetAccountAddress.
func (tb *TxBilling) GetAccountAddress() proto.AccountAddress {
	return tb.TxContent.Producer
}

// GetAccountNonce implements interfaces/Transaction.GetAccountNonce.
func (tb *TxBilling) GetAccountNonce() pi.AccountNonce {
	return tb.TxContent.Nonce
}

// GetHash implements interfaces/Transaction.GetHash.
func (tb *TxBilling) GetHash() hash.Hash {
	return *tb.TxHash
}

// GetTransactionType implements interfaces/Transaction.GetTransactionType.
func (tb *TxBilling) GetTransactionType() pi.TransactionType {
	return pi.TransactionTypeBilling
}

// Sign computes tx of TxContent and signs it.
func (tb *TxBilling) Sign(signer *asymmetric.PrivateKey) error {
	enc, err := tb.TxContent.MarshalHash()
	if err != nil {
		return err
	}
	h := hash.THashH(enc)
	tb.TxHash = &h
	tb.Signee = signer.PubKey()

	signature, err := signer.Sign(h[:])
	if err != nil {
		return err
	}
	tb.Signature = signature

	return nil
}

// Verify verifies the signature of TxBilling.
func (tb *TxBilling) Verify() (err error) {
	var enc []byte
	if enc, err = tb.TxContent.MarshalHash(); err != nil {
		return
	} else if h := hash.THashH(enc); !tb.TxHash.IsEqual(&h) {
		err = ErrSignVerification
		return
	} else if !tb.Signature.Verify(h[:], tb.Signee) {
		err = ErrSignVerification
		return
	}
	return
}

// GetDatabaseID gets the database ID.
func (tb *TxBilling) GetDatabaseID() *proto.DatabaseID {
	return &tb.TxContent.BillingRequest.Header.DatabaseID
}

func init() {
	pi.RegisterTransaction(pi.TransactionTypeBilling, (*TxBilling)(nil))
}
