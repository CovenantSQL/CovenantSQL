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

package asymmetric

import (
	"crypto/ecdsa"
	"math/big"
	"time"

	"errors"

	ec "github.com/btcsuite/btcd/btcec"
	log "github.com/sirupsen/logrus"
	mine "github.com/thunderdb/ThunderDB/pow/cpuminer"
)

// PrivateKeyBytesLen defines the length in bytes of a serialized private key.
const PrivateKeyBytesLen = 32

// PrivateKey wraps an ec.PrivateKey as a convenience mainly for signing things with the the
// private key without having to directly import the ecdsa package.
type PrivateKey ec.PrivateKey

// PublicKey wraps an ec.PublicKey as a convenience mainly verifying signatures with the the
// public key without having to directly import the ecdsa package.
type PublicKey ec.PublicKey

// MarshalBinary does the serialization
func (pub *PublicKey) MarshalBinary() (keyBytes []byte, err error) {
	if pub == nil {
		return nil, errors.New("nil public key")
	}
	return pub.Serialize(), nil
}

// UnmarshalBinary does the deserialization
func (pub *PublicKey) UnmarshalBinary(keyBytes []byte) (err error) {
	if pub == nil {
		return errors.New("nil public key")
	}
	pubNew, err := ParsePubKey(keyBytes)
	if err == nil {
		*pub = *pubNew
	}
	return
}

// IsEqual return true if two keys are equal
func (pub *PublicKey) IsEqual(public *PublicKey) bool {
	return (*ec.PublicKey)(pub).IsEqual((*ec.PublicKey)(public))
}

// toECDSA returns the public key as a *ecdsa.PublicKey.
func (pub *PublicKey) toECDSA() *ecdsa.PublicKey {
	return (*ecdsa.PublicKey)(pub)
}

// Serialize is a function that converts a public key
// to uncompressed byte array
func (pub *PublicKey) Serialize() []byte {
	return (*ec.PublicKey)(pub).SerializeCompressed()
}

// ParsePubKey recovers the public key from pubKeyStr
func ParsePubKey(pubKeyStr []byte) (*PublicKey, error) {
	key, err := ec.ParsePubKey(pubKeyStr, ec.S256())
	return (*PublicKey)(key), err
}

// PrivKeyFromBytes returns a private and public key for `curve' based on the private key passed
// as an argument as a byte slice.
func PrivKeyFromBytes(pk []byte) (*PrivateKey, *PublicKey) {
	x, y := ec.S256().ScalarBaseMult(pk)

	priv := &ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{
			Curve: ec.S256(),
			X:     x,
			Y:     y,
		},
		D: new(big.Int).SetBytes(pk),
	}

	return (*PrivateKey)(priv), (*PublicKey)(&priv.PublicKey)
}

// Serialize returns the private key number d as a big-endian binary-encoded number, padded to a
// length of 32 bytes.
func (private *PrivateKey) Serialize() []byte {
	b := make([]byte, 0, PrivateKeyBytesLen)
	return paddedAppend(PrivateKeyBytesLen, b, private.D.Bytes())
}

// PubKey return the public key
func (private *PrivateKey) PubKey() *PublicKey {
	return (*PublicKey)((*ec.PrivateKey)(private).PubKey())
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

// GenSecp256k1KeyPair generate Secp256k1(used by Bitcoin) key pair
func GenSecp256k1KeyPair() (
	privateKey *PrivateKey,
	publicKey *PublicKey,
	err error) {

	privateKeyEc, err := ec.NewPrivateKey(ec.S256())
	if err != nil {
		log.Errorf("private key generation error: %s", err)
		return nil, nil, err
	}
	publicKey = (*PublicKey)(privateKeyEc.PubKey())
	privateKey = (*PrivateKey)(privateKeyEc)
	return
}

// GetPubKeyNonce will make his best effort to find a difficult enough
// nonce.
func GetPubKeyNonce(
	publicKey *PublicKey,
	difficulty int,
	timeThreshold time.Duration,
	quit chan struct{}) (nonce mine.NonceInfo) {

	miner := mine.NewCPUMiner(quit)
	nonceCh := make(chan mine.NonceInfo)
	// if miner finished his work before timeThreshold
	// make sure writing to the Stop chan non-blocking.
	stop := make(chan struct{}, 1)
	block := mine.MiningBlock{
		Data:      publicKey.Serialize(),
		NonceChan: nonceCh,
		Stop:      stop,
	}

	go miner.ComputeBlockNonce(block, mine.Uint256{}, difficulty)

	time.Sleep(timeThreshold)
	// stop miner
	block.Stop <- struct{}{}

	return <-block.NonceChan
}
