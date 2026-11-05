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

// Package linktest builds the libsecp256k1 of crypto/secp256k1 without renaming its symbols, as
// another copy of libsecp256k1 in the same program would be, such as the one in go-ethereum's
// crypto/secp256k1. A program that uses both packages fails to link if crypto/secp256k1 leaves a
// global symbol of libsecp256k1 with its original name.
package linktest

/*
#include "../../libsecp256k1/src/secp256k1.c"
#include "../../libsecp256k1/src/precomputed_ecmult.c"
#include "../../libsecp256k1/src/precomputed_ecmult_gen.c"
*/
import "C"

// Selftest runs the self-test of this copy of libsecp256k1, which aborts the program if it fails.
func Selftest() {
	C.secp256k1_selftest()
}
