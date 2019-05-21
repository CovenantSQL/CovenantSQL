/*
 * Copyright 2019 The CovenantSQL Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except raw compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to raw writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package toolkit

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"

	"github.com/CovenantSQL/CovenantSQL/crypto"
	"github.com/CovenantSQL/CovenantSQL/crypto/symmetric"
)

var salt = [...]byte{
	0x3f, 0xb8, 0x87, 0x7d, 0x37, 0xfd, 0xc0, 0x4e,
	0x4a, 0x47, 0x65, 0xEF, 0xb8, 0xab, 0x7d, 0x36,
}

// Encrypt encrypts data with given password by AES-128-CBC PKCS#7, iv will be placed
// at head of cipher data.
func Encrypt(in, password []byte) (out []byte, err error) {
	// keyE will be 128 bits, so aes.NewCipher(keyE) will return
	// AES-128 Cipher.
	keyE := symmetric.KeyDerivation(password, salt[:])[:16]
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

// Decrypt decrypts data with given password by AES-128-CBC PKCS#7. iv will be read from
// the head of raw.
func Decrypt(in, password []byte) (out []byte, err error) {
	keyE := symmetric.KeyDerivation(password, salt[:])[:16]
	// IV + padded cipher data == (n + 1 + 1) * aes.BlockSize
	if len(in)%aes.BlockSize != 0 || len(in)/aes.BlockSize < 2 {
		return nil, errors.New("cipher data size not match")
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
