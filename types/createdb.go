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
	"github.com/CovenantSQL/CovenantSQL/crypto"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/verifier"
	"github.com/CovenantSQL/CovenantSQL/proto"
)

//go:generate hsp

// CreateDatabaseHeader defines the database creation transaction header.
type CreateDatabaseHeader struct {
	Owner          proto.AccountAddress
	ResourceMeta   ResourceMeta
	GasPrice       uint64
	AdvancePayment uint64
	TokenType      TokenType
	Nonce          pi.AccountNonce
}

// GetAccountNonce implements interfaces/Transaction.GetAccountNonce.
func (h *CreateDatabaseHeader) GetAccountNonce() pi.AccountNonce {
	return h.Nonce
}

// CreateDatabase defines the database creation transaction.
type CreateDatabase struct {
	CreateDatabaseHeader
	pi.TransactionTypeMixin
	verifier.DefaultHashSignVerifierImpl
}

// NewCreateDatabase returns new instance.
func NewCreateDatabase(header *CreateDatabaseHeader) *CreateDatabase {
	return &CreateDatabase{
		CreateDatabaseHeader: *header,
		TransactionTypeMixin: *pi.NewTransactionTypeMixin(pi.TransactionTypeCreateDatabase),
	}
}

// Sign implements interfaces/Transaction.Sign.
func (cd *CreateDatabase) Sign(signer *asymmetric.PrivateKey) (err error) {
	return cd.DefaultHashSignVerifierImpl.Sign(&cd.CreateDatabaseHeader, signer)
}

// Verify implements interfaces/Transaction.Verify.
func (cd *CreateDatabase) Verify() error {
	return cd.DefaultHashSignVerifierImpl.Verify(&cd.CreateDatabaseHeader)
}

// GetAccountAddress implements interfaces/Transaction.GetAccountAddress.
func (cd *CreateDatabase) GetAccountAddress() proto.AccountAddress {
	addr, _ := crypto.PubKeyHash(cd.Signee)
	return addr
}

func init() {
	pi.RegisterTransaction(pi.TransactionTypeCreateDatabase, (*CreateDatabase)(nil))
}
