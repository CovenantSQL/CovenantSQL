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

package main

import (
	cw "github.com/CovenantSQL/CovenantSQL/cmd/cql-secure-gateway/casbin"
	"github.com/pkg/errors"
)

var (
	// ErrInvalidConfig defines error for invalid SG config.
	ErrInvalidConfig = errors.New("invalid config")
	// ErrFieldEncryption defines invalid field encryption config (usually assigned multiple encryption keys to single column).
	ErrFieldEncryption = errors.New("invalid field encryption config")
	// ErrInvalidField defines invalid field format.
	ErrInvalidField = cw.ErrInvalidField
	// ErrUnauthorizedQuery defines error for unauthorized query.
	ErrUnauthorizedQuery = errors.New("unauthorized query")
	// ErrUnsupportedEncryptionFieldQuery defines error for unsupported encryption field query.
	ErrUnsupportedEncryptionFieldQuery = errors.New("unsupported encryption field query")
	// ErrNilPointer defines error for nil output.
	ErrNilPointer = errors.New("destination pointer is nil")
	// ErrNotPointer defines error for scanning to non pointer type.
	ErrNotPointer = errors.New("destination not a pointer")
	// ErrUnsupportedType defines invalid type to encrypt/decrypt.
	ErrUnsupportedType = errors.New("unsupported type for encryption")
)
