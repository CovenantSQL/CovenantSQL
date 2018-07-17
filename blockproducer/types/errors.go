/*
 * Copyright 2018 The ThunderDB Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the “License”);
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an “AS IS” BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package types

import (
	"errors"
)

var (
	// ErrHashVerification indicates a failed hash verification.
	ErrHashVerification = errors.New("hash verification failed")

	// ErrSignVerification indicates a failed signature verification.
	ErrSignVerification = errors.New("signature verification failed")

	// ErrMerkleRootVerification indicates a failed merkle root verificatin.
	ErrMerkleRootVerification = errors.New("merkle root verification failed")

	// ErrNodePublicKeyNotMatch indicates that the public key given with a node does not match the
	// one in the key store.
	ErrNodePublicKeyNotMatch = errors.New("node publick key doesn't match")
)
