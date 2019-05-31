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

// UpdatePermissionHeader defines the updating sqlchain permission transaction header.
type UpdatePermissionHeader struct {
	TargetSQLChain proto.AccountAddress
	TargetUser     proto.AccountAddress
	Permission     *UserPermission
	Nonce          interfaces.AccountNonce
}

// GetAccountNonce implements interfaces/Transaction.GetAccountNonce.
func (u *UpdatePermissionHeader) GetAccountNonce() interfaces.AccountNonce {
	return u.Nonce
}

// UpdatePermission defines the updating sqlchain permission transaction.
type UpdatePermission struct {
	UpdatePermissionHeader
	interfaces.TransactionTypeMixin
	verifier.DefaultHashSignVerifierImpl
}

// NewUpdatePermission returns new instance.
func NewUpdatePermission(header *UpdatePermissionHeader) *UpdatePermission {
	return &UpdatePermission{
		UpdatePermissionHeader: *header,
		TransactionTypeMixin:   *interfaces.NewTransactionTypeMixin(interfaces.TransactionTypeUpdatePermission),
	}
}

// Sign implements interfaces/Transaction.Sign.
func (up *UpdatePermission) Sign(signer *asymmetric.PrivateKey) (err error) {
	return up.DefaultHashSignVerifierImpl.Sign(&up.UpdatePermissionHeader, signer)
}

// Verify implements interfaces/Transaction.Verify.
func (up *UpdatePermission) Verify() error {
	return up.DefaultHashSignVerifierImpl.Verify(&up.UpdatePermissionHeader)
}

// GetAccountAddress implements interfaces/Transaction.GetAccountAddress.
func (up *UpdatePermission) GetAccountAddress() proto.AccountAddress {
	addr, _ := crypto.PubKeyHash(up.Signee)
	return addr
}

func init() {
	interfaces.RegisterTransaction(interfaces.TransactionTypeUpdatePermission, (*UpdatePermission)(nil))
}
