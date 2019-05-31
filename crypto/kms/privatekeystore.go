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

package kms

import (
	"bytes"
	"errors"
	"io/ioutil"
	"os"

	"github.com/btcsuite/btcutil/base58"

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/crypto/symmetric"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

var (
	// ErrNotKeyFile indicates specified key file is empty
	ErrNotKeyFile = errors.New("private key file empty")
	// ErrHashNotMatch indicates specified key hash is wrong
	ErrHashNotMatch = errors.New("private key hash not match")
	// ErrInvalidBase58Version indicates specified key is not base58 version
	ErrInvalidBase58Version = errors.New("invalid base58 version")
	// PrivateKeyStoreVersion defines the private key version byte.
	PrivateKeyStoreVersion byte = 0x23
	// oldPrivateKDFSalt is the old KDF salt for private key encryption
	oldPrivateKDFSalt = "auxten-key-salt-auxten"
	// privateKDFSalt is the KDF salt for private key encryption
	privateKDFSalt = []byte{
		0xC0, 0x4E, 0xA4, 0x71, 0x49, 0x65, 0x41, 0x31,
		0x79, 0x4b, 0x6a, 0x70, 0x2f, 0x39, 0x45, 0x43,
	}
)

// DecodePrivateKey loads private key from private key bytes form.
func DecodePrivateKey(keyBytes []byte, masterKey []byte) (key *asymmetric.PrivateKey, err error) {
	var (
		isBinaryKey bool
		decData     []byte
	)

	// It's very impossible to get an raw private key base58 decodable.
	// So if it's not base58 decodable we just make fileContent the encData
	encData, version, err := base58.CheckDecode(string(keyBytes))
	switch err {
	case base58.ErrChecksum:
		return

	case base58.ErrInvalidFormat:
		// be compatible with the original binary private key format
		isBinaryKey = true
		encData = keyBytes
	}

	if version != 0 && version != PrivateKeyStoreVersion {
		return nil, ErrInvalidBase58Version
	}

	if isBinaryKey {
		decData, err = symmetric.DecryptWithPassword(encData, masterKey, []byte(oldPrivateKDFSalt))
		if err != nil {
			log.Error("decrypt private key error")
			return
		}

		// sha256 + privateKey
		if len(decData) != hash.HashBSize+asymmetric.PrivateKeyBytesLen {
			log.WithFields(log.Fields{
				"expected": hash.HashBSize + asymmetric.PrivateKeyBytesLen,
				"actual":   len(decData),
			}).Error("wrong binary private key file size")
			return nil, ErrNotKeyFile
		}

		computedHash := hash.DoubleHashB(decData[hash.HashBSize:])
		if !bytes.Equal(computedHash, decData[:hash.HashBSize]) {
			return nil, ErrHashNotMatch
		}
		key, _ = asymmetric.PrivKeyFromBytes(decData[hash.HashBSize:])
	} else {
		decData, err = symmetric.DecryptWithPassword(encData, masterKey, privateKDFSalt)
		if err != nil {
			log.Error("decrypt private key error")
			return
		}

		// privateKey
		if len(decData) != asymmetric.PrivateKeyBytesLen {
			log.WithFields(log.Fields{
				"expected": asymmetric.PrivateKeyBytesLen,
				"actual":   len(decData),
			}).Error("wrong base58 private key file size")
			return nil, ErrNotKeyFile
		}
		key, _ = asymmetric.PrivKeyFromBytes(decData)
	}

	return
}

// LoadPrivateKey loads private key from keyFilePath, and verifies the hash head.
func LoadPrivateKey(keyFilePath string, masterKey []byte) (key *asymmetric.PrivateKey, err error) {
	fileContent, err := ioutil.ReadFile(keyFilePath)
	if err != nil {
		log.WithField("path", keyFilePath).WithError(err).Error("read key file failed")
		return
	}

	return DecodePrivateKey(fileContent, masterKey)
}

// EncodePrivateKey encode private to key to string format.
func EncodePrivateKey(key *asymmetric.PrivateKey, masterKey []byte) (keyBytes []byte, err error) {
	serializedKey := key.Serialize()
	encKey, err := symmetric.EncryptWithPassword(serializedKey, masterKey, privateKDFSalt)
	if err != nil {
		return
	}

	keyBytes = []byte(base58.CheckEncode(encKey, PrivateKeyStoreVersion))

	return
}

// SavePrivateKey saves private key with its hash on the head to keyFilePath,
// default perm is 0600.
func SavePrivateKey(keyFilePath string, key *asymmetric.PrivateKey, masterKey []byte) (err error) {
	var keyBytes []byte
	keyBytes, err = EncodePrivateKey(key, masterKey)
	if err != nil {
		return
	}

	return ioutil.WriteFile(keyFilePath, keyBytes, 0600)
}

// InitLocalKeyPair initializes local private key.
func InitLocalKeyPair(privateKeyPath string, masterKey []byte) (err error) {
	var privateKey *asymmetric.PrivateKey
	var publicKey *asymmetric.PublicKey
	initLocalKeyStore()
	privateKey, err = LoadPrivateKey(privateKeyPath, masterKey)
	if err != nil {
		log.WithError(err).Info("load private key failed")
		if err == ErrNotKeyFile {
			log.WithField("path", privateKeyPath).Error("not a valid private key file")
			return
		}
		if _, ok := err.(*os.PathError); (ok || err == os.ErrNotExist) && conf.GConf.GenerateKeyPair {
			log.Info("private key file not exist, generating one")
			privateKey, publicKey, err = asymmetric.GenSecp256k1KeyPair()
			if err != nil {
				log.WithError(err).Error("generate private key failed")
				return
			}
			log.WithField("path", privateKeyPath).Info("saving new private key file")
			err = SavePrivateKey(privateKeyPath, privateKey, masterKey)
			if err != nil {
				log.WithError(err).Error("save private key failed")
				return
			}
		} else {
			log.WithField("path", privateKeyPath).WithError(err).Error("unexpected error while loading private key")
			return
		}
	}
	if publicKey == nil {
		publicKey = privateKey.PubKey()
	}
	log.Debugf("\n### Public Key ###\n%#x\n### Public Key ###\n", publicKey.Serialize())
	SetLocalKeyPair(privateKey, publicKey)
	return
}
