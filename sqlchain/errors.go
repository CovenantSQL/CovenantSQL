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

	// ErrMultipleAckOfResponse indicates that multiple acknowledgements for a same query response
	// is detected.
	ErrMultipleAckOfResponse = errors.New("multiple acknowledgements of same response")

	// ErrMultipleAckOfSeqNo indicates that multiple acknowledgements for a same sequence number is
	// detected.
	ErrMultipleAckOfSeqNo = errors.New("multiple acknowledgements of same sequence number")

	// ErrQueryExpired indicates that a received query Response/Ack has expired.
	ErrQueryExpired = errors.New("query has expired")

	// ErrQueryNotCached indicates that a wanted query is not cached locally.
	ErrQueryNotCached = errors.New("query is not cached locally")

	// ErrQuerySignedByAnotherBlock indicates that a query is already signed in a known block
	// during the verification of a new introduced block.
	ErrQuerySignedByAnotherBlock = errors.New("query has been packed by another block")

	// ErrCorruptedIndex indicates that a corrupted index item is detected.
	ErrCorruptedIndex = errors.New("corrupted index item")

	// ErrUnknownMuxRequest indicates that the multiplexing request endpoint is not found.
	ErrUnknownMuxRequest = errors.New("unknown multiplexing request")

	// ErrUnknownProducer indicates that the block has an unknown producer.
	ErrUnknownProducer = errors.New("unknown block producer")

	// ErrInvalidProducer indicates that the block has an invalid producer.
	ErrInvalidProducer = errors.New("invalid block producer")

	// ErrUnavailableBillingRang indicates that the billing range is not available now.
	ErrUnavailableBillingRang = errors.New("unavailable billing range")

	// ErrHashNotMatch indicates that a message hash value doesn't match the original hash value
	// given in its hash field.
	ErrHashNotMatch = errors.New("hash value doesn't match")

	// ErrMetaStateNotFound indicates that meta state not found in db.
	ErrMetaStateNotFound = errors.New("meta state not found in db")

	// ErrAckQueryNotFound indicates that an acknowledged query record is not found.
	ErrAckQueryNotFound = errors.New("acknowledged query not found")

	// ErrQueryNotFound indicates that a query is not found in the index.
	ErrQueryNotFound = errors.New("query not found")

	// ErrInvalidRequest indicates the query is invalid.
	ErrInvalidRequest = errors.New("invalid request")

	// ErrResponseSeqNotMatch indicates that a response sequence id doesn't match the original one
	// in the index.
	ErrResponseSeqNotMatch = errors.New("response sequence id doesn't match")
)
