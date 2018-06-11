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
	pb "github.com/golang/protobuf/proto"
	"github.com/thunderdb/ThunderDB/crypto/hash"
	mine "github.com/thunderdb/ThunderDB/pow/cpuminer"
	"github.com/thunderdb/ThunderDB/proto"
	"github.com/thunderdb/ThunderDB/types"
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
	// BPPublicKeyStr is the public key of Block Producer
	BPPublicKeyStr = "02c1db96f2ba7e1cb4e9822d12de0f63f" +
		"b666feb828c7f509e81fab9bd7a34039c"
	// BPNodeID is the node id of Block Producer
	// 	{{172044431 0 0 2594073388332474261}    39 00000000011a34cb8142780f692a4097d883aa2ac8a534a070a134f11bcca573}
	BPNodeID = "00000000011a34cb8142780f692a4097d883aa2ac8a534a070a134f11bcca573"
	// BPNonce is the nonce, SEE: cmd/idminer for more
	BPNonce = mine.Uint256{
		172044431,
		0,
		0,
		2594073388332474261,
	}
	// BPPublicKey point to BlockProducer public key
	BPPublicKey *ec.PublicKey
)

var (
	// ErrBucketNotInitialized indicates bucket not initialized
	ErrBucketNotInitialized = errors.New("bucket not initialized")
	// ErrNilNode indicates input node is nil
	ErrNilNode = errors.New("nil node")
	// ErrKeyNotFound indicates key not found
	ErrKeyNotFound = errors.New("key not found")
	// ErrNotValidNodeID indicates that is not valid node id
	ErrNotValidNodeID = errors.New("not valid node id")
	// ErrNodeIDKeyNonceNotMatch indicates node id, key, nonce not match
	ErrNodeIDKeyNonceNotMatch = errors.New("nodeID, key, nonce not match")
)

// InitPublicKeyStore opens a db file, if not exist, creates it.
// and creates a bucket if not exist
func InitPublicKeyStore(dbPath string) (err error) {
	pksOnce.Do(func() {
		var bdb *bolt.DB
		bdb, err = bolt.Open(dbPath, 0600, nil)
		if err != nil {
			log.Errorf("InitPublicKeyStore failed: %s", err)
			return
		}

		name := []byte(kmsBucketName)
		err = (*bolt.DB)(bdb).Update(func(tx *bolt.Tx) error {
			if _, err := tx.CreateBucketIfNotExists(name); err != nil {
				log.Errorf("could not create bucket: %s", err)
				return err
			}
			return nil // return from Update func
		})
		if err != nil {
			log.Errorf("InitPublicKeyStore failed: %s", err)
			return
		}

		pks = &PublicKeyStore{
			db:     bdb,
			bucket: name,
		}

		// Load BlockProducer public key, set it in public key store
		// as all inputs of this func are pre defined. there should not
		// be any error, if any then panic!
		var publicKeyBytes []byte
		publicKeyBytes, err = hex.DecodeString(BPPublicKeyStr)
		if err == nil {
			BPPublicKey, err = ec.ParsePubKey(publicKeyBytes, ec.S256())
			if err == nil {
				node := &proto.Node{
					ID:        proto.NodeID(BPNodeID),
					Addr:      "",
					PublicKey: BPPublicKey,
					Nonce:     BPNonce,
				}
				err = setPublicKey(node)
			}
		}
	})

	return
}

// GetPublicKey gets a PublicKey of given id
// Returns an error if the id was not found
func GetPublicKey(id proto.NodeID) (publicKey *ec.PublicKey, err error) {
	node, err := GetNodeInfo(id)
	if err == nil {
		publicKey = node.PublicKey
	}
	return
}

// GetNodeInfo gets node info of given id
// Returns an error if the id was not found
func GetNodeInfo(id proto.NodeID) (nodeInfo *proto.Node, err error) {
	err = (*bolt.DB)(pks.db).View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(pks.bucket)
		if bucket == nil {
			return ErrBucketNotInitialized
		}
		byteVal := bucket.Get([]byte(id))
		if byteVal == nil {
			return ErrKeyNotFound
		}

		nodeInfoTypes := &types.Node{}
		err = pb.Unmarshal(byteVal, nodeInfoTypes)
		if err == nil {
			publicKey, err := ec.ParsePubKey(nodeInfoTypes.PublicKey.PublicKey, ec.S256())
			if err == nil {
				nodeInfo = &proto.Node{
					ID:        proto.NodeID(nodeInfoTypes.ID.NodeID),
					Addr:      nodeInfoTypes.Addr,
					PublicKey: publicKey,
					Nonce: mine.Uint256{
						A: nodeInfoTypes.Nonce.A,
						B: nodeInfoTypes.Nonce.B,
						C: nodeInfoTypes.Nonce.C,
						D: nodeInfoTypes.Nonce.D,
					},
				}
			}
		}

		return err // return from View func
	})
	return
}

// SetPublicKey verifies nonce and set Public Key
func SetPublicKey(id proto.NodeID, nonce mine.Uint256, publicKey *ec.PublicKey) (err error) {
	nodeInfo := &proto.Node{
		ID:        id,
		Addr:      "",
		PublicKey: publicKey,
		Nonce:     nonce,
	}
	return SetNodeInfo(nodeInfo)
}

// SetNodeInfo verifies nonce and sets {proto.Node.ID: proto.Node}
func SetNodeInfo(nodeInfo *proto.Node) (err error) {
	if nodeInfo == nil {
		return ErrNilNode
	}
	keyHash := mine.HashBlock(nodeInfo.PublicKey.SerializeCompressed(), nodeInfo.Nonce)
	id := nodeInfo.ID
	idHash, err := hash.NewHashFromStr(string(id))
	if err != nil {
		return ErrNotValidNodeID
	}
	if !keyHash.IsEqual(idHash) {
		return ErrNodeIDKeyNonceNotMatch
	}

	return setPublicKey(nodeInfo)
}

// setPublicKey sets id and its publicKey
func setPublicKey(nodeInfo *proto.Node) (err error) {
	nodeBuf, err := pb.Marshal(&types.Node{
		ID: &types.NodeID{
			NodeID: string(nodeInfo.ID),
		},
		Addr: nodeInfo.Addr,
		PublicKey: &types.PublicKey{
			PublicKey: nodeInfo.PublicKey.SerializeCompressed(),
		},
		Nonce: &types.Nonce{
			A: nodeInfo.Nonce.A,
			B: nodeInfo.Nonce.B,
			C: nodeInfo.Nonce.C,
			D: nodeInfo.Nonce.D,
		},
	})
	if err == nil {
		return (*bolt.DB)(pks.db).Update(func(tx *bolt.Tx) error {
			bucket := tx.Bucket(pks.bucket)
			if bucket == nil {
				return ErrBucketNotInitialized
			}
			return bucket.Put([]byte(nodeInfo.ID), nodeBuf)
		})
	}
	return
}

// DelNode removes PublicKey to the id
func DelNode(id proto.NodeID) (err error) {
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
