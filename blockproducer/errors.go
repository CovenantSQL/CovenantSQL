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

package blockproducer

import "errors"

var (
	// ErrInvalidDBPeersConfig defines database peers invalid error.
	ErrInvalidDBPeersConfig = errors.New("invalid database peers config")
	// ErrNoSuchDatabase defines database meta not exists error.
	ErrNoSuchDatabase = errors.New("no such database")
	// ErrDatabaseAllocation defines database allocation failure error.
	ErrDatabaseAllocation = errors.New("allocate database failed")
	// ErrMetricNotCollected defines errors collected.
	ErrMetricNotCollected = errors.New("metric not collected")

	// Errors on main chain

	// ErrCorruptedIndex defines index corrupted error.
	ErrCorruptedIndex = errors.New("corrupted index item")
	// ErrParentNotFound defines that the parent block cannot be found.
	ErrParentNotFound = errors.New("previous block cannot be found")
	// ErrInvalidHash defines invalid hash error.
	ErrInvalidHash = errors.New("Hash is invalid")
	// ErrExistedTx defines existed tx error.
	ErrExistedTx = errors.New("Tx existed")
	// ErrInvalidMerkleTreeRoot defines invalid merkle tree root error.
	ErrInvalidMerkleTreeRoot = errors.New("Block merkle tree root does not match the tx hashes")
	// ErrParentNotMatch defines invalid parent hash.
	ErrParentNotMatch = errors.New("Block's parent hash cannot match best block")
	// ErrNoSuchBlock defines no such block error.
	ErrNoSuchBlock = errors.New("Cannot find such block")
	// ErrNoSuchTxBilling defines no such txbilling error.
	ErrNoSuchTxBilling = errors.New("Cannot find such txbilling")
	// ErrSmallerSequenceID defines that new sequence id is smaller the old one.
	ErrSmallerSequenceID = errors.New("SequanceID should be bigger than the old one")
	// ErrInvalidBillingRequest defines BillingRequest is invalid
	ErrInvalidBillingRequest = errors.New("The BillingRequest is invalid")
	// ErrSignVerification indicates a failed signature verification.
	ErrSignVerification = errors.New("signature verification failed")

	// ErrBalanceOverflow indicates that there will be an overflow after balance manipulation.
	ErrBalanceOverflow = errors.New("balance overflow")
	// ErrInsufficientBalance indicates that an account has insufficient balance for spending.
	ErrInsufficientBalance = errors.New("insufficient balance")
	// ErrAccountNotFound indicates that an account is not found.
	ErrAccountNotFound = errors.New("account not found")
	// ErrAccountExists indicates that the an account already exists.
	ErrAccountExists = errors.New("account already exists")
	// ErrDatabaseNotFound indicates that a database is not found.
	ErrDatabaseNotFound = errors.New("database not found")
	// ErrDatabaseExists indicates that the database already exists.
	ErrDatabaseExists = errors.New("database already exists")
	// ErrDatabaseUserExists indicates that the database user already exists.
	ErrDatabaseUserExists = errors.New("database user already exists")
	// ErrInvalidAccountNonce indicates that a transaction has a invalid account nonce.
	ErrInvalidAccountNonce = errors.New("invalid account nonce")
	// ErrUnknownTransactionType indicates that a transaction has a unknown type and cannot be
	// further processed.
	ErrUnknownTransactionType = errors.New("unknown transaction type")
)
