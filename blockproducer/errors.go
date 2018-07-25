/*
 * Copyright 2018 The ThunderDB Authors.
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
)
