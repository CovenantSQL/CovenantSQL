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

// Package secp256k1 signs and verifies ECDSA signatures on the secp256k1 curve with
// libsecp256k1.
//
// The libsecp256k1 directory holds COPYING, include/, src/*.h, src/secp256k1.c and
// src/precomputed_ecmult*.c of libsecp256k1 v0.8.0, unmodified. To update it, replace them with
// the same files of a newer release.
package secp256k1

/*
// Use the x86_64 assembly, as the upstream build does by default.
#cgo amd64 CFLAGS: -DUSE_ASM_X86_64=1
#include "libsecp256k1/src/secp256k1.c"
#include "libsecp256k1/src/precomputed_ecmult.c"
#include "libsecp256k1/src/precomputed_ecmult_gen.c"
*/
import "C"

import (
	"crypto/rand"
	"errors"
	"unsafe"
)

var (
	// ErrInvalidMsgLen is returned for a message that isn't 32 bytes long.
	ErrInvalidMsgLen = errors.New("invalid message length, need 32 bytes")
	// ErrInvalidKey is returned for a private key that isn't a 32-byte number in [1, N-1].
	ErrInvalidKey = errors.New("invalid private key")
)

var context *C.secp256k1_context

func init() {
	context = C.secp256k1_context_create(C.SECP256K1_CONTEXT_NONE)
	// libsecp256k1 recommends randomizing the context to protect signing from side channels.
	seed := make([]byte, 32)
	rand.Read(seed)
	if C.secp256k1_context_randomize(context, cBytes(seed)) != 1 {
		panic("secp256k1: could not randomize the context")
	}
}

func cBytes(b []byte) *C.uchar {
	return (*C.uchar)(unsafe.Pointer(&b[0]))
}

// Sign returns the ECDSA signature of the 32-byte msg by the 32-byte private key seckey, as R
// and S of 32 bytes each. The nonce follows RFC 6979, and S is at most half the group order.
func Sign(msg, seckey []byte) ([]byte, error) {
	if len(msg) != 32 {
		return nil, ErrInvalidMsgLen
	}
	if len(seckey) != 32 {
		return nil, ErrInvalidKey
	}
	var sig C.secp256k1_ecdsa_signature
	// With the default nonce function, which is RFC 6979, signing fails only for an invalid key.
	if C.secp256k1_ecdsa_sign(context, &sig, cBytes(msg), cBytes(seckey), nil, nil) != 1 {
		return nil, ErrInvalidKey
	}
	out := make([]byte, 64)
	C.secp256k1_ecdsa_signature_serialize_compact(context, cBytes(out), &sig)
	return out, nil
}

// VerifySignature reports whether signature, R and S of 32 bytes each, is a valid signature of
// the 32-byte msg by the serialized public key pubkey. It rejects an S above half the group
// order, which Sign never produces.
func VerifySignature(pubkey, msg, signature []byte) bool {
	if len(msg) != 32 || len(signature) != 64 || len(pubkey) == 0 {
		return false
	}
	var (
		sig C.secp256k1_ecdsa_signature
		key C.secp256k1_pubkey
	)
	return C.secp256k1_ecdsa_signature_parse_compact(context, &sig, cBytes(signature)) == 1 &&
		C.secp256k1_ec_pubkey_parse(context, &key, cBytes(pubkey), C.size_t(len(pubkey))) == 1 &&
		C.secp256k1_ecdsa_verify(context, &sig, cBytes(msg), &key) == 1
}
