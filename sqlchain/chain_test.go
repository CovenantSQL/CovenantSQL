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

package sqlchain

import (
	"bytes"
	"encoding/hex"
	"io/ioutil"
	"math/rand"
	"reflect"
	"testing"
	"time"

	pb "github.com/golang/protobuf/proto"
	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
	pbtypes "gitlab.com/thunderdb/ThunderDB/types"
)

func TestState(t *testing.T) {
	state := &State{
		node:   nil,
		Head:   hash.Hash{},
		Height: 0,
	}

	rand.Read(state.Head[:])
	buffer, err := state.marshal()

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	rState := &State{}
	err = rState.unmarshal(buffer)

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	err = rState.unmarshal(nil)

	if err != nil {
		t.Logf("Error occurred as expected: %v", err)
	} else {
		t.Fatal("Unexpected result: returned nil while expecting an error")
	}

	if !reflect.DeepEqual(state, rState) {
		t.Fatalf("Values don't match: v1 = %v, v2 = %v", state, rState)
	}

	buffer, err = pb.Marshal(&pbtypes.State{
		Head:   &pbtypes.Hash{Hash: []byte("xxxx")},
		Height: 0,
	})

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	err = rState.unmarshal(buffer)

	if err != nil {
		t.Logf("Error occurred as expected: %v", err)
	} else {
		t.Fatal("Unexpected result: returned nil while expecting an error")
	}
}

func TestIndexKey(t *testing.T) {
	for i := 0; i < 100; i++ {
		b1, err := createRandomBlock(rootHash, false)

		if err != nil {
			t.Fatalf("Error occurred: %v", err)
		}

		b2, err := createRandomBlock(rootHash, false)

		if err != nil {
			t.Fatalf("Error occurred: %v", err)
		}

		// Test partial order
		bi1 := newBlockNode(b1.SignedHeader, nil)
		bi2 := newBlockNode(b2.SignedHeader, nil)
		bi1.height = rand.Int31()
		bi2.height = rand.Int31()
		k1 := bi1.indexKey()
		k2 := bi2.indexKey()

		if c1, c2 := bytes.Compare(k1, k2) < 0, bi1.height < bi2.height; c1 != c2 {
			t.Fatalf("Unexpected compare result: heights=%d,%d keys=%s,%s",
				bi1.height, bi2.height, hex.EncodeToString(k1), hex.EncodeToString(k2))
		}

		if c1, c2 := bytes.Compare(k1, k2) > 0, bi1.height > bi2.height; c1 != c2 {
			t.Fatalf("Unexpected compare result: heights=%d,%d keys=%s,%s",
				bi1.height, bi2.height, hex.EncodeToString(k1), hex.EncodeToString(k2))
		}
	}
}

func TestChain(t *testing.T) {
	fl, err := ioutil.TempFile("", "chain")

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	fl.Close()

	// Create new chain
	genesis, err := createRandomBlock(rootHash, true)

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	chain, err := NewChain(&Config{
		DataDir: fl.Name(),
		Genesis: genesis,
		Period:  300 * time.Second,
	})

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	t.Logf("Create new chain: genesis hash = %s", genesis.SignedHeader.BlockHash.String())

	// Push blocks
	for block, err := createRandomBlock(
		genesis.SignedHeader.BlockHash, false,
	); err == nil; block, err = createRandomBlock(block.SignedHeader.BlockHash, false) {
		err = chain.PushBlock(block.SignedHeader)

		if err != nil {
			t.Fatalf("Error occurred: %v", err)
		}

		t.Logf("Pushed new block: height = %d,  %s <- %s",
			chain.state.Height,
			block.SignedHeader.ParentHash.String(),
			block.SignedHeader.BlockHash.String())

		if chain.state.Height >= testHeight {
			break
		}
	}

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	// Reload chain from DB file and rebuild memory cache
	chain.db.Close()
	chain, err = LoadChain(&Config{
		DataDir: fl.Name(),
		Period:  300 * time.Second,
	})

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}
}
