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
	// ErrNoSuchDatabase defines database meta not exists error.
	ErrNoSuchDatabase = errors.New("no such database")
	// ErrParentNotFound defines that the parent block cannot be found.
	ErrParentNotFound = errors.New("previous block cannot be found")
	// ErrInvalidHash defines invalid hash error.
	ErrInvalidHash = errors.New("Hash is invalid")
	// ErrExistedTx defines existed tx error.
	ErrExistedTx = errors.New("Tx existed")
	// ErrParentNotMatch defines invalid parent hash.
	ErrParentNotMatch = errors.New("Block's parent hash cannot match best block")
	// ErrTooManyTransactionsInBlock defines error of too many transactions in a block.
	ErrTooManyTransactionsInBlock = errors.New("too many transactions in block")
	// ErrBalanceOverflow indicates that there will be an overflow after balance manipulation.
	ErrBalanceOverflow = errors.New("balance overflow")
	// ErrInsufficientBalance indicates that an account has insufficient balance for spending.
	ErrInsufficientBalance = errors.New("insufficient balance")
	// ErrInsufficientTransfer indicates that the transfer amount is insufficient for paying arrears.
	ErrInsufficientTransfer = errors.New("insufficient transfer")
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
	// ErrInvalidSender indicates that tx.Signee != tx.Sender.
	ErrInvalidSender = errors.New("invalid sender")
	// ErrInvalidRange indicates that the billing range is invalid.
	ErrInvalidRange = errors.New("invalid billing range")
	// ErrNoSuchMiner indicates that this miner does not exist or register.
	ErrNoSuchMiner = errors.New("no such miner")
	// ErrNoEnoughMiner indicates that there is not enough miners
	ErrNoEnoughMiner = errors.New("can not get enough miners")
	// ErrAccountPermissionDeny indicates that the sender does not own admin permission to the sqlchain.
	ErrAccountPermissionDeny = errors.New("account permission deny")
	// ErrNoSuperUserLeft indicates there is no super user in sqlchain.
	ErrNoSuperUserLeft = errors.New("no super user left")
	// ErrInvalidPermission indicates that the permission is invalid.
	ErrInvalidPermission = errors.New("invalid permission")
	// ErrMinerUserNotMatch indicates that the miner and user do not match.
	ErrMinerUserNotMatch = errors.New("miner and user do not match")
	// ErrInsufficientAdvancePayment indicates that the advance payment is insufficient.
	ErrInsufficientAdvancePayment = errors.New("insufficient advance payment")
	// ErrNilGenesis indicates that the genesis block is nil in config.
	ErrNilGenesis = errors.New("nil genesis block")
	// ErrMultipleGenesis indicates that there're multiple genesis blocks while loading.
	ErrMultipleGenesis = errors.New("multiple genesis blocks")
	// ErrGenesisHashNotMatch indicates that the genesis block hash in config doesn't match
	// the persisted one.
	ErrGenesisHashNotMatch = errors.New("persisted genesis block hash not match")
	// ErrInvalidGasPrice indicates that the gas price is invalid.
	ErrInvalidGasPrice = errors.New("gas price is invalid")
	// ErrInvalidMinerCount indicates that the miner node count is invalid.
	ErrInvalidMinerCount = errors.New("miner node count is invalid")
	// ErrLocalNodeNotFound indicates that the local node id is not found in the given peer list.
	ErrLocalNodeNotFound = errors.New("local node id not found in peer list")
	// ErrNoAvailableBranch indicates that there is no available branch from the state storage.
	ErrNoAvailableBranch = errors.New("no available branch from state storage")
	// ErrWrongTokenType indicates that token type in transfer is wrong.
	ErrWrongTokenType = errors.New("wrong token type")
)
