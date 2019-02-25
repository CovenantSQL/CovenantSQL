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
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	hsp "github.com/CovenantSQL/HashStablePack/marshalhash"
	ec "github.com/btcsuite/btcd/btcec"

	mine "github.com/CovenantSQL/CovenantSQL/pow/cpuminer"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

// PrivateKeyBytesLen defines the length in bytes of a serialized private key.
const PrivateKeyBytesLen = 32

var parsedPublicKeyCache sync.Map

// PrivateKey wraps an ec.PrivateKey as a convenience mainly for signing things with the the
// private key without having to directly import the ecdsa package.
type PrivateKey ec.PrivateKey

// PublicKey wraps an ec.PublicKey as a convenience mainly verifying signatures with the the
// public key without having to directly import the ecdsa package.
type PublicKey ec.PublicKey

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message.
func (k PublicKey) Msgsize() (s int) {
	s = hsp.BytesPrefixSize + ec.PubKeyBytesLenCompressed
	return
}

// IsValid test a PublicKey is not nil.
func (k *PublicKey) IsValid() bool {
	return k != nil && k.X != nil && k.Y != nil
}

// MarshalHash marshals for hash.
func (k *PublicKey) MarshalHash() (keyBytes []byte, err error) {
	return k.MarshalBinary()
}

// MarshalBinary does the serialization.
func (k *PublicKey) MarshalBinary() (keyBytes []byte, err error) {
	return k.Serialize(), nil
}

// UnmarshalBinary does the deserialization.
func (k *PublicKey) UnmarshalBinary(keyBytes []byte) (err error) {
	if len(keyBytes) == 0 {
		return errors.New("empty key bytes")
	}
	pubKeyI, ok := parsedPublicKeyCache.Load(string(keyBytes))
	if ok {
		*k = *pubKeyI.(*PublicKey)
	} else {
		pubNew, err := ParsePubKey(keyBytes)
		if err == nil {
			*k = *pubNew
			parsedPublicKeyCache.Store(string(keyBytes), pubNew)
		}
	}
	return
}

// MarshalYAML implements the yaml.Marshaler interface.
func (k PublicKey) MarshalYAML() (interface{}, error) {
	return fmt.Sprintf("%x", k.Serialize()), nil
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (k *PublicKey) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var str string
	if err := unmarshal(&str); err != nil {
		return err
	}

	// load public key string
	pubKeyBytes, err := hex.DecodeString(str)
	if err != nil {
		return err
	}

	err = k.UnmarshalBinary(pubKeyBytes)

	return err
}

// IsEqual return true if two keys are equal.
func (k *PublicKey) IsEqual(public *PublicKey) bool {
	return (*ec.PublicKey)(k).IsEqual((*ec.PublicKey)(public))
}

// Serialize is a function that converts a public key
// to uncompressed byte array
//
// TODO(leventeliu): use SerializeUncompressed, which is about 40 times faster than
// SerializeCompressed.
//
// BenchmarkParsePublicKey-12                 50000             39819 ns/op            2401 B/op         35 allocs/op
// BenchmarkParsePublicKey-12               1000000              1039 ns/op             384 B/op          9 allocs/op.
func (k *PublicKey) Serialize() []byte {
	return (*ec.PublicKey)(k).SerializeCompressed()
}

// ParsePubKey recovers the public key from pubKeyStr.
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

// PubKey return the public key.
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

// GenSecp256k1KeyPair generate Secp256k1(used by Bitcoin) key pair.
func GenSecp256k1KeyPair() (
	privateKey *PrivateKey,
	publicKey *PublicKey,
	err error) {

	privateKeyEc, err := ec.NewPrivateKey(ec.S256())
	if err != nil {
		log.WithError(err).Error("private key generation failed")
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
