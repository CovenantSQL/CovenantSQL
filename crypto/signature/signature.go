/*
 * Copyright 2018 The ThunderDB Authors.
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

// Package signature is a wrapper of btcsuite's signature package, except that it only exports types and
// functions which will be used by ThunderDB.
package signature

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"math/big"

	"github.com/btcsuite/btcd/btcec"
)

// PrivKeyBytesLen defines the length in bytes of a serialized private key.
const PrivKeyBytesLen = 32

// Signature is a type representing an ecdsa signature.
type Signature struct {
	R *big.Int
	S *big.Int
}

// PrivateKey wraps an ecdsa.PrivateKey as a convenience mainly for signing things with the the
// private key without having to directly import the ecdsa package.
type PrivateKey ecdsa.PrivateKey

// PublicKey wraps an ecdsa.PublicKey as a convenience mainly verifying signatures with the the
// public key without having to directly import the ecdsa package.
type PublicKey ecdsa.PublicKey

// Serialize returns the private key number d as a big-endian binary-encoded number, padded to a
// length of 32 bytes.
func (p *PrivateKey) Serialize() []byte {
	b := make([]byte, 0, PrivKeyBytesLen)
	return paddedAppend(PrivKeyBytesLen, b, p.D.Bytes())
}

// Sign generates an ECDSA signature for the provided hash (which should be the result of hashing
// a larger message) using the private key. Produced signature is deterministic (same message and
// same key yield the same signature) and canonical in accordance with RFC6979 and BIP0062.
func (p *PrivateKey) Sign(hash []byte) (*Signature, error) {
	s, e := p.toBTCEC().Sign(hash)
	return (*Signature)(s), e
}

// toBTCEC returns the private key as a *btcec.PrivateKey.
func (p *PrivateKey) toBTCEC() *btcec.PrivateKey {
	return (*btcec.PrivateKey)(p)
}

// toECDSA returns the public key as a *ecdsa.PublicKey.
func (p *PublicKey) toECDSA() *ecdsa.PublicKey {
	return (*ecdsa.PublicKey)(p)
}

// Verify calls ecdsa.Verify to verify the signature of hash using the public key. It returns true
// if the signature is valid, false otherwise.
func (sig *Signature) Verify(hash []byte, signee *PublicKey) bool {
	return ecdsa.Verify(signee.toECDSA(), hash, sig.R, sig.S)
}

// PrivKeyFromBytes returns a private and public key for `curve' based on the private key passed
// as an argument as a byte slice.
func PrivKeyFromBytes(curve elliptic.Curve, pk []byte) (*PrivateKey, *PublicKey) {
	x, y := curve.ScalarBaseMult(pk)

	priv := &ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{
			Curve: curve,
			X:     x,
			Y:     y,
		},
		D: new(big.Int).SetBytes(pk),
	}

	return (*PrivateKey)(priv), (*PublicKey)(&priv.PublicKey)
}

// paddedAppend appends the src byte slice to dst, returning the new slice. If the length of the
// source is smaller than the passed size, leading zero bytes are appended to the dst slice before
// appending src.
func paddedAppend(size uint, dst, src []byte) []byte {
	for i := 0; i < int(size)-len(src); i++ {
		dst = append(dst, 0)
	}
	return append(dst, src...)
}
