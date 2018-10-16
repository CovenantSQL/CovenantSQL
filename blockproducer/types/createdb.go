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
	"github.com/CovenantSQL/CovenantSQL/proto"
)

//go:generate hsp

// CreateDatabaseHeader defines the database creation transaction header.
type CreateDatabaseHeader struct {
	Owner proto.AccountAddress
	Nonce pi.AccountNonce
}

// GetAccountAddress implements interfaces/Transaction.GetAccountAddress.
func (h *CreateDatabaseHeader) GetAccountAddress() proto.AccountAddress {
	return h.Owner
}

// GetAccountNonce implements interfaces/Transaction.GetAccountNonce.
func (h *CreateDatabaseHeader) GetAccountNonce() pi.AccountNonce {
	return h.Nonce
}

// CreateDatabase defines the database creation transaction.
type CreateDatabase struct {
	CreateDatabaseHeader
	DefaultHashSignVerifierImpl
}

// Serialize implements interfaces/Transaction.Serialize.
func (cd *CreateDatabase) Serialize() ([]byte, error) {
	return serialize(cd)
}

// Deserialize implements interfaces/Transaction.Deserialize.
func (cd *CreateDatabase) Deserialize(enc []byte) error {
	return deserialize(enc, cd)
}

// GetTransactionType implements interfaces/Transaction.GetTransactionType.
func (cd *CreateDatabase) GetTransactionType() pi.TransactionType {
	return pi.TransactionTypeCreataDatabase
}

// Sign implements interfaces/Transaction.Sign.
func (cd *CreateDatabase) Sign(signer *asymmetric.PrivateKey) (err error) {
	return cd.DefaultHashSignVerifierImpl.Sign(&cd.CreateDatabaseHeader, signer)
}

// Verify implements interfaces/Transaction.Verify.
func (cd *CreateDatabase) Verify() error {
	return cd.DefaultHashSignVerifierImpl.Verify(&cd.CreateDatabaseHeader)
}
