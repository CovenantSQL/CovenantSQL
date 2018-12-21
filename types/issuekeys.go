/*
 *  Copyright 2018 The CovenantSQL Authors.
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
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
	"github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	"github.com/CovenantSQL/CovenantSQL/crypto"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/verifier"
	"github.com/CovenantSQL/CovenantSQL/proto"
)

//go:generate hsp

// MinerKey defines an encryption key associated with miner address.
type MinerKey struct {
	Miner         proto.AccountAddress
	EncryptionKey string
}

// IssueKeysHeader defines an encryption key header.
type IssueKeysHeader struct {
	TargetSQLChain proto.AccountAddress
	MinerKeys      []MinerKey
	Nonce          interfaces.AccountNonce
}

// GetAccountNonce implements interfaces/Transaction.GetAccountNonce.
func (h *IssueKeysHeader) GetAccountNonce() interfaces.AccountNonce {
	return h.Nonce
}

// IssueKeys defines the database creation transaction.
type IssueKeys struct {
	IssueKeysHeader
	interfaces.TransactionTypeMixin
	verifier.DefaultHashSignVerifierImpl
}

// NewIssueKeys returns new instance.
func NewIssueKeys(header *IssueKeysHeader) *IssueKeys {
	return &IssueKeys{
		IssueKeysHeader:      *header,
		TransactionTypeMixin: *interfaces.NewTransactionTypeMixin(interfaces.TransactionTypeIssueKeys),
	}
}

// Sign implements interfaces/Transaction.Sign.
func (ik *IssueKeys) Sign(signer *asymmetric.PrivateKey) (err error) {
	return ik.DefaultHashSignVerifierImpl.Sign(&ik.IssueKeysHeader, signer)
}

// Verify implements interfaces/Transaction.Verify.
func (ik *IssueKeys) Verify() error {
	return ik.DefaultHashSignVerifierImpl.Verify(&ik.IssueKeysHeader)
}

// GetAccountAddress implements interfaces/Transaction.GetAccountAddress.
func (ik *IssueKeys) GetAccountAddress() proto.AccountAddress {
	addr, _ := crypto.PubKeyHash(ik.Signee)
	return addr
}

func init() {
	interfaces.RegisterTransaction(interfaces.TransactionTypeIssueKeys, (*IssueKeys)(nil))
}
