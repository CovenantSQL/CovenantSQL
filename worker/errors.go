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

package worker

import "errors"

var (
	// ErrInvalidRequest defines invalid request structure during request.
	ErrInvalidRequest = errors.New("invalid request supplied")
	// ErrInvalidRequestSeq defines invalid sequence no of request.
	ErrInvalidRequestSeq = errors.New("invalid request sequence applied")
	// ErrAlreadyExists defines error on re-creating existing database instance.
	ErrAlreadyExists = errors.New("database instance already exists")
	// ErrNotExists defines errors on manipulating a non-exists database instance.
	ErrNotExists = errors.New("database instance not exists")
	// ErrInvalidDBConfig defines errors on received invalid db config from block producer.
	ErrInvalidDBConfig = errors.New("invalid database configuration")
	// ErrSpaceLimitExceeded defines errors on disk space exceeding limit.
	ErrSpaceLimitExceeded = errors.New("space limit exceeded")
	// ErrUnknownMuxRequest indicates that the a multiplexing request endpoint is not found.
	ErrUnknownMuxRequest = errors.New("unknown multiplexing request")
	// ErrPermissionDeny indicates that the requester has no permission to send read or write query.
	ErrPermissionDeny = errors.New("permission deny")
	// ErrInvalidPermission indicates that the requester sends a unrecognized permission.
	ErrInvalidPermission = errors.New("invalid permission")
	// ErrInvalidTransactionType indicates that the transaction type is invalid.
	ErrInvalidTransactionType = errors.New("invalid transaction type")
)
