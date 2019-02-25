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

// Package crypto implements Asymmetric, Symmetric Encryption and Hash function.
package crypto

import (
	"bytes"
	"crypto/aes"
	"errors"

	ec "github.com/btcsuite/btcd/btcec"

	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
)

var errInvalidPadding = errors.New("invalid PKCS#7 padding")

// Implement PKCS#7 padding with block size of 16 (AES block size).

// AddPKCSPadding adds padding to a block of data.
func AddPKCSPadding(src []byte) []byte {
	padding := aes.BlockSize - len(src)%aes.BlockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(src, padtext...)
}

// RemovePKCSPadding removes padding from data that was added with addPKCSPadding.
func RemovePKCSPadding(src []byte) ([]byte, error) {
	length := len(src)
	padLength := int(src[length-1])
	if padLength > aes.BlockSize || length < aes.BlockSize {
		return nil, errInvalidPadding
	}

	return src[:length-padLength], nil
}

// EncryptAndSign (inputPublicKey, inData) MAIN PROCEDURE:
//	1. newPrivateKey, newPubKey := genSecp256k1Keypair()
//	2. encKey, HMACKey := SHA512(ECDH(newPrivateKey, inputPublicKey))
//	3. PaddedIn := PKCSPadding(in)
//	4. OutBytes := IV + newPubKey + AES-256-CBC(encKey, PaddedIn) + HMAC-SHA-256(HMACKey)
func EncryptAndSign(inputPublicKey *asymmetric.PublicKey, inData []byte) ([]byte, error) {
	return ec.Encrypt((*ec.PublicKey)(inputPublicKey), inData)
}

// DecryptAndCheck (inputPrivateKey, inData) MAIN PROCEDURE:
//	1. Decrypt the inData
//  2. Verify the HMAC.
func DecryptAndCheck(inputPrivateKey *asymmetric.PrivateKey, inData []byte) ([]byte, error) {
	return ec.Decrypt((*ec.PrivateKey)(inputPrivateKey), inData)
}
