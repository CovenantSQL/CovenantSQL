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
	"math/big"
	"math/rand"
	"reflect"
	"testing"

	pb "github.com/golang/protobuf/proto"
	"github.com/thunderdb/ThunderDB/crypto/asymmetric"
	"github.com/thunderdb/ThunderDB/crypto/hash"
	pbtypes "github.com/thunderdb/ThunderDB/types"
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
		t.Fatalf("Error occurred: %s", err.Error())
	}

	rState := &State{}
	err = rState.unmarshal(buffer)

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	rand.Read(buffer)
	err = rState.unmarshal(buffer)

	if err != nil {
		t.Logf("Error occurred as expected: %s", err.Error())
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
		t.Fatalf("Error occurred: %s", err.Error())
	}

	err = rState.unmarshal(buffer)

	if err != nil {
		t.Logf("Error occurred as expected: %s", err.Error())
	} else {
		t.Fatal("Unexpected result: returned nil while expecting an error")
	}
}

func TestGenesis(t *testing.T) {
	genesis, err := createRandomBlock(rootHash, true)

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	if err = verifyGenesis(genesis); err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	if err = verifyGenesisHeader(genesis.SignedHeader); err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	if err = verifyGenesis(nil); err != nil {
		t.Logf("Error occurred as expected: %s", err.Error())
	} else {
		t.Fatal("Unexpected result: returned nil while expecting an error")
	}

	if err = verifyGenesisHeader(nil); err != nil {
		t.Logf("Error occurred as expected: %s", err.Error())
	} else {
		t.Fatal("Unexpected result: returned nil while expecting an error")
	}

	// Test non-genesis block
	genesis, err = createRandomBlock(rootHash, false)

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	if err = verifyGenesis(genesis); err != nil {
		t.Logf("Error occurred as expected: %s", err.Error())
	} else {
		t.Fatal("Unexpected result: returned nil while expecting an error")
	}

	if err = verifyGenesisHeader(genesis.SignedHeader); err != nil {
		t.Logf("Error occurred as expected: %s", err.Error())
	} else {
		t.Fatal("Unexpected result: returned nil while expecting an error")
	}

	// Test altered public key block
	genesis, err = createRandomBlock(rootHash, true)

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	_, pub, err := asymmetric.GenSecp256k1KeyPair()

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	genesis.SignedHeader.Signee = (*asymmetric.PublicKey)(pub)

	if err = verifyGenesis(genesis); err != nil {
		t.Logf("Error occurred as expected: %s", err.Error())
	} else {
		t.Fatal("Unexpected result: returned nil while expecting an error")
	}

	if err = verifyGenesisHeader(genesis.SignedHeader); err != nil {
		t.Logf("Error occurred as expected: %s", err.Error())
	} else {
		t.Fatal("Unexpected result: returned nil while expecting an error")
	}

	// Test altered signature
	genesis, err = createRandomBlock(rootHash, true)

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	genesis.SignedHeader.Signature.R.Add(genesis.SignedHeader.Signature.R, big.NewInt(int64(1)))
	genesis.SignedHeader.Signature.S.Add(genesis.SignedHeader.Signature.S, big.NewInt(int64(1)))

	if err = verifyGenesis(genesis); err != nil {
		t.Logf("Error occurred as expected: %s", err.Error())
	} else {
		t.Fatal("Unexpected result: returned nil while expecting an error")
	}

	if err = verifyGenesisHeader(genesis.SignedHeader); err != nil {
		t.Logf("Error occurred as expected: %s", err.Error())
	} else {
		t.Fatal("Unexpected result: returned nil while expecting an error")
	}
}

func TestChain(t *testing.T) {
	fl, err := ioutil.TempFile("", "chain")

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	fl.Close()

	// Create new chain
	genesis, err := createRandomBlock(rootHash, true)

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	chain, err := NewChain(&Config{
		DataDir: fl.Name(),
		Genesis: genesis,
	})

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	t.Logf("Create new chain: genesis hash = %s", genesis.SignedHeader.BlockHash.String())

	// Push blocks
	for block, err := createRandomBlock(
		genesis.SignedHeader.BlockHash, false,
	); err == nil; block, err = createRandomBlock(block.SignedHeader.BlockHash, false) {
		err = chain.PushBlock(block.SignedHeader)

		if err != nil {
			t.Fatalf("Error occurred: %s", err.Error())
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
		t.Fatalf("Error occurred: %s", err.Error())
	}

	// Reload chain from DB file and rebuild memory cache
	chain.db.Close()
	chain, err = LoadChain(&Config{DataDir: fl.Name()})

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}
}
