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

	ec "github.com/btcsuite/btcd/btcec"
	log "github.com/sirupsen/logrus"
)

// ErrNotKeyFile indicates specified key file is empty
var ErrNotKeyFile = errors.New("master key file empty")

// LoadMasterKey loads master key from keyFilePath
func LoadMasterKey(keyFilePath string) (key *ec.PrivateKey, err error) {
	fileContent, err := ioutil.ReadFile(keyFilePath)
	if err != nil {
		log.Errorf("error read key file: %s, err: %s", keyFilePath, err)
		return
	}

	if len(fileContent) != ec.PrivKeyBytesLen {
		log.Errorf("master key file size should be %d bytes", ec.PrivKeyBytesLen)
		return nil, ErrNotKeyFile
	}

	key, _ = ec.PrivKeyFromBytes(ec.S256(), fileContent)
	return
}

// SaveMasterKey saves master key to keyFilePath, default perm is 0600
func SaveMasterKey(keyFilePath string, key *ec.PrivateKey) (err error) {
	serializedKey := key.Serialize()
	return ioutil.WriteFile(keyFilePath, serializedKey, 0600)
}

// GenerateMasterKey generates a new EC private key
func GenerateMasterKey() (key *ec.PrivateKey, err error) {
	return ec.NewPrivateKey(ec.S256())
}
