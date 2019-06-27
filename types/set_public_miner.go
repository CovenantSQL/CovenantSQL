/*
 * Copyright 2019 The CovenantSQL Authors.
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

// SetPublicMinerHeader defines the miner register transaction header.
type SetPublicMinerHeader struct {
	Miner   proto.AccountAddress
	Enabled uint8
	Nonce   pi.AccountNonce
}

// SetPublicMiner defines the miner register transaction.
type SetPublicMiner struct {
	SetPublicMinerHeader
	pi.TransactionTypeMixin
	verifier.DefaultHashSignVerifierImpl
}

// GetAccountNonce implements interfaces/Transaction.GetAccountNonce.
func (mr *SetPublicMinerHeader) GetAccountNonce() pi.AccountNonce {
	return mr.Nonce
}

// GetAccountAddress implements interfaces/Transaction.GetAccountAddress.
func (mr *SetPublicMiner) GetAccountAddress() proto.AccountAddress {
	addr, _ := crypto.PubKeyHash(mr.Signee)
	return addr
}

// Sign implements interfaces/Transaction.Sign.
func (mr *SetPublicMiner) Sign(signer *asymmetric.PrivateKey) error {
	return mr.DefaultHashSignVerifierImpl.Sign(&mr.SetPublicMinerHeader, signer)
}

// Verify implements interfaces/Transaction.Verify.
func (mr *SetPublicMiner) Verify() error {
	return mr.DefaultHashSignVerifierImpl.Verify(&mr.SetPublicMinerHeader)
}

// NewMinerRegister returns new instance.
func NewMinerRegister(header *SetPublicMinerHeader) *SetPublicMiner {
	return &SetPublicMiner{
		SetPublicMinerHeader: *header,
		TransactionTypeMixin: *pi.NewTransactionTypeMixin(pi.TransactionTypeSetPublicMiner),
	}
}

func init() {
	pi.RegisterTransaction(pi.TransactionTypeSetPublicMiner, (*SetPublicMiner)(nil))
}
