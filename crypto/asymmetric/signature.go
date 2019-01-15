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

package asymmetric

import (
	"crypto/elliptic"
	"errors"
	"math/big"

	hsp "github.com/CovenantSQL/HashStablePack/marshalhash"
	ec "github.com/btcsuite/btcd/btcec"
	lru "github.com/hashicorp/golang-lru"

	"github.com/CovenantSQL/CovenantSQL/crypto/secp256k1"
	"github.com/CovenantSQL/CovenantSQL/utils"
)

var (
	// BypassSignature is the flag indicate if bypassing signature sign & verify
	BypassSignature = false
	bypassS         *Signature
	verifyCache     *lru.Cache
)

// For test Signature.Sign mock
func init() {
	priv, _ := ec.NewPrivateKey(ec.S256())
	ss, _ := (*ec.PrivateKey)(priv).Sign(([]byte)("00000000000000000000000000000000"))
	bypassS = (*Signature)(ss)
	verifyCache, _ = lru.New(256)
}

// Signature is a type representing an ecdsa signature.
type Signature struct {
	R *big.Int
	S *big.Int
}

// Serialize converts a signature to stirng
func (s *Signature) Serialize() []byte {
	return (*ec.Signature)(s).Serialize()
}

// ParseSignature recovers the signature from a sigStr using koblitz curve.
func ParseSignature(sigStr []byte) (*Signature, error) {
	return ParseDERSignature(sigStr, ec.S256())
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
func (private *PrivateKey) Sign(hash []byte) (*Signature, error) {
	if len(hash) != 32 {
		return nil, errors.New("only hash can be signed")
	}
	if BypassSignature {
		return bypassS, nil
	}
	seckey := utils.PaddedBigBytes(private.D, private.Params().BitSize/8)
	defer zeroBytes(seckey)
	sb, e := secp256k1.Sign(hash, seckey)
	s := &Signature{
		R: new(big.Int).SetBytes(sb[:32]),
		S: new(big.Int).SetBytes(sb[32:64]),
	}
	//s, e := (*ec.PrivateKey)(private).Sign(hash)

	return (*Signature)(s), e
}

// Verify calls ecdsa.Verify to verify the signature of hash using the public key. It returns true
// if the signature is valid, false otherwise.
func (s *Signature) Verify(hash []byte, signee *PublicKey) bool {
	if BypassSignature {
		return true
	}
	if signee == nil || s == nil {
		return false
	}

	cacheKey := make([]byte, 64+len(hash)+ec.PubKeyBytesLenUncompressed)
	signature := cacheKey[:64]
	copy(signature, utils.PaddedBigBytes(s.R, 32))
	copy(signature[32:], utils.PaddedBigBytes(s.S, 32))
	copy(cacheKey[64:64+len(hash)], hash)
	signeeBytes := (*ec.PublicKey)(signee).SerializeUncompressed()
	copy(cacheKey[64+len(hash):], signeeBytes)

	if _, ok := verifyCache.Get(string(cacheKey)); ok {
		return true
	}
	valid := secp256k1.VerifySignature(signeeBytes, hash, signature)
	if valid {
		verifyCache.Add(string(cacheKey), nil)
	}
	return valid
	//return ecdsa.Verify(signee.toECDSA(), hash, s.R, s.S)
}

// MarshalBinary does the serialization.
func (s *Signature) MarshalBinary() (keyBytes []byte, err error) {
	if s == nil {
		err = errors.New("nil signature")
		return
	}

	keyBytes = s.Serialize()
	return
}

// MarshalHash marshals for hash
func (s *Signature) MarshalHash() (keyBytes []byte, err error) {
	return s.MarshalBinary()
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (s Signature) Msgsize() (sz int) {
	sz = hsp.BytesPrefixSize + 70
	return
}

// UnmarshalBinary does the deserialization.
func (s *Signature) UnmarshalBinary(keyBytes []byte) (err error) {
	if s == nil {
		err = errors.New("nil signature")
		return
	}

	var sig *Signature
	sig, err = ParseSignature(keyBytes)
	if err != nil {
		return
	}
	*s = *sig
	return
}

func zeroBytes(bytes []byte) {
	for i := range bytes {
		bytes[i] = 0
	}
}
