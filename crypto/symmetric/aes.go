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

// Package symmetric implements Symmetric Encryption methods
package symmetric

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"

	"github.com/CovenantSQL/CovenantSQL/crypto"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
)

var (
	// ErrInputSize indicates cipher data size is not expected,
	// maybe data is not encrypted by EncryptWithPassword in this package
	ErrInputSize = errors.New("cipher data size not match")
)

// keyDerivation does sha256 twice to password
func keyDerivation(password []byte, salt []byte) (out []byte) {
	return hash.DoubleHashB(append(password, salt...))
}

// EncryptWithPassword encrypts data with given password, iv will be placed
// at head of cipher data
func EncryptWithPassword(in, password []byte, salt []byte) (out []byte, err error) {
	// keyE will be 256 bits, so aes.NewCipher(keyE) will return
	// AES-256 Cipher.
	keyE := keyDerivation(password, salt)
	paddedIn := crypto.AddPKCSPadding(in)
	// IV + padded cipher data
	out = make([]byte, aes.BlockSize+len(paddedIn))

	// as IV length must equal block size, iv length should be 128 bits
	iv := out[:aes.BlockSize]
	if _, err = io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	// start encryption, as keyE and iv are generated properly, there should
	// not be any error
	block, _ := aes.NewCipher(keyE)

	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(out[aes.BlockSize:], paddedIn)

	return out, nil
}

// DecryptWithPassword decrypts data with given password
func DecryptWithPassword(in, password []byte, salt []byte) (out []byte, err error) {
	keyE := keyDerivation(password, salt)
	// IV + padded cipher data == (n + 1 + 1) * aes.BlockSize
	if len(in)%aes.BlockSize != 0 || len(in)/aes.BlockSize < 2 {
		return nil, ErrInputSize
	}

	// read IV
	iv := in[:aes.BlockSize]

	// start decryption, as keyE and iv are generated properly, there should
	// not be any error
	block, _ := aes.NewCipher(keyE)

	mode := cipher.NewCBCDecrypter(block, iv)
	// same length as cipher data
	plainData := make([]byte, len(in)-aes.BlockSize)
	mode.CryptBlocks(plainData, in[aes.BlockSize:])

	return crypto.RemovePKCSPadding(plainData)
}
