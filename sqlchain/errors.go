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
	// ErrParentNotFound indicates an error failing to find parent node during a chain reloading.
	ErrParentNotFound = errors.New("could not find parent node")
	// ErrInvalidBlock indicates an invalid block which does not extend the best chain while
	// pushing new blocks.
	ErrInvalidBlock = errors.New("invalid block")
	// ErrMultipleAckOfSeqNo indicates that multiple acknowledgements for a same sequence number is
	// detected.
	ErrMultipleAckOfSeqNo = errors.New("multiple acknowledgements of same sequence number")
	// ErrUnknownMuxRequest indicates that the multiplexing request endpoint is not found.
	ErrUnknownMuxRequest = errors.New("unknown multiplexing request")
	// ErrQueryExpired indicates that a received query Response/Ack has expired.
	ErrQueryExpired = errors.New("query has expired")
	// ErrUnknownProducer indicates that the block has an unknown producer.
	ErrUnknownProducer = errors.New("unknown block producer")
	// ErrInvalidProducer indicates that the block has an invalid producer.
	ErrInvalidProducer = errors.New("invalid block producer")
	// ErrQueryNotFound indicates that a query is not found in the index.
	ErrQueryNotFound = errors.New("query not found")
	// ErrResponseSeqNotMatch indicates that a response sequence id doesn't match the original one
	// in the index.
	ErrResponseSeqNotMatch = errors.New("response sequence id doesn't match")
)
