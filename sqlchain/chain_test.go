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

	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
	"gitlab.com/thunderdb/ThunderDB/kayak"
	"gitlab.com/thunderdb/ThunderDB/proto"
)

func TestState(t *testing.T) {
	state := &State{
		node:   nil,
		Head:   hash.Hash{},
		Height: 0,
	}

	rand.Read(state.Head[:])
	buffer, err := state.MarshalBinary()

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	rState := &State{}
	err = rState.UnmarshalBinary(buffer)

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	err = rState.UnmarshalBinary(nil)

	if err != nil {
		t.Logf("Error occurred as expected: %v", err)
	} else {
		t.Fatal("Unexpected result: returned nil while expecting an error")
	}

	if !reflect.DeepEqual(state, rState) {
		t.Fatalf("Values don't match: v1 = %v, v2 = %v", state, rState)
	}
}

func TestIndexKey(t *testing.T) {
	for i := 0; i < 10; i++ {
		b1, err := createRandomBlock(genesisHash, false)

		if err != nil {
			t.Fatalf("Error occurred: %v", err)
		}

		b2, err := createRandomBlock(genesisHash, false)

		if err != nil {
			t.Fatalf("Error occurred: %v", err)
		}

		// Test partial order
		bi1 := newBlockNode(b1, nil)
		bi2 := newBlockNode(b2, nil)
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
	genesis, err := createRandomBlock(genesisHash, true)

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	chain, err := NewChain(&Config{
		DataDir:        fl.Name(),
		Genesis:        genesis,
		Period:         1 * time.Second,
		TimeResolution: 100 * time.Millisecond,
		QueryTTL:       10,
		Server: &kayak.Server{
			ID: proto.NodeID("X1"),
		},
	})

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	t.Logf("Create new chain: genesis = %s, inittime = %s, period = %.9f secs",
		genesis.SignedHeader.BlockHash,
		chain.rt.ChainInitTime.Format(time.RFC3339Nano),
		chain.rt.Period.Seconds())

	// Push blocks
	for {
		chain.isMyTurn = (rand.Intn(10) == 0)
		t.Logf("Chain state: head = %s, height = %d, turn = %d, nextturnstart = %s, ismyturn = %t",
			chain.state.Head, chain.state.Height, chain.rt.NextTurn,
			chain.rt.ChainInitTime.Add(
				chain.cfg.Period*time.Duration(chain.rt.NextTurn)).Format(time.RFC3339Nano),
			chain.isMyTurn)
		acks, err := createRandomQueries(10)

		if err != nil {
			t.Fatalf("Error occurred: %v", err)
		}

		for _, ack := range acks {
			if err = chain.VerifyAndPushAckedQuery(ack); err != nil {
				t.Fatalf("Error occurred: %v", err)
			}
		}

		// Run main cycle
		var now time.Time
		var d time.Duration

		for {
			now, d = chain.rt.TillNextTurn()

			t.Logf("Wake up at: now = %s, d = %.9f secs",
				now.Format(time.RFC3339Nano), d.Seconds())

			if d > 0 {
				time.Sleep(d)
			} else {
				chain.RunCurrentTurn(now)
				break
			}
		}

		// Advise block if it's not my turn
		if !chain.IsMyTurn() {
			block, err := createRandomBlockWithQueries(
				genesis.SignedHeader.BlockHash, chain.state.Head, acks)

			if err != nil {
				t.Fatalf("Error occurred: %v", err)
			}

			if err = chain.CheckAndPushNewBlock(block); err != nil {
				t.Fatalf("Error occurred: %v, block = %+v", err, block)
			}

			t.Logf("Pushed new block: height = %d,  %s <- %s",
				chain.state.Height,
				block.SignedHeader.ParentHash,
				block.SignedHeader.BlockHash)
		} else {
			t.Logf("Produced new block: height = %d,  %s <- %s",
				chain.state.Height,
				chain.rt.pendingBlock.SignedHeader.ParentHash,
				chain.rt.pendingBlock.SignedHeader.BlockHash)
		}

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
		DataDir:        fl.Name(),
		Genesis:        genesis,
		Period:         1 * time.Second,
		TimeResolution: 100 * time.Millisecond,
		QueryTTL:       10,
		Server: &kayak.Server{
			ID: proto.NodeID("X1"),
		},
	})

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}
}
