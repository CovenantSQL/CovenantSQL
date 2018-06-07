/*
 * Copyright 2018 The ThunderDB Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the “License”);
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an “AS IS” BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package sqlchain

import (
	"io/ioutil"
	"testing"
	"time"

	bolt "github.com/coreos/bbolt"
	"github.com/thunderdb/ThunderDB/common"
	"github.com/thunderdb/ThunderDB/crypto/hash"
)

func TestChain(t *testing.T) {
	fl, err := ioutil.TempFile("", "chain")

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	// Cold init
	chain, err := NewChain(&Config{DataDir: fl.Name()})

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	// Create genesis
	header := Header{
		Version: int32(0x01000000),
		Producer: common.Address{
			0x0, 0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8, 0x9,
			0xa, 0xb, 0xc, 0xd, 0xe, 0xf, 0x0, 0x1, 0x2, 0x3,
		},
		RootHash:   genesisHash,
		ParentHash: genesisHash,
		MerkleRoot: hash.Hash{
			0xea, 0xf0, 0x2c, 0xa3, 0x48, 0xc5, 0x24, 0xe6,
			0x39, 0x26, 0x55, 0xba, 0x4d, 0x29, 0x60, 0x3c,
			0xd1, 0xa7, 0x34, 0x7d, 0x9d, 0x65, 0xcf, 0xe9,
			0x3c, 0xe1, 0xeb, 0xff, 0xdc, 0xa2, 0x26, 0x94,
		},
		Timestamp: time.Now(),
	}

	signedHeader := SignedHeader{
		Header:    header,
		BlockHash: genesisHash,
		Signee:    nil,
		Signature: nil,
	}

	block := Block{
		SignedHeader: &signedHeader,
		Queries:      Queries{},
	}

	chain.state.Head = genesisHash
	chain.state.Height = 0
	chain.state.node = newBlockNode(block.SignedHeader, nil)
	err = chain.db.Update(func(tx *bolt.Tx) (err error) {
		bucket, err := tx.CreateBucketIfNotExists(metaBucket[:])

		if err != nil {
			return err
		}

		bucket.CreateBucketIfNotExists(metaBlockIndexBucket)

		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	// Push blocks
	for _, b := range testBlocks {
		err := chain.PushBlock(b.SignedHeader)

		if err != nil {
			t.Fatalf("Error occurred: %s", err.Error())
		}
	}

	// Rebuild chain
	chain.db.Close()
	_, err = NewChain(&Config{DataDir: fl.Name()})

	if err != nil {
		// FIXME(leventeliu)
		t.Logf("init error: %s", err.Error())
	}
}
