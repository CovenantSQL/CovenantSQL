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

//TODO(lambda): merge similar part of types.ProviderProfile

// ProvideServiceHeader define the miner providing service transaction header.
type ProvideServiceHeader struct {
	Space         uint64  // reserved storage space in bytes
	Memory        uint64  // reserved memory in bytes
	LoadAvgPerCPU float64 // max loadAvg15 per CPU
	TargetUser    []proto.AccountAddress
	GasPrice      uint64
	TokenType     TokenType
	NodeID        proto.NodeID
	Nonce         interfaces.AccountNonce
}

// GetAccountNonce implements interfaces/Transaction.GetAccountNonce.
func (h *ProvideServiceHeader) GetAccountNonce() interfaces.AccountNonce {
	return h.Nonce
}

// ProvideService define the miner providing service transaction.
type ProvideService struct {
	ProvideServiceHeader
	interfaces.TransactionTypeMixin
	verifier.DefaultHashSignVerifierImpl
}

// NewProvideService returns new instance.
func NewProvideService(h *ProvideServiceHeader) *ProvideService {
	return &ProvideService{
		ProvideServiceHeader: *h,
		TransactionTypeMixin: *interfaces.NewTransactionTypeMixin(interfaces.TransactionTypeProvideService),
	}
}

// Sign implements interfaces/Transaction.Sign.
func (ps *ProvideService) Sign(signer *asymmetric.PrivateKey) (err error) {
	return ps.DefaultHashSignVerifierImpl.Sign(&ps.ProvideServiceHeader, signer)
}

// Verify implements interfaces/Transaction.Verify.
func (ps *ProvideService) Verify() error {
	return ps.DefaultHashSignVerifierImpl.Verify(&ps.ProvideServiceHeader)
}

// GetAccountAddress implements interfaces/Transaction.GetAccountAddress.
func (ps *ProvideService) GetAccountAddress() proto.AccountAddress {
	addr, _ := crypto.PubKeyHash(ps.Signee)
	return addr
}

func init() {
	interfaces.RegisterTransaction(interfaces.TransactionTypeProvideService, (*ProvideService)(nil))
}
