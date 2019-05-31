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

package xenomint

import (
	"errors"
)

var (
	// ErrMissingParent indicates the parent of the current query attempt is missing.
	ErrMissingParent = errors.New("query missing parent")
	// ErrInvalidRequest indicates the query is invalid.
	ErrInvalidRequest = errors.New("invalid request")
	// ErrQueryConflict indicates the there is a conflict on query replay.
	ErrQueryConflict = errors.New("query conflict")
	// ErrMuxServiceNotFound indicates that the multiplexing service endpoint is not found.
	ErrMuxServiceNotFound = errors.New("mux service not found")
	// ErrStatefulQueryParts indicates query contains stateful query parts.
	ErrStatefulQueryParts = errors.New("query contains stateful query parts")
	// ErrInvalidTableName indicates query contains invalid table name in ddl statement.
	ErrInvalidTableName = errors.New("invalid table name in ddl")
)
