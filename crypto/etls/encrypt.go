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

package etls

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"

	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
)

// KeyDerivation .according to ANSI X9.63 we should do a key derivation before using
// it as a symmetric key, there is not really a common standard KDF(Key Derivation Func).
// But as SSL/TLS/DTLS did it described in "RFC 4492 TLS ECC", we prefer a Double
// SHA-256 with it.
func KeyDerivation(rawKey []byte, keyLen int, hSuite *hash.HashSuite) (key []byte) {
	hashLen := hSuite.HashLen

	cnt := (keyLen-1)/hashLen + 1
	m := make([]byte, cnt*hashLen)
	copy(m, hSuite.HashFunc(rawKey))

	// Repeatedly call HashFunc until bytes generated is enough.
	// Each call to HashFunc uses data: prev hash + rawKey.
	d := make([]byte, hashLen+len(rawKey))
	start := 0
	for i := 1; i < cnt; i++ {
		start += hashLen
		copy(d, m[start-hashLen:start])
		copy(d[hashLen:], rawKey)
		copy(m[start:], hSuite.HashFunc(d))
	}
	return m[:keyLen]
}

func newDecStream(block cipher.Block, iv []byte) (cipher.Stream, error) {
	return cipher.NewCFBDecrypter(block, iv), nil
}

func newEncStream(block cipher.Block, iv []byte) (cipher.Stream, error) {
	return cipher.NewCFBEncrypter(block, iv), nil
}

func newAESCFBEncStream(key, iv []byte) (cipher.Stream, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return newEncStream(block, iv)
}

func newAESCFBDecStream(key, iv []byte) (cipher.Stream, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return newDecStream(block, iv)
}

type cipherInfo struct {
	keyLen       int
	ivLen        int
	newDecStream func(key, iv []byte) (cipher.Stream, error)
	newEncStream func(key, iv []byte) (cipher.Stream, error)
}

// Cipher struct keeps cipher mode, key, iv
type Cipher struct {
	encStream cipher.Stream
	decStream cipher.Stream
	key       []byte
	info      *cipherInfo
}

// NewCipher creates a cipher that can be used in Dial(), Listen() etc.
func NewCipher(rawKey []byte) (c *Cipher) {
	mi := &cipherInfo{
		32,
		16,
		newAESCFBDecStream,
		newAESCFBEncStream,
	}
	hSuite := &hash.HashSuite{
		HashLen:  hash.HashBSize,
		HashFunc: hash.DoubleHashB,
	}
	key := KeyDerivation(rawKey, mi.keyLen, hSuite)
	c = &Cipher{key: key, info: mi}

	return c
}

// initEncrypt Initializes the block cipher with CFB mode, returns IV.
func (c *Cipher) initEncrypt() (iv []byte, err error) {
	iv = make([]byte, c.info.ivLen)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}
	c.encStream, err = c.info.newEncStream(c.key, iv)
	return
}

func (c *Cipher) initDecrypt(iv []byte) (err error) {
	c.decStream, err = c.info.newDecStream(c.key, iv)
	return
}

func (c *Cipher) encrypt(dst, src []byte) {
	c.encStream.XORKeyStream(dst, src)
}

func (c *Cipher) decrypt(dst, src []byte) {
	c.decStream.XORKeyStream(dst, src)
}
