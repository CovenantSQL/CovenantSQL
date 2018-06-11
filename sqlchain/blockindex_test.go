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
	"testing"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/thunderdb/ThunderDB/crypto/hash"
	"github.com/thunderdb/ThunderDB/crypto/signature"
	"github.com/thunderdb/ThunderDB/proto"
)

var (
	testBlocks  []Block
	genesisHash = hash.Hash{
		0xea, 0xf0, 0x2c, 0xa3, 0x48, 0xc5, 0x24, 0xe6,
		0x39, 0x26, 0x55, 0xba, 0x4d, 0x29, 0x60, 0x3c,
		0xd1, 0xa7, 0x34, 0x7d, 0x9d, 0x65, 0xcf, 0xe9,
		0x3c, 0xe1, 0xeb, 0xff, 0xdc, 0xa2, 0x26, 0x94,
	}
)

func produceTestBlocks() (err error) {
	blockNum := 10
	testBlocks = make([]Block, blockNum)

	keyStore := struct {
		name string
		key  []byte
	}{
		name: "Test curve",
		key: []byte{
			0xea, 0xf0, 0x2c, 0xa3, 0x48, 0xc5, 0x24, 0xe6,
			0x39, 0x26, 0x55, 0xba, 0x4d, 0x29, 0x60, 0x3c,
			0xd1, 0xa7, 0x34, 0x7d, 0x9d, 0x65, 0xcf, 0xe9,
			0x3c, 0xe1, 0xeb, 0xff, 0xdc, 0xa2, 0x26, 0x94,
		},
	}

	priv, pub := signature.PrivKeyFromBytes(btcec.S256(), keyStore.key)
	prev := genesisHash

	for index := 0; index < blockNum; index++ {
		header := Header{
			Version: int32(0x01000000),
			Producer: proto.NodeID([]byte{
				0x0, 0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8, 0x9,
				0xa, 0xb, 0xc, 0xd, 0xe, 0xf, 0x0, 0x1, 0x2, 0x3,
			}),
			RootHash:   genesisHash,
			ParentHash: prev,
			MerkleRoot: hash.Hash{
				0xea, 0xf0, 0x2c, 0xa3, 0x48, 0xc5, 0x24, 0xe6,
				0x39, 0x26, 0x55, 0xba, 0x4d, 0x29, 0x60, 0x3c,
				0xd1, 0xa7, 0x34, 0x7d, 0x9d, 0x65, 0xcf, 0xe9,
				0x3c, 0xe1, 0xeb, 0xff, 0xdc, 0xa2, 0x26, 0x94,
			},
			Timestamp: time.Now().Add(time.Duration(index) * time.Hour),
		}

		signedHeader := SignedHeader{
			Header: header,
			BlockHash: hash.Hash{
				0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
				0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
				0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
				0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
			},
			Signee:    nil,
			Signature: nil,
		}

		testBlocks[index].SignedHeader = &signedHeader
		testBlocks[index].SignedHeader.Signee = pub
		err = testBlocks[index].SignHeader(priv)

		if err != nil {
			return err
		}

		prev = signedHeader.BlockHash
	}

	return nil
}

func init() {
	err := produceTestBlocks()

	if err != nil {
		panic(err)
	}
}

func TestNewBlockNode(t *testing.T) {
	parent := newBlockNode(testBlocks[0].SignedHeader, nil)

	if parent == nil {
		t.Fatalf("unexpected result: nil")
	} else if parent.parent != nil {
		t.Fatalf("unexpected parent: %v", parent.parent)
	} else if parent.height != 0 {
		t.Fatalf("unexpected height: %d", parent.height)
	}

	child := newBlockNode(testBlocks[1].SignedHeader, parent)

	if child == nil {
		t.Fatalf("unexpected result: nil")
	} else if child.parent != parent {
		t.Fatalf("unexpected parent: %v", child.parent)
	} else if child.height != parent.height+1 {
		t.Fatalf("unexpected height: %d", child.height)
	}
}

func TestInitBlockNode(t *testing.T) {
	parent := &blockNode{
		parent: nil,
		hash: hash.Hash{
			0xea, 0xf0, 0x2c, 0xa3, 0x48, 0xc5, 0x24, 0xe6,
			0x39, 0x26, 0x55, 0xba, 0x4d, 0x29, 0x60, 0x3c,
			0xd1, 0xa7, 0x34, 0x7d, 0x9d, 0x65, 0xcf, 0xe9,
			0x3c, 0xe1, 0xeb, 0xff, 0xdc, 0xa2, 0x26, 0x94,
		},
		height: -1,
	}

	child := &blockNode{
		parent: nil,
		hash: hash.Hash{
			0xea, 0xf0, 0x2c, 0xa3, 0x48, 0xc5, 0x24, 0xe6,
			0x39, 0x26, 0x55, 0xba, 0x4d, 0x29, 0x60, 0x3c,
			0xd1, 0xa7, 0x34, 0x7d, 0x9d, 0x65, 0xcf, 0xe9,
			0x3c, 0xe1, 0xeb, 0xff, 0xdc, 0xa2, 0x26, 0x94,
		},
		height: -1,
	}

	parent.initBlockNode(testBlocks[0].SignedHeader, nil)

	if parent == nil {
		t.Fatalf("unexpected result: nil")
	} else if parent.parent != nil {
		t.Fatalf("unexpected parent: %v", parent.parent)
	} else if parent.height != 0 {
		t.Fatalf("unexpected height: %d", parent.height)
	}

	child.initBlockNode(testBlocks[1].SignedHeader, parent)

	if child == nil {
		t.Fatalf("unexpected result: nil")
	} else if child.parent != parent {
		t.Fatalf("unexpected parent: %v", child.parent)
	} else if child.height != parent.height+1 {
		t.Fatalf("unexpected height: %d", child.height)
	}
}

func TestAncestor(t *testing.T) {
	cfg := &Config{}
	index := newBlockIndex(cfg)
	parent := (*blockNode)(nil)

	for _, b := range testBlocks {
		bn := newBlockNode(b.SignedHeader, parent)
		index.AddBlock(bn)
		parent = bn
	}

	for i, b := range testBlocks {
		bn := index.LookupNode(&b.SignedHeader.BlockHash)

		if bn == nil {
			t.Fatalf("unexpected loopup result: %v", bn)
		}

		for j := int32(i - 1); j < int32(i+1); j++ {
			a := bn.ancestor(j)

			if j < 0 || j > bn.height {
				if a != nil {
					t.Fatalf("unexpected ancestor: %v", a)
				}
			} else {
				if a.height != j {
					t.Fatalf("unexpected ancestor height: got %d while expecting %d", a.height, j)
				}
			}
		}
	}
}

func TestIndex(t *testing.T) {
	cfg := &Config{}
	index := newBlockIndex(cfg)
	parent := (*blockNode)(nil)

	for _, b := range testBlocks {
		bn := newBlockNode(b.SignedHeader, parent)
		index.AddBlock(bn)
		parent = bn
	}

	for _, b := range testBlocks {
		if !index.HasBlock(&b.SignedHeader.BlockHash) {
			t.Fatalf("unexpected loopup result: %v", false)
		}
	}

	for _, b := range testBlocks {
		bn := index.LookupNode(&b.SignedHeader.BlockHash)

		if bn == nil {
			t.Fatalf("unexpected loopup result: %v", bn)
		}
	}
}
