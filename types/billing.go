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
	pi "github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/verifier"
	"github.com/CovenantSQL/CovenantSQL/proto"
)

//go:generate hsp

// BillingHeader defines the customer's billing and block rewards in transaction.
type BillingHeader struct {
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

// NewBillingHeader generates new BillingHeader.
func NewBillingHeader(nonce pi.AccountNonce, bReq *BillingRequest, producer proto.AccountAddress, receivers []*proto.AccountAddress,
	fees []uint64, rewards []uint64) *BillingHeader {
	return &BillingHeader{
		Nonce:          nonce,
		BillingRequest: *bReq,
		Producer:       producer,
		Receivers:      receivers,
		Fees:           fees,
		Rewards:        rewards,
	}
}

// Billing is a type of tx, that is used to record sql chain billing and block rewards.
type Billing struct {
	BillingHeader
	pi.TransactionTypeMixin
	verifier.DefaultHashSignVerifierImpl
}

// NewBilling generates a new Billing.
func NewBilling(header *BillingHeader) *Billing {
	return &Billing{
		BillingHeader:        *header,
		TransactionTypeMixin: *pi.NewTransactionTypeMixin(pi.TransactionTypeBilling),
	}
}

// Sign implements interfaces/Transaction.Sign.
func (tb *Billing) Sign(signer *asymmetric.PrivateKey) (err error) {
	return tb.DefaultHashSignVerifierImpl.Sign(&tb.BillingHeader, signer)
}

// Verify implements interfaces/Transaction.Verify.
func (tb *Billing) Verify() error {
	return tb.DefaultHashSignVerifierImpl.Verify(&tb.BillingHeader)
}

// GetAccountAddress implements interfaces/Transaction.GetAccountAddress.
func (tb *Billing) GetAccountAddress() proto.AccountAddress {
	return tb.Producer
}

// GetAccountNonce implements interfaces/Transaction.GetAccountNonce.
func (tb *Billing) GetAccountNonce() pi.AccountNonce {
	return tb.Nonce
}

// GetDatabaseID gets the database ID.
func (tb *Billing) GetDatabaseID() proto.DatabaseID {
	return tb.BillingRequest.Header.DatabaseID
}

func init() {
	pi.RegisterTransaction(pi.TransactionTypeBilling, (*Billing)(nil))
}
