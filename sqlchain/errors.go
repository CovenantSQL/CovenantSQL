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

package sqlchain

import (
	"errors"
)

var (
	// ErrFieldConversion indicates an incompatible field conversion during data unmarshalling from
	// a protobuf message, such as converting a non-digital string to a BigInt.
	ErrFieldConversion = errors.New("incompatible field type conversion")

	// ErrFieldLength indicates an unexpected array field length during data unmarshalling from a
	// protobuf message. Since protobuf doesn't have fixed size array, we need to check the slice
	// length before read it back from a slice field to a fixed size array field.
	ErrFieldLength = errors.New("unexpected slice field length")

	// ErrHashVerification indicates a failed hash verification.
	ErrHashVerification = errors.New("hash verification failed")

	// ErrSignVerification indicates a failed signature verification.
	ErrSignVerification = errors.New("signature verification failed")

	// ErrNodePublicKeyNotMatch indicates that the public key given with a node does not match the
	// one in the key store.
	ErrNodePublicKeyNotMatch = errors.New("node publick key doesn't match")

	// ErrNilValue indicates that an unexpected but not fatal nil value is detected , hence return
	// it as an error.
	ErrNilValue = errors.New("unexpected nil value")

	// ErrParentNotFound indicates an error failing to find parent node during a chain reloading.
	ErrParentNotFound = errors.New("could not find parent node")

	// ErrBlockExists indicates that a received block is already in the indexed.
	ErrBlockExists = errors.New("block already exists")

	// ErrInvalidBlock indicates an invalid block which does not extend the best chain while
	// pushing new blocks.
	ErrInvalidBlock = errors.New("invalid block")

	// ErrBlockTimestampOutOfPeriod indicates a block producing timestamp verification failure.
	ErrBlockTimestampOutOfPeriod = errors.New("block timestamp is out of producing period")

	// ErrQueryExists indicates that a query already exists in memory index during adding.
	ErrQueryExists = errors.New("query already exists in index")
)
