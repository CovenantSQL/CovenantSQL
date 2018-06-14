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
	"math/big"
	"math/rand"
	"reflect"
	"testing"

	"github.com/thunderdb/ThunderDB/crypto/asymmetric"
)

func TestSign(t *testing.T) {
	block, err := createRandomBlock(rootHash, true)

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	err = block.Verify()

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}
}

func TestSerialization(t *testing.T) {
	block, err := createRandomBlock(rootHash, true)

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	sheader := block.SignedHeader
	buffer, err := sheader.marshal()

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	rSHeader := &SignedHeader{}
	err = rSHeader.unmarshal(buffer)

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	rand.Read(buffer)
	err = rSHeader.unmarshal(buffer)

	if err != nil {
		t.Logf("Error occurred as expected: %v", err)
	} else {
		t.Fatal("Unexpected result: returned nil while expecting an error")
	}

	if !reflect.DeepEqual(sheader.Header, rSHeader.Header) {
		t.Fatalf("Values don't match: v1 = %+v, v2 = %+v", sheader.Header, rSHeader.Header)
	}

	if err = sheader.Verify(); err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	if err = rSHeader.Verify(); err != nil {
		t.Fatalf("Error occurred: %v", err)
	}
}

func TestGenesis(t *testing.T) {
	genesis, err := createRandomBlock(rootHash, true)

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	if err = genesis.VerifyAsGenesis(); err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	if err = genesis.SignedHeader.VerifyAsGenesis(); err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	// Test non-genesis block
	genesis, err = createRandomBlock(rootHash, false)

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	if err = genesis.VerifyAsGenesis(); err != nil {
		t.Logf("Error occurred as expected: %v", err)
	} else {
		t.Fatal("Unexpected result: returned nil while expecting an error")
	}

	if err = genesis.SignedHeader.VerifyAsGenesis(); err != nil {
		t.Logf("Error occurred as expected: %v", err)
	} else {
		t.Fatal("Unexpected result: returned nil while expecting an error")
	}

	// Test altered public key block
	genesis, err = createRandomBlock(rootHash, true)

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	_, pub, err := asymmetric.GenSecp256k1KeyPair()

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	genesis.SignedHeader.Signee = pub

	if err = genesis.VerifyAsGenesis(); err != nil {
		t.Logf("Error occurred as expected: %v", err)
	} else {
		t.Fatal("Unexpected result: returned nil while expecting an error")
	}

	if err = genesis.SignedHeader.VerifyAsGenesis(); err != nil {
		t.Logf("Error occurred as expected: %v", err)
	} else {
		t.Fatal("Unexpected result: returned nil while expecting an error")
	}

	// Test altered signature
	genesis, err = createRandomBlock(rootHash, true)

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	genesis.SignedHeader.Signature.R.Add(genesis.SignedHeader.Signature.R, big.NewInt(int64(1)))
	genesis.SignedHeader.Signature.S.Add(genesis.SignedHeader.Signature.S, big.NewInt(int64(1)))

	if err = genesis.VerifyAsGenesis(); err != nil {
		t.Logf("Error occurred as expected: %v", err)
	} else {
		t.Fatalf("Unexpected error: %v", err)
	}

	if err = genesis.SignedHeader.VerifyAsGenesis(); err != nil {
		t.Logf("Error occurred as expected: %v", err)
	} else {
		t.Fatal("Unexpected result: returned nil while expecting an error")
	}

	// Test nil signature
	genesis.SignedHeader = nil

	if err = genesis.VerifyAsGenesis(); err == ErrNilValue {
		t.Logf("Error occurred as expected: %v", err)
	} else {
		t.Fatalf("Unexpected error: %v", err)
	}
}
