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

package kms

import (
	"io/ioutil"

	"errors"

	"bytes"

	ec "github.com/btcsuite/btcd/btcec"
	log "github.com/sirupsen/logrus"
	"github.com/thunderdb/ThunderDB/crypto/hash"
	"github.com/thunderdb/ThunderDB/crypto/symmetric"
)

// ErrNotKeyFile indicates specified key file is empty
var ErrNotKeyFile = errors.New("private key file empty")

// ErrHashNotMatch indicates specified key hash is wrong
var ErrHashNotMatch = errors.New("private key hash not match")

// LoadPrivateKey loads private key from keyFilePath, and verifies the hash
// head
func LoadPrivateKey(keyFilePath string, masterKey []byte) (key *ec.PrivateKey, err error) {
	fileContent, err := ioutil.ReadFile(keyFilePath)
	if err != nil {
		log.Errorf("error read key file: %s, err: %s", keyFilePath, err)
		return
	}

	decData, err := symmetric.DecryptWithPassword(fileContent, masterKey)
	if err != nil {
		log.Errorf("decrypt private key error")
		return
	}

	// sha256 + privateKey
	if len(decData) != hash.HashBSize+ec.PrivKeyBytesLen {
		log.Errorf("private key file size should be %d bytes", hash.HashBSize+ec.PrivKeyBytesLen)
		return nil, ErrNotKeyFile
	}

	computedHash := hash.DoubleHashB(decData[hash.HashBSize:])
	if bytes.Compare(computedHash, decData[:hash.HashBSize]) != 0 {
		return nil, ErrHashNotMatch
	}

	key, _ = ec.PrivKeyFromBytes(ec.S256(), decData[hash.HashBSize:])
	return
}

// SavePrivateKey saves private key with its hash on the head to keyFilePath,
// default perm is 0600
func SavePrivateKey(keyFilePath string, key *ec.PrivateKey, masterKey []byte) (err error) {
	serializedKey := key.Serialize()
	keyHash := hash.DoubleHashB(serializedKey)
	rawData := append(keyHash, serializedKey...)
	encKey, err := symmetric.EncryptWithPassword(rawData, masterKey)
	if err != nil {
		log.Errorf("encrypt private key error")
		return
	}
	return ioutil.WriteFile(keyFilePath, encKey, 0600)
}

// GeneratePrivateKey generates a new EC private key
func GeneratePrivateKey() (key *ec.PrivateKey, err error) {
	return ec.NewPrivateKey(ec.S256())
}
