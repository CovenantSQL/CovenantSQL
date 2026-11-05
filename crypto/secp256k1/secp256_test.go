/*
 * Copyright 2026 The CovenantSQL Authors.
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

package secp256k1

import (
	"testing"

	"github.com/CovenantSQL/CovenantSQL/crypto/secp256k1/internal/linktest"
)

// TestLink links this package together with linktest, a copy of libsecp256k1 with the original
// symbol names. The test binary fails to link if prefix.h misses a global symbol.
func TestLink(t *testing.T) {
	linktest.Selftest()
	msg := []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	key := []byte("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	if _, err := Sign(msg, key); err != nil {
		t.Fatalf("could not sign: %v", err)
	}
}
