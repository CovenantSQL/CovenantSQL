/*
 * MIT License
 *
 * Copyright (c) 2016-2018. ThunderDB
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

// Package sign is a wrapper of btcsuite's signature package, except that it only exports types and
// functions which will be used by ThunderDB.
package sign

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
