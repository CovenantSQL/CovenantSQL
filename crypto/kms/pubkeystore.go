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
	"database/sql"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/pkg/errors"

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	mine "github.com/CovenantSQL/CovenantSQL/pow/cpuminer"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	xs "github.com/CovenantSQL/CovenantSQL/xenomint/sqlite"
)

// PublicKeyStore holds db and bucket name.
type PublicKeyStore struct {
	db *xs.SQLite3
}

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

var (
	initTableSQL = `CREATE TABLE IF NOT EXISTS "kms" (
		"id"   TEXT,
		"node" BLOB,
		UNIQUE ("id")
	)`
	deleteAllSQL    = `DELETE FROM "kms"`
	deleteRecordSQL = `DELETE FROM "kms" WHERE "id" = ?`
	setRecordSQL    = `INSERT OR REPLACE INTO "kms" ("id", "node") VALUES(?, ?)`
	getRecordSQL    = `SELECT "node" FROM "kms" WHERE "id" = ? LIMIT 1`
	getAllNodeIDSQL = `SELECT "id" FROM "kms"`
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
	if conf.GConf.BP == nil {
		seedBP := &conf.GConf.SeedBPNodes[0]
		conf.GConf.BP = &conf.BPInfo{
			PublicKey: seedBP.PublicKey,
			NodeID:    seedBP.ID,
			Nonce:     seedBP.Nonce,
		}
	}

	BP = conf.GConf.BP
	err := hash.Decode(&conf.GConf.BP.RawNodeID.Hash, string(conf.GConf.BP.NodeID))
	if err != nil {
		log.WithError(err).Fatal("BP.NodeID error")
	}
}

var (
	// ErrPKSNotInitialized indicates public keystore not initialized
	ErrPKSNotInitialized = errors.New("public keystore not initialized")
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
	// close already opened public key store
	ClosePublicKeyStore()

	pksLock.Lock()
	InitBP()

	var strg *xs.SQLite3

	if strg, err = func() (strg *xs.SQLite3, err error) {
		// test if the keystore is a valid sqlite database
		// if so, truncate and upgrade to new version

		if err = removeFileIfIsNotSQLite(dbPath); err != nil {
			return
		}
		if strg, err = xs.NewSqlite(dbPath); err != nil {
			return
		}
		if _, err = strg.Writer().Exec(initTableSQL); err != nil {
			return
		}

		return
	}(); err != nil {
		pksLock.Unlock()
		log.WithError(err).Error("InitPublicKeyStore failed")
		return
	}

	// pks is the singleton instance
	pks = &PublicKeyStore{
		db: strg,
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

	if err = func() (err error) {
		var rawNodeInfo []byte
		if err = pks.db.Writer().QueryRow(getRecordSQL, string(id)).Scan(&rawNodeInfo); err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				err = ErrKeyNotFound
			}
			return
		}
		err = utils.DecodeMsgPack(rawNodeInfo, &nodeInfo)
		log.Debugf("get node info: %#v", nodeInfo)

		return
	}(); err != nil {
		err = errors.Wrap(err, "get node info failed")
	}
	return
}

// GetAllNodeID get all node ids exist in store.
func GetAllNodeID() (nodeIDs []proto.NodeID, err error) {
	if pks == nil || pks.db == nil {
		return nil, ErrPKSNotInitialized
	}

	if err = func() (err error) {
		var rows *sql.Rows
		if rows, err = pks.db.Writer().Query(getAllNodeIDSQL); err != nil {
			return
		}

		defer rows.Close()

		for rows.Next() {
			var rawNodeID string
			if err = rows.Scan(&rawNodeID); err != nil {
				return
			}

			nodeIDs = append(nodeIDs, proto.NodeID(rawNodeID))
		}

		return
	}(); err != nil {
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
	if !Unittest {
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

	_, err = pks.db.Writer().Exec(setRecordSQL, string(nodeInfo.ID), nodeBuf.Bytes())
	if err != nil {
		err = errors.Wrap(err, "set node info failed")
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

	_, err = pks.db.Writer().Exec(deleteRecordSQL, string(id))
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
		_, err = pks.db.Writer().Exec(deleteAllSQL)
		if err != nil {
			err = errors.Wrap(err, "remove bucket failed")
			return
		}
	}
	return
}

// ResetBucket this bucket.
func ResetBucket() error {
	// cause we are going to reset the bucket, the return of removeBucket
	// is not useful
	return removeBucket()
}

// ClosePublicKeyStore closes the public key store.
func ClosePublicKeyStore() {
	pksLock.Lock()
	defer pksLock.Unlock()
	if pks != nil {
		_ = pks.db.Close()
		pks = nil
	}
}

func removeFileIfIsNotSQLite(filename string) (err error) {
	var (
		f          *os.File
		fileHeader [6]byte
	)
	if f, err = os.Open(filename); err != nil && os.IsNotExist(err) {
		// file not exists
		err = nil
		return
	} else if err != nil {
		// may be no read permission
		return
	}

	if _, err = f.Read(fileHeader[:]); err != nil && errors.Cause(err) != io.EOF {
		// read file failed
		_ = f.Close()
		return
	}

	if string(fileHeader[:]) == "SQLite" {
		// valid sqlite file
		err = nil
		_ = f.Close()
		return
	}

	_ = f.Close()

	// backup and remove file
	bakFile := filename + ".bak"
	if _, err = os.Stat(bakFile); err != nil && os.IsNotExist(err) {
		err = nil
		_ = os.Rename(filename, filename+".bak")
	} else {
		_ = os.Remove(filename)
	}

	return
}
