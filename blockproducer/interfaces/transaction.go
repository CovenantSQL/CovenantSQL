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

	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/proto"
)

//go:generate hsp

type AccountNonce uint32
type TransactionType uint32

func (t TransactionType) Bytes() (b []byte) {
	b = make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(t))
	return
}

func FromBytes(b []byte) TransactionType {
	return TransactionType(binary.BigEndian.Uint32(b))
}

const (
	TransactionTypeBilling TransactionType = iota + 1
	TransactionTypeTransfer
	TransactionTypeCreateAccount
	TransactionTypeDeleteAccount
	TransactionTypeAddDatabaseUser
	TransactionTypeAlterDatabaseUser
	TransactionTypeDeleteDatabaseUser
	TransactionTypeNumber
)

// Serializer is the interface implemented by an object that can serialize itself into binary form.
type Serializer interface {
	Serialize() ([]byte, error)
}

// Deserializer is the interface implemented by an object that can deserialize a binary
// representation of itself.
type Deserializer interface {
	Deserialize(enc []byte) error
}

// Transaction is the interface implemented by an object that can be verified and processed by
// block producers.
type Transaction interface {
	Serializer
	Deserializer
	GetAccountAddress() proto.AccountAddress
	GetAccountNonce() AccountNonce
	GetHash() hash.Hash
	GetTransactionType() TransactionType
	Verify() error
}
