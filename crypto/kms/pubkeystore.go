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

	ec "github.com/btcsuite/btcd/btcec"
	bolt "github.com/coreos/bbolt"
	"github.com/thunderdb/ThunderDB/proto"
)

const (
	// kmsBucketName is the boltdb bucket name
	kmsBucketName = "kms"
	// BPPublicKey is the public key of Block Producer
	BPPublicKey = "02c1db96f2ba7e1cb4e9822d12de0f63f" +
		"b666feb828c7f509e81fab9bd7a34039c"
)

// PublicKeyStore holds db and bucket name
type PublicKeyStore struct {
	db     *bolt.DB
	bucket []byte
}

var (
	// ErrBucketNotInitialized indicates bucket not initialized
	ErrBucketNotInitialized = errors.New("bucket not initialized")
	// ErrKeyNotFound indicates key not found
	ErrKeyNotFound = errors.New("key not found")
)

// NewPublicKeyStore opens a db file, if not exist, creates it.
// and creates a bucket if not exist
func NewPublicKeyStore(dbPath string) (pks *PublicKeyStore, err error) {
	bdb, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		return nil, err
	}

	name := []byte(kmsBucketName)
	err = (*bolt.DB)(bdb).Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(name); err != nil {
			return errors.New("could not create bucket: " + err.Error())
		}
		return nil // return from Update func
	})
	if err != nil {
		return nil, err
	}

	pks = &PublicKeyStore{
		db:     bdb,
		bucket: name,
	}
	return
}

// GetPublicKey gets a PublicKey of given id
// Returns an error if the id was not found
func (ks *PublicKeyStore) GetPublicKey(id proto.NodeID) (publicKey *ec.PublicKey, err error) {
	err = (*bolt.DB)(ks.db).View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(ks.bucket)
		if bucket == nil {
			return ErrBucketNotInitialized
		}
		byteVal := bucket.Get([]byte(id))
		if byteVal == nil {
			return ErrKeyNotFound
		}

		publicKey, err = ec.ParsePubKey(byteVal, ec.S256())
		if err != nil {
			return err
		}
		return nil // return from View func
	})
	return
}

// SetPublicKey sets id and its publicKey
func (ks *PublicKeyStore) SetPublicKey(id proto.NodeID, publicKey *ec.PublicKey) (err error) {
	return (*bolt.DB)(ks.db).Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(ks.bucket)
		if bucket == nil {
			return ErrBucketNotInitialized
		}
		return bucket.Put([]byte(id), publicKey.SerializeCompressed())
	})
}

// DelPublicKey removes PublicKey to the id
func (ks *PublicKeyStore) DelPublicKey(id proto.NodeID) (err error) {
	return (*bolt.DB)(ks.db).Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(ks.bucket)
		if bucket == nil {
			return ErrBucketNotInitialized
		}
		return bucket.Delete([]byte(id))
	})
}

// RemoveBucket this bucket
func (ks *PublicKeyStore) RemoveBucket() error {
	err := (*bolt.DB)(ks.db).Update(func(tx *bolt.Tx) error {
		return tx.DeleteBucket([]byte(ks.bucket))
	})
	// ks.bucket == nil means bucket not exist
	ks.bucket = nil
	return err
}
