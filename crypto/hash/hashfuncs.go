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

package hash

import (
	"encoding/binary"
	"hash/fnv"

	// "crypto/sha256" benchmark is at least 10% faster on
	// i7-4870HQ CPU @ 2.50GHz than "github.com/minio/sha256-simd"
	"crypto/sha256"
	// "minio/blake2b-simd" benchmark is at least 3% faster on
	// i7-4870HQ CPU @ 2.50GHz than "golang.org/x/crypto/blake2b"
	// and supports more CPU instructions
	blake2b "github.com/minio/blake2b-simd"
)

// HashBSize is the size of HashB.
const HashBSize = sha256.Size

// HashSuite contains the hash length and the func handler.
type HashSuite struct {
	HashLen  int
	HashFunc func([]byte) []byte
}

// HashB calculates hash(b) and returns the resulting bytes.
func HashB(b []byte) []byte {
	hash := sha256.Sum256(b)
	return hash[:]
}

// HashH calculates hash(b) and returns the resulting bytes as a Hash.
func HashH(b []byte) Hash {
	return Hash(sha256.Sum256(b))
}

// FNVHash32B calculates hash(b) into [0, 2^64] and returns the resulting bytes.
func FNVHash32B(b []byte) []byte {
	hash := fnv.New32()
	hash.Write(b)
	return hash.Sum(nil)
}

// FNVHash32uint return the uint32 value of fnv hash 32 of b.
func FNVHash32uint(b []byte) uint32 {
	return binary.BigEndian.Uint32(FNVHash32B(b))
}

// DoubleHashB calculates hash(hash(b)) and returns the resulting bytes.
func DoubleHashB(b []byte) []byte {
	first := sha256.Sum256(b)
	second := sha256.Sum256(first[:])
	return second[:]
}

// DoubleHashH calculates hash(hash(b)) and returns the resulting bytes as a
// Hash.
func DoubleHashH(b []byte) Hash {
	first := sha256.Sum256(b)
	return Hash(sha256.Sum256(first[:]))
}

// THashB is a combination of blake2b-512 and SHA256
//  The cryptographic hash function BLAKE2 is an improved version of the
// SHA-3 finalist BLAKE.
func THashB(b []byte) []byte {
	first := blake2b.Sum512(b)
	second := sha256.Sum256(first[:])
	return second[:]
}

// THashH calculates sha256(blake2b-512(b)) and returns the resulting bytes as a
// Hash.
func THashH(b []byte) Hash {
	first := blake2b.Sum512(b)
	return Hash(sha256.Sum256(first[:]))
}
