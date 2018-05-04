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

// Package crypto implements Asymmetric, Symmetric Encryption and Hash function
package crypto

import (
	"bytes"
	"crypto/aes"
	"errors"
	"github.com/btcsuite/btcd/btcec"
)

var errInvalidPadding = errors.New("invalid PKCS#7 padding")

// Implement PKCS#7 padding with block size of 16 (AES block size).

// AddPKCSPadding adds padding to a block of data
func AddPKCSPadding(src []byte) []byte {
	padding := aes.BlockSize - len(src)%aes.BlockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(src, padtext...)
}

// RemovePKCSPadding removes padding from data that was added with addPKCSPadding
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
func EncryptAndSign(inputPublicKey *btcec.PublicKey, inData []byte) ([]byte, error) {
	return btcec.Encrypt(inputPublicKey, inData)
}

// DecryptAndCheck (inputPrivateKey, inData) MAIN PROCEDURE:
//	1. Decrypt the inData
//  2. Verify the HMAC
func DecryptAndCheck(inputPrivateKey *btcec.PrivateKey, inData []byte) ([]byte, error) {
	return btcec.Decrypt(inputPrivateKey, inData)
}
