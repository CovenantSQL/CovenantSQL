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
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	bolt "github.com/coreos/bbolt"
	"github.com/pkg/errors"

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	mine "github.com/CovenantSQL/CovenantSQL/pow/cpuminer"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

// PublicKeyStore holds db and bucket name.
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
	pksLock sync.Mutex
	// Unittest is a test flag
	Unittest bool
)

var (
	//HACK(auxten): maybe each BP uses distinct key pair is safer

	// BP hold the initial BP info
	BP *conf.BPInfo
)

func init() {
	//HACK(auxten) if we were running go test
	if strings.HasSuffix(os.Args[0], ".test") ||
		strings.HasSuffix(os.Args[0], ".test.exe") ||
		strings.HasPrefix(filepath.Base(os.Args[0]), "___") {
		_, testFile, _, _ := runtime.Caller(0)
		confFile := filepath.Join(filepath.Dir(testFile), "config.yaml")
		log.WithField("conf", confFile).Debug("current test filename")
		log.Debugf("os.Args: %#v", os.Args)

		var err error
		conf.GConf, err = conf.LoadConfig(confFile)
		if err != nil {
			log.WithError(err).Fatal("load config for test in kms failed")
		}
		InitBP()
	}
}

// InitBP initializes kms.BP struct with conf.GConf.
func InitBP() {
	if conf.GConf == nil {
		log.Fatal("must call conf.LoadConfig first")
	}
	BP = conf.GConf.BP

	err := hash.Decode(&BP.RawNodeID.Hash, string(BP.NodeID))
	if err != nil {
		log.WithError(err).Fatal("BP.NodeID error")
	}
}

var (
	// ErrPKSNotInitialized indicates public keystore not initialized
	ErrPKSNotInitialized = errors.New("public keystore not initialized")
	// ErrBucketNotInitialized indicates bucket not initialized
	ErrBucketNotInitialized = errors.New("bucket not initialized")
	// ErrNilNode indicates input node is nil
	ErrNilNode = errors.New("nil node")
	// ErrKeyNotFound indicates key not found
	ErrKeyNotFound = errors.New("key not found")
	// ErrNodeIDKeyNonceNotMatch indicates node id, key, nonce not match
	ErrNodeIDKeyNonceNotMatch = errors.New("nodeID, key, nonce not match")
)

// InitPublicKeyStore opens a db file, if not exist, creates it.
// and creates a bucket if not exist.
func InitPublicKeyStore(dbPath string, initNodes []proto.Node) (err error) {
	//testFlag := flag.Lookup("test")
	//log.Debugf("%#v %#v", testFlag, testFlag.Value)
	pksLock.Lock()
	InitBP()

	var bdb *bolt.DB
	bdb, err = bolt.Open(dbPath, 0600, nil)
	if err != nil {
		log.WithError(err).Error("InitPublicKeyStore failed")
		pksLock.Unlock()
		return
	}

	name := []byte(kmsBucketName)
	err = bdb.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(name); err != nil {
			log.WithError(err).Error("could not create bucket")
			return err
		}
		return nil // return from Update func
	})
	if err != nil {
		log.WithError(err).Error("InitPublicKeyStore failed")
		pksLock.Unlock()
		return
	}

	// pks is the singleton instance
	pks = &PublicKeyStore{
		db:     bdb,
		bucket: name,
	}
	pksLock.Unlock()

	for _, n := range initNodes {
		err = setNode(&n)
		if err != nil {
			err = errors.Wrap(err, "set init nodes failed")
			return
		}
	}

	return
}

// GetPublicKey gets a PublicKey of given id
// Returns an error if the id was not found.
func GetPublicKey(id proto.NodeID) (publicKey *asymmetric.PublicKey, err error) {
	node, err := GetNodeInfo(id)
	if err == nil {
		publicKey = node.PublicKey
	}
	return
}

// GetNodeInfo gets node info of given id
// Returns an error if the id was not found.
func GetNodeInfo(id proto.NodeID) (nodeInfo *proto.Node, err error) {
	pksLock.Lock()
	defer pksLock.Unlock()
	if pks == nil || pks.db == nil {
		return nil, ErrPKSNotInitialized
	}

	err = pks.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(pks.bucket)
		if bucket == nil {
			return ErrBucketNotInitialized
		}
		byteVal := bucket.Get([]byte(id))
		if byteVal == nil {
			return ErrKeyNotFound
		}
		err = utils.DecodeMsgPack(byteVal, &nodeInfo)
		log.Debugf("get node info: %#v", nodeInfo)
		return err // return from View func
	})
	if err != nil {
		err = errors.Wrap(err, "get node info failed")
	}
	return
}

