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
	"errors"

	"encoding/hex"

	"sync"

	log "github.com/sirupsen/logrus"

	ec "github.com/btcsuite/btcd/btcec"
	"github.com/coreos/bbolt"
	"github.com/thunderdb/ThunderDB/crypto/hash"
	mine "github.com/thunderdb/ThunderDB/pow/cpuminer"
	"github.com/thunderdb/ThunderDB/proto"
)

// PublicKeyStore holds db and bucket name
type PublicKeyStore struct {
	db     *bolt.DB
	bucket []byte
}

const (
	// kmsBucketName is the boltdb bucket name
	kmsBucketName = "kms"
)

var (
	// pks holds the singleton instance
	pks     *PublicKeyStore
	pksOnce sync.Once
)

var (
	// BPPublicKey is the public key of Block Producer
	BPPublicKeyStr = "02c1db96f2ba7e1cb4e9822d12de0f63f" +
		"b666feb828c7f509e81fab9bd7a34039c"
	// BPNodeID is the node id of Block Producer
	// 	{{235455434 0 0 1537228675571897187} 42 00000000003beb34cc9324644920e085c63675d2de5d822b4b61ac7d13a967df}
	BPNodeID = "00000000003beb34cc9324644920e085" +
		"c63675d2de5d822b4b61ac7d13a967df"
	// BPNonce is the nonce, SEE cmd/idminer for more
	BPNonce = mine.Uint256{
		235455434,
		0,
		0,
		1537228675571897187,
	}
	// BPPublicKey point to BlockProducer public key
	BPPublicKey *ec.PublicKey
)

var (
	// ErrBucketNotInitialized indicates bucket not initialized
	ErrBucketNotInitialized = errors.New("bucket not initialized")
	// ErrKeyNotFound indicates key not found
	ErrKeyNotFound = errors.New("key not found")
	// ErrNotValidNodeID indicates that is not valid node id
	ErrNotValidNodeID = errors.New("not valid node id")
	// ErrNodeIDKeyNonceNotMatch indicates node id, key, nonce not match
	ErrNodeIDKeyNonceNotMatch = errors.New("nodeID, key, nonce not match")
)

// InitPublicKeyStore opens a db file, if not exist, creates it.
// and creates a bucket if not exist
//	IF ANY ERROR, the func will FATAL
func InitPublicKeyStore(dbPath string) {
	pksOnce.Do(func() {
		bdb, err := bolt.Open(dbPath, 0600, nil)
		if err != nil {
			log.Fatalf("InitPublicKeyStore failed: %s", err)
		}

		name := []byte(kmsBucketName)
		err = (*bolt.DB)(bdb).Update(func(tx *bolt.Tx) error {
			if _, err := tx.CreateBucketIfNotExists(name); err != nil {
				return errors.New("could not create bucket: " + err.Error())
			}
			return nil // return from Update func
		})
		if err != nil {
			log.Fatalf("InitPublicKeyStore failed: %s", err)
		}

		pks = &PublicKeyStore{
			db:     bdb,
			bucket: name,
		}

		// Load BlockProducer public key, set it in public key store
		// as all inputs of this func are pre defined. there should not
		// be any error, if any then panic!
		publicKeyBytes, err := hex.DecodeString(BPPublicKeyStr)
		if err == nil {
			BPPublicKey, err = ec.ParsePubKey(publicKeyBytes, ec.S256())
			if err == nil {
				err = setPublicKey(proto.NodeID(BPNodeID), BPPublicKey)
			}
		}
		if err != nil {
			log.Fatalf("InitPublicKeyStore failed: %s", err)
		}
	})

	return
}

// GetPublicKey gets a PublicKey of given id
// Returns an error if the id was not found
func GetPublicKey(id proto.NodeID) (publicKey *ec.PublicKey, err error) {
	err = (*bolt.DB)(pks.db).View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(pks.bucket)
		if bucket == nil {
			return ErrBucketNotInitialized
		}
		byteVal := bucket.Get([]byte(id))
		if byteVal == nil {
			return ErrKeyNotFound
		}

		publicKey, err = ec.ParsePubKey(byteVal, ec.S256())
		return err // return from View func
	})
	return
}

// SetPublicKey verify nonce and set Public Key
func SetPublicKey(id proto.NodeID, nonce mine.Uint256, publicKey *ec.PublicKey) (err error) {
	keyHash := mine.HashBlock(publicKey.SerializeCompressed(), nonce)
	idHash, err := hash.NewHashFromStr(string(id))
	if err != nil {
		return ErrNotValidNodeID
	}

	if !keyHash.IsEqual(idHash) {
		return ErrNodeIDKeyNonceNotMatch
	}
	return setPublicKey(id, publicKey)
}

// setPublicKey sets id and its publicKey
func setPublicKey(id proto.NodeID, publicKey *ec.PublicKey) (err error) {
	return (*bolt.DB)(pks.db).Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(pks.bucket)
		if bucket == nil {
			return ErrBucketNotInitialized
		}
		return bucket.Put([]byte(id), publicKey.SerializeCompressed())
	})
}

// DelPublicKey removes PublicKey to the id
func DelPublicKey(id proto.NodeID) (err error) {
	return (*bolt.DB)(pks.db).Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(pks.bucket)
		if bucket == nil {
			return ErrBucketNotInitialized
		}
		return bucket.Delete([]byte(id))
	})
}

// removeBucket this bucket
func removeBucket() error {
	err := (*bolt.DB)(pks.db).Update(func(tx *bolt.Tx) error {
		return tx.DeleteBucket([]byte(pks.bucket))
	})
	// ks.bucket == nil means bucket not exist
	pks.bucket = nil
	return err
}
