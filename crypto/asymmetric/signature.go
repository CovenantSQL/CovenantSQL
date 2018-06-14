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
package asymmetric

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"math/big"

	ec "github.com/btcsuite/btcd/btcec"
)

// Signature is a type representing an ecdsa signature.
type Signature struct {
	R *big.Int
	S *big.Int
}

func (s *Signature) toec() *ec.Signature {
	return (*ec.Signature)(s)
}

// Serialize converts a signature to stirng
func (s *Signature) Serialize() []byte {
	return (*ec.Signature)(s).Serialize()
}

// ParseDERSignature recovers the signature from a sigStr
func ParseDERSignature(sigStr []byte, curve elliptic.Curve) (*Signature, error) {
	sig, err := ec.ParseDERSignature(sigStr, curve)
	return (*Signature)(sig), err
}

// IsEqual return true if two signature is equal
func (s *Signature) IsEqual(signature *Signature) bool {
	return (*ec.Signature)(s).IsEqual((*ec.Signature)(signature))
}

// Sign generates an ECDSA signature for the provided hash (which should be the result of hashing
// a larger message) using the private key. Produced signature is deterministic (same message and
// same key yield the same signature) and canonical in accordance with RFC6979 and BIP0062.
func (p *PrivateKey) Sign(hash []byte) (*Signature, error) {
	s, e := (*ec.PrivateKey)(p).Sign(hash)
	return (*Signature)(s), e
}

// Verify calls ecdsa.Verify to verify the signature of hash using the public key. It returns true
// if the signature is valid, false otherwise.
func (s *Signature) Verify(hash []byte, signee *PublicKey) bool {
	return ecdsa.Verify(signee.toECDSA(), hash, s.R, s.S)
}
