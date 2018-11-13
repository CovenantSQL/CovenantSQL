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
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/proto"
)

//go:generate hsp

// BaseAccount defines the base account type header.
type BaseAccount struct {
	Account
	pi.TransactionTypeMixin
}

// NewBaseAccount returns new instance.
func NewBaseAccount(account *Account) *BaseAccount {
	return &BaseAccount{
		Account:              *account,
		TransactionTypeMixin: *pi.NewTransactionTypeMixin(pi.TransactionTypeBaseAccount),
	}
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

// Hash implements interfaces/Transaction.Hash.
func (b *BaseAccount) Hash() (h hash.Hash) {
	return
}

// Sign implements interfaces/Transaction.Sign.
func (b *BaseAccount) Sign(signer *asymmetric.PrivateKey) (err error) {
	return
}

// Verify implements interfaces/Transaction.Verify.
func (b *BaseAccount) Verify() (err error) {
	return
}

func init() {
	pi.RegisterTransaction(pi.TransactionTypeBaseAccount, (*BaseAccount)(nil))
}
