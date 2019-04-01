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

package types

import (
	"bytes"
	"encoding"
	"reflect"
	"testing"

	"github.com/pkg/errors"

	"github.com/CovenantSQL/CovenantSQL/crypto/verifier"
	"github.com/CovenantSQL/CovenantSQL/utils"
)

func TestHeader_MarshalUnmarshalBinary(t *testing.T) {

	block, err := generateRandomBlock(genesisHash, false)
	header := &block.SignedHeader.BPHeader
	if err != nil {
		t.Fatalf("failed to generate block: %v", err)
	}

	enc, err := utils.EncodeMsgPack(header)
	if err != nil {
		t.Fatalf("failed to mashal binary: %v", err)
	}

	dec := &BPHeader{}
	err = utils.DecodeMsgPack(enc.Bytes(), dec)
	if err != nil {
		t.Fatalf("failed to unmashal binary: %v", err)
	}

	if !reflect.DeepEqual(header, dec) {
		t.Fatalf("value not math:\n\tv1 = %+v\n\tv2 = %+v", block, dec)
	}
}

func TestSignedHeader_MarshalUnmashalBinary(t *testing.T) {
	block, err := generateRandomBlock(genesisHash, false)
	signedHeader := &block.SignedHeader
	if err != nil {
		t.Fatalf("failed to generate block: %v", err)
	}

	enc, err := utils.EncodeMsgPack(signedHeader)
	if err != nil {
		t.Fatalf("failed to mashal binary: %v", err)
	}

	dec := &BPSignedHeader{}
	err = utils.DecodeMsgPack(enc.Bytes(), dec)
	if err != nil {
		t.Fatalf("failed to unmashal binary: %v", err)
	}

	if !reflect.DeepEqual(signedHeader, dec) {
		t.Fatalf("value not math:\n\tv1 = %+v\n\tv2 = %+v", signedHeader, dec)
	}

}

func TestBlock_MarshalUnmarshalBinary(t *testing.T) {
	block, err := generateRandomBlock(genesisHash, false)
	if err != nil {
		t.Fatalf("failed to generate block: %v", err)
	}
	h := reflect.TypeOf(block)
	_, ok := h.(encoding.BinaryMarshaler)
	if ok {
		t.Log("dec hash BinaryMashaler interface")
	}

	enc, err := utils.EncodeMsgPack(block)
	if err != nil {
		t.Fatalf("failed to mashal binary: %v", err)
	}

	dec := &BPBlock{}
	err = utils.DecodeMsgPack(enc.Bytes(), dec)
	if err != nil {
		t.Fatalf("failed to unmashal binary: %v", err)
	}

	bts1, err := block.MarshalHash()
	if err != nil {
		t.Fatal(err)
	}
	bts2, err := dec.MarshalHash()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bts1, bts2) {
		t.Fatal("hash not stable")
	}
}

func TestBlock_PackAndSignBlock(t *testing.T) {
	block, err := generateRandomBlock(genesisHash, false)
	if err != nil {
		t.Fatalf("failed to generate block: %v", err)
	}

	err = block.verifyMerkleRoot()
	if err != nil {
		t.Fatalf("failed to verify: %v", err)
	}

	err = block.VerifyHash()
	if err != nil {
		t.Fatalf("failed to verify: %v", err)
	}

	err = block.Verify()
	if err != nil {
		t.Fatalf("failed to verify: %v", err)
	}

	block.SignedHeader.DataHash[0]++
	err = block.Verify()
	if errors.Cause(err) != verifier.ErrHashValueNotMatch {
		t.Fatalf("unexpected error: %v", err)
	}
	err = block.VerifyHash()
	if errors.Cause(err) != verifier.ErrHashValueNotMatch {
		t.Fatalf("unexpected error: %v", err)
	}
	err = block.SetHash()
	if err != nil {
		t.Fatalf("failed to set hash: %v", err)
	}
	err = block.VerifyHash()
	if err != nil {
		t.Fatalf("failed to verify: %v", err)
	}

	block.SignedHeader.MerkleRoot[0]++
	err = block.Verify()
	if err != ErrMerkleRootVerification {
		t.Fatalf("unexpected error: %v", err)
	}
	err = block.VerifyHash()
	if err != ErrMerkleRootVerification {
		t.Fatalf("unexpected error: %v", err)
	}
	err = block.verifyMerkleRoot()
	if err != ErrMerkleRootVerification {
		t.Fatalf("unexpected error: %v", err)
	}

	tb, err := generateRandomTransfer()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	block.Transactions = append(block.Transactions, tb)
	err = block.Verify()
	if err != ErrMerkleRootVerification {
		t.Fatalf("unexpected error: %v", err)
	}
}
