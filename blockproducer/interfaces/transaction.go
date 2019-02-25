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

package interfaces

import (
	"encoding/binary"
	"time"

	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/proto"
)

//go:generate hsp

// AccountNonce defines the an account nonce.
type AccountNonce uint32

// TransactionType defines a transaction type.
type TransactionType uint32

// Bytes encodes a TransactionType to a byte slice.
func (t TransactionType) Bytes() (b []byte) {
	b = make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(t))
	return
}

// FromBytes decodes a TransactionType from a byte slice.
func FromBytes(b []byte) TransactionType {
	return TransactionType(binary.BigEndian.Uint32(b))
}

const (
	// TransactionTypeBilling defines billing transaction type.
	TransactionTypeBilling TransactionType = iota
	// TransactionTypeTransfer defines transfer transaction type.
	TransactionTypeTransfer
	// TransactionTypeCreateAccount defines account creation transaction type.
	TransactionTypeCreateAccount
	// TransactionTypeDeleteAccount defines account deletion transaction type.
	TransactionTypeDeleteAccount
	// TransactionTypeAddDatabaseUser defines database user addition transaction type.
	TransactionTypeAddDatabaseUser
	// TransactionTypeAlterDatabaseUser defines database user alteration transaction type.
	TransactionTypeAlterDatabaseUser
	// TransactionTypeDeleteDatabaseUser defines database user deletion transaction type.
	TransactionTypeDeleteDatabaseUser
	// TransactionTypeBaseAccount defines base account transaction type.
	TransactionTypeBaseAccount
	// TransactionTypeCreateDatabase defines database creation transaction type.
	TransactionTypeCreateDatabase
	// TransactionTypeProvideService defines miner providing database service type.
	TransactionTypeProvideService
	// TransactionTypeUpdatePermission defines admin user grant/revoke permission type.
	TransactionTypeUpdatePermission
	// TransactionTypeIssueKeys defines SQLChain owner assign symmetric key.
	TransactionTypeIssueKeys
	// TransactionTypeUpdateBilling defines SQLChain update billing information.
	TransactionTypeUpdateBilling
	// TransactionTypeNumber defines transaction types number.
	TransactionTypeNumber
)

func (t TransactionType) String() string {
	switch t {
	case TransactionTypeBilling:
		return "Billing"
	case TransactionTypeTransfer:
		return "Transfer"
	case TransactionTypeCreateAccount:
		return "CreateAccount"
	case TransactionTypeDeleteAccount:
		return "DeleteAccount"
	case TransactionTypeAddDatabaseUser:
		return "AddDatabaseUser"
	case TransactionTypeAlterDatabaseUser:
		return "AlterDatabaseUser"
	case TransactionTypeDeleteDatabaseUser:
		return "DeleteDatabaseUser"
	case TransactionTypeBaseAccount:
		return "BaseAccount"
	case TransactionTypeCreateDatabase:
		return "CreateDatabase"
	case TransactionTypeProvideService:
		return "ProvideService"
	case TransactionTypeUpdatePermission:
		return "UpdatePermission"
	case TransactionTypeIssueKeys:
		return "IssueKeys"
	case TransactionTypeUpdateBilling:
		return "UpdateBilling"
	default:
		return "Unknown"
	}
}

// TransactionState defines a transaction state.
type TransactionState uint32

// Transaction state transition:
// [o] ---[ Add ]--> Pending ---[ Produce Block ]--> Packed ---[ Irreversible ]--> Confirmed
//        |                     |                              x
//        |                     x                              +------[ Prune ]--> Not Found
//        x                     |
//        |                     +------------------------------------[ Expire ]--> Expired
//        |
//        +----------------------------------------------------------------------> Not Found.
const (
	TransactionStatePending TransactionState = iota
	TransactionStatePacked
	TransactionStateConfirmed
	TransactionStateExpired
	TransactionStateNotFound
)

func (s TransactionState) String() string {
	switch s {
	case TransactionStatePending:
		return "Pending"
	case TransactionStatePacked:
		return "Packed"
	case TransactionStateConfirmed:
		return "Confirmed"
	case TransactionStateExpired:
		return "Expired"
	case TransactionStateNotFound:
		return "Not Found"
	default:
		return "Unknown"
	}
}

// Transaction is the interface implemented by an object that can be verified and processed by
// block producers.
type Transaction interface {
	GetTransactionType() TransactionType
	GetAccountAddress() proto.AccountAddress
	GetAccountNonce() AccountNonce
	GetTimestamp() time.Time
	Hash() hash.Hash
	Sign(signer *asymmetric.PrivateKey) error
	Verify() error
	MarshalHash() ([]byte, error)
	Msgsize() int
}
