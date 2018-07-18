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

package types

import (
	"encoding"
	"reflect"
	"testing"

	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
)

func TestHeader_MarshalUnmarshalBinary(t *testing.T) {
	block, err := generateRandomBlock(genesisHash, false)
	header := block.SignedHeader.Header
	if err != nil {
		t.Fatalf("Failed to generate block: %v", err)
	}

	enc, err := header.MarshalBinary()
	if err != nil {
		t.Fatalf("Failed to mashal binary: %v", err)
	}

	dec := Header{}
	err = dec.UnmarshalBinary(enc)
	if err != nil {
		t.Fatalf("Failed to unmashal binary: %v", err)
	}

	if !reflect.DeepEqual(header, dec) {
		t.Fatalf("Value not math:\n\tv1 = %+v\n\tv2 = %+v", block, dec)
	}
}

func TestSignedHeader_MarshalUnmashalBinary(t *testing.T) {
	block, err := generateRandomBlock(genesisHash, false)
	signedHeader := block.SignedHeader
	if err != nil {
		t.Fatalf("Failed to generate block: %v", err)
	}

	enc, err := signedHeader.MarshalBinary()
	if err != nil {
		t.Fatalf("Failed to mashal binary: %v", err)
	}

	dec := SignedHeader{}
	err = dec.UnmarshalBinary(enc)
	if err != nil {
		t.Fatalf("Failed to unmashal binary: %v", err)
	}

	if !reflect.DeepEqual(signedHeader, dec) {
		t.Fatalf("Value not math:\n\tv1 = %+v\n\tv2 = %+v", signedHeader, dec)
	}

}

func TestBlock_MarshalUnmarshalBinary(t *testing.T) {
	block, err := generateRandomBlock(genesisHash, false)
	if err != nil {
		t.Fatalf("Failed to generate block: %v", err)
	}
	h := reflect.TypeOf(block)
	_, ok := h.(encoding.BinaryMarshaler)
	if ok {
		t.Log("dec hash BinaryMashaler interface")
	}

	enc, err := block.MarshalBinary()
	if err != nil {
		t.Fatalf("Failed to mashal binary: %v", err)
	}

	dec := Block{}

	err = dec.UnmarshalBinary(enc)
	if err != nil {
		t.Fatalf("Failed to unmashal binary: %v", err)
	}

	if !reflect.DeepEqual(block.SignedHeader.Header, dec.SignedHeader.Header) {
		t.Fatalf("Value not match:\n\tv1 = %+v\n\tv2 = %+v", block, dec)
	}

	if !block.SignedHeader.Signature.IsEqual(dec.SignedHeader.Signature) {
		t.Fatalf("Value not match:\n\tv1 = %+v\n\tv2 = %+v", block.SignedHeader.Signature, dec.SignedHeader.Signature)
	}

	if !block.SignedHeader.Signee.IsEqual(dec.SignedHeader.Signee) {
		t.Fatalf("Value not match:\n\tv1 = %+v\n\tv2 = %+v", block.SignedHeader.Signee, dec.SignedHeader.Signee)
	}

	if !reflect.DeepEqual(block.Transactions, dec.Transactions) {
		t.Fatalf("Value not match:\n\tv1 = %+v\n\tv2 = %+v", block.Transactions, dec.Transactions)
	}
}

func TestBlock_PackAndSignBlock(t *testing.T) {
	block, err := generateRandomBlock(genesisHash, false)
	if err != nil {
		t.Fatalf("Failed to generate block: %v", err)
	}

	err = block.Verify()
	if err != nil {
		t.Fatalf("Failed to verify: %v", err)
	}

	block.SignedHeader.BlockHash[0]++
	err = block.Verify()
	if err != ErrHashVerification {
		t.Fatalf("Unexpeted error: %v", err)
	}

	h := hash.Hash{}
	block.PushTx(&h)
	err = block.Verify()
	if err != ErrMerkleRootVerification {
		t.Fatalf("Unexpeted error: %v", err)
	}
}
