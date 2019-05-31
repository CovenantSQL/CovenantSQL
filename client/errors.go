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

package client

import "github.com/pkg/errors"

// Various errors the driver might returns.
var (
	// ErrQueryInTransaction represents a read query is presented during user transaction.
	ErrQueryInTransaction = errors.New("only write is supported during transaction")
	// ErrNotInitialized represents the driver is not initialized yet.
	ErrNotInitialized = errors.New("driver not initialized")
	// ErrAlreadyInitialized represents the driver is already initialized.
	ErrAlreadyInitialized = errors.New("driver already initialized")
	// ErrInvalidRequestSeq defines invalid sequence no of request.
	ErrInvalidRequestSeq = errors.New("invalid request sequence applied")
	// ErrInvalidProfile indicates the SQLChain profile is invalid.
	ErrInvalidProfile = errors.New("invalid sqlchain profile")
	// ErrNoSuchTokenBalance indicates no such token balance in chain.
	ErrNoSuchTokenBalance = errors.New("no such token balance")
)
