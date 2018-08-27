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

package types

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/utils"
)

func TestSignAndVerify(t *testing.T) {
	block, err := createRandomBlock(genesisHash, true)

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	if err = block.Verify(); err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	block.SignedHeader.BlockHash[0]++

	if err = block.Verify(); err != ErrHashVerification {
		t.Fatalf("Unexpected error: %v", err)
	}

	h := &hash.Hash{}
	block.PushAckedQuery(h)

	if err = block.Verify(); err != ErrMerkleRootVerification {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestHeaderMarshalUnmarshaler(t *testing.T) {
	block, err := createRandomBlock(genesisHash, false)

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	origin := &block.SignedHeader.Header
	enc, err := utils.EncodeMsgPack(origin)

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	dec := &Header{}
	if err = utils.DecodeMsgPack(enc.Bytes(), dec); err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	if !reflect.DeepEqual(origin, dec) {
		t.Fatalf("Values don't match:\n\tv1 = %+v\n\tv2 = %+v", origin, dec)
	}
}

func TestSignedHeaderMarshaleUnmarshaler(t *testing.T) {
	block, err := createRandomBlock(genesisHash, true)

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	origin := &block.SignedHeader
	enc, err := utils.EncodeMsgPack(origin)

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	dec := &SignedHeader{}

	if err = utils.DecodeMsgPack(enc.Bytes(), dec); err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	if !reflect.DeepEqual(origin.Header, dec.Header) {
		t.Fatalf("Values don't match:\n\tv1 = %+v\n\tv2 = %+v", origin.Header, dec.Header)
	}

	if err = origin.Verify(); err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	if err = dec.Verify(); err != nil {
		t.Fatalf("Error occurred: %v", err)
	}
}

func TestBlockMarshalUnmarshaler(t *testing.T) {
	origin, err := createRandomBlock(genesisHash, false)

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	enc, err := utils.EncodeMsgPack(origin)

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	dec := &Block{}

	if err = utils.DecodeMsgPack(enc.Bytes(), dec); err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	if !reflect.DeepEqual(origin, dec) {
		t.Fatalf("Values don't match:\n\tv1 = %+v\n\tv2 = %+v", origin, dec)
	}
}

func TestGenesis(t *testing.T) {
	genesis, err := createRandomBlock(genesisHash, true)

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
	genesis, err = createRandomBlock(genesisHash, false)

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
	genesis, err = createRandomBlock(genesisHash, true)

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
	genesis, err = createRandomBlock(genesisHash, true)

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
}
