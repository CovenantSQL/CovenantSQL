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

package main

import (
	"encoding/binary"
	"errors"
	"flag"
	"os"

	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	ct "github.com/CovenantSQL/CovenantSQL/sqlchain/types"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/coreos/bbolt"
)

type blockNode struct {
	count  int32
	height int32
	block  *ct.Block
	parent *blockNode
}

type blockIndex map[hash.Hash]*blockNode

var (
	rootHash = &hash.Hash{}
	// Bucket keys
	blockBucket             = []byte("block")
	blockCount2HeightBucket = []byte("block-count-to-height")
	// Errors
	errBlockBucketNotFound = errors.New("block bucket not found")
	// Flags
	path string
)

func init() {
	flag.StringVar(&path, "file", "observer.db", "Data file of observer to be upgraded")
	flag.Parse()
	log.SetLevel(log.DebugLevel)
}

func int32ToBytes(h int32) (data []byte) {
	data = make([]byte, 4)
	binary.BigEndian.PutUint32(data, uint32(h))
	return
}

func bytesToInt32(data []byte) int32 {
	return int32(binary.BigEndian.Uint32(data))
}

func process() (err error) {
	var (
		db  *bolt.DB
		bis = make(map[string]blockIndex)
	)
	if db, err = bolt.Open(path, 0600, nil); err != nil {
		return
	}
	if err = db.View(func(tx *bolt.Tx) (err error) {
		var (
			bk, dbbk *bolt.Bucket
			cur      *bolt.Cursor
		)

		if bk = tx.Bucket(blockBucket); bk == nil {
			err = errBlockBucketNotFound
			return
		}

		cur = bk.Cursor()
		for k, v := cur.First(); k != nil; k, v = cur.Next() {
			// Ensure that the cursor is pointing to a bucket
			if v != nil {
				log.WithFields(log.Fields{
					"k": string(k),
				}).Warn("KV pair detected in block bucket")
				continue
			}

			// Get bucket from key
			if dbbk = cur.Bucket().Bucket(k); dbbk == nil {
				log.WithFields(log.Fields{
					"k": string(k),
				}).Warn("Failed to get bucket")
				continue
			}

			// Start processing a new database bucket
			log.WithFields(log.Fields{
				"database": string(k),
			}).Info("Start processing a new database bucket")
			var (
				fb bool
				bi = make(blockIndex)
			)
			if err = dbbk.ForEach(func(k, v []byte) (err error) {
				if k == nil || v == nil {
					log.Warn("Unexpected nil value of key or value")
					return
				}
				var (
					height       = bytesToInt32(k)
					ok           bool
					block        *ct.Block
					parent, this *blockNode
				)
				if err = utils.DecodeMsgPack(v, &block); err != nil {
					log.WithFields(log.Fields{
						"height": height,
						"block":  block,
					}).WithError(err).Warn("Failed to decode block data")
					// Do not return error and keep `ForEach` running
					err = nil
					return
				}
				if block.ParentHash().IsEqual(rootHash) || !fb {
					fb = true
					// Clean block index -- seems that this db is restarted at some point
					bi = make(blockIndex)
					// Add first block
					this = &blockNode{
						height: height,
						block:  block,
					}
				} else {
					// Add block according to parent
					if parent, ok = bi[block.SignedHeader.ParentHash]; !ok {
						log.WithFields(log.Fields{
							"height": height,
							"block":  block,
						}).Warn("Failed to lookup parent node")
						this = &blockNode{
							// Temporary set to height for unrecoverable chain
							count:  height,
							height: height,
							block:  block,
						}
					} else {
						this = &blockNode{
							count:  parent.count + 1,
							height: height,
							block:  block,
							parent: parent,
						}
					}
				}
				log.WithFields(log.Fields{
					"count":  this.count,
					"height": this.height,
					"parent": this.block.SignedHeader.ParentHash.String(),
					"block":  this.block.SignedHeader.BlockHash.String(),
				}).Info("Add new block to index")
				bi[block.SignedHeader.BlockHash] = this
				return
			}); err != nil {
				return
			}
			bis[string(k)] = bi
		}
		return
	}); err != nil {
		return
	}

	// Rewrite count -> height index
	if err = db.Update(func(tx *bolt.Tx) (err error) {
		var bk, dbbk *bolt.Bucket
		if bk, err = tx.CreateBucketIfNotExists(blockCount2HeightBucket); err != nil {
			return
		}
		for dbid, bi := range bis {
			// Start processing a new database bucket
			log.WithFields(log.Fields{
				"database": dbid,
			}).Info("Start adding count2height index for a new database bucket")
			if dbbk, err = bk.CreateBucketIfNotExists([]byte(dbid)); err != nil {
				return
			}
			for _, v := range bi {
				if err = dbbk.Put(int32ToBytes(v.count), int32ToBytes(v.height)); err != nil {
					return
				}
				log.WithFields(log.Fields{
					"diff":   v.height - v.count,
					"count":  v.count,
					"height": v.height,
					"parent": v.block.SignedHeader.ParentHash.String(),
					"block":  v.block.SignedHeader.BlockHash.String(),
				}).Info("Add new block count2height index")
			}
		}
		return
	}); err != nil {
		return
	}

	return
}

func main() {
	if err := process(); err != nil {
		log.WithError(err).Error("Failed to process data file")
		os.Exit(1)
	}
}