// GetAllNodeID get all node ids exist in store.
func GetAllNodeID() (nodeIDs []proto.NodeID, err error) {
	if pks == nil || pks.db == nil {
		return nil, ErrPKSNotInitialized
	}

	err = pks.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(pks.bucket)
		if bucket == nil {
			return ErrBucketNotInitialized
		}
		err := bucket.ForEach(func(k, v []byte) error {
			nodeIDs = append(nodeIDs, proto.NodeID(k))
			return nil
		})

		return err // return from View func
	})
	if err != nil {
		err = errors.Wrap(err, "get all node id failed")
	}
	return

}

// SetPublicKey verifies nonce and set Public Key.
func SetPublicKey(id proto.NodeID, nonce mine.Uint256, publicKey *asymmetric.PublicKey) (err error) {
	nodeInfo := &proto.Node{
		ID:        id,
		Addr:      "",
		PublicKey: publicKey,
		Nonce:     nonce,
	}
	return SetNode(nodeInfo)
}

// SetNode verifies nonce and sets {proto.Node.ID: proto.Node}.
func SetNode(nodeInfo *proto.Node) (err error) {
	if nodeInfo == nil {
		return ErrNilNode
	}

	// FIXME(ggicci): client node id is a fake one.
	if !Unittest && nodeInfo.Role != proto.Client {
		if !IsIDPubNonceValid(nodeInfo.ID.ToRawNodeID(), &nodeInfo.Nonce, nodeInfo.PublicKey) {
			return ErrNodeIDKeyNonceNotMatch
		}
	}

	return setNode(nodeInfo)
}

// IsIDPubNonceValid returns if `id == HashBlock(key, nonce)`.
func IsIDPubNonceValid(id *proto.RawNodeID, nonce *mine.Uint256, key *asymmetric.PublicKey) bool {
	if key == nil || id == nil || nonce == nil {
		return false
	}
	keyHash := mine.HashBlock(key.Serialize(), *nonce)
	return keyHash.IsEqual(&id.Hash)
}

// setNode sets id and its publicKey.
func setNode(nodeInfo *proto.Node) (err error) {
	pksLock.Lock()
	defer pksLock.Unlock()
	if pks == nil || pks.db == nil {
		return ErrPKSNotInitialized
	}

	nodeBuf, err := utils.EncodeMsgPack(nodeInfo)
	if err != nil {
		err = errors.Wrap(err, "marshal node info failed")
		return
	}
	log.Debugf("set node: %#v", nodeInfo)

	err = pks.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(pks.bucket)
		if bucket == nil {
			return ErrBucketNotInitialized
		}
		return bucket.Put([]byte(nodeInfo.ID), nodeBuf.Bytes())
	})
	if err != nil {
		err = errors.Wrap(err, "get node info failed")
	}

	return
}

// DelNode removes PublicKey to the id.
func DelNode(id proto.NodeID) (err error) {
	pksLock.Lock()
	defer pksLock.Unlock()
	if pks == nil || pks.db == nil {
		return ErrPKSNotInitialized
	}

	err = pks.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(pks.bucket)
		if bucket == nil {
			return ErrBucketNotInitialized
		}
		return bucket.Delete([]byte(id))
	})
	if err != nil {
		err = errors.Wrap(err, "del node failed")
	}
	return
}

// removeBucket this bucket.
func removeBucket() (err error) {
	pksLock.Lock()
	defer pksLock.Unlock()
	if pks != nil {
		err = pks.db.Update(func(tx *bolt.Tx) error {
			return tx.DeleteBucket([]byte(kmsBucketName))
		})
		if err != nil {
			err = errors.Wrap(err, "remove bucket failed")
			return
		}
		// ks.bucket == nil means bucket not exist
		pks.bucket = nil
	}
	return
}

// ResetBucket this bucket.
func ResetBucket() error {
	// cause we are going to reset the bucket, the return of removeBucket
	// is not useful
	removeBucket()
	pksLock.Lock()
	defer pksLock.Unlock()
	bucketName := []byte(kmsBucketName)
	err := pks.db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(bucketName)
		return err
	})
	pks.bucket = bucketName
	if err != nil {
		err = errors.Wrap(err, "reset bucket failed")
	}

	return err
}
