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
	"reflect"
	"testing"

	"gitlab.com/thunderdb/ThunderDB/utils"
	"gitlab.com/thunderdb/ThunderDB/utils/log"

	"gitlab.com/thunderdb/ThunderDB/crypto/asymmetric"
	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
)

var (
	peerNum uint32 = 32
)

func TestBillingRequestHeader_MarshalUnmarshalBinary(t *testing.T) {
	reqHeader := generateRandomBillingRequestHeader()
	b, err := utils.EncodeMsgPack(reqHeader)
	if err != nil {
		t.Fatalf("unexpect error when marshal request header: %v", err)
	}

	newReqHeader := &BillingRequestHeader{}
	err = utils.DecodeMsgPack(b.Bytes(), newReqHeader)
	if err != nil {
		t.Fatalf("unexpect error when unmashll request header: %v", err)
	}

	if !reflect.DeepEqual(reqHeader, newReqHeader) {
		t.Fatalf("values not match:\n\tv0=%+v\n\tv1=%+v", reqHeader, newReqHeader)
	}
}

func TestBillingRequest_MarshalUnmarshalBinary(t *testing.T) {
	req, err := generateRandomBillingRequest()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	enc, err := utils.EncodeMsgPack(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	dec := &BillingRequest{}
	err = utils.DecodeMsgPack(enc.Bytes(), dec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !reflect.DeepEqual(req, dec) {
		log.Debug(req)
		log.Debug(dec)
		t.Fatal("values not match")
	}
}

func TestBillingRequest_PackRequestHeader(t *testing.T) {
	req, err := generateRandomBillingRequest()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	enc, err := req.Header.MarshalHash()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	h := hash.THashH(enc)
	if !h.IsEqual(&req.RequestHash) {
		t.Fatalf("hash not matched: \n\tv1=%v\n\tv2=%v", req.RequestHash, h)
	}
}

func TestBillingRequest_SignRequestHeader(t *testing.T) {
	req, err := generateRandomBillingRequest()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	enc, err := req.Header.MarshalHash()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	h := hash.THashH(enc)
	if !h.IsEqual(&req.RequestHash) {
		t.Fatalf("hash not matched: \n\tv1=%v\n\tv2=%v", req.RequestHash, h)
	}

	for i, sign := range req.Signatures {
		if !sign.Verify(req.RequestHash[:], req.Signees[i]) {
			t.Fatalf("signature cannot match the hash and public key: %v", req)
		}
	}

	priv, pub, err := asymmetric.GenSecp256k1KeyPair()
	sign, err := req.SignRequestHeader(priv)
	if !sign.Verify(req.RequestHash[:], pub) {
		t.Fatalf("signature cannot match the hash and public key: %v", req)
	}
	sign, err = req.SignRequestHeader(priv)
	if !sign.Verify(req.RequestHash[:], pub) {
		t.Fatalf("signature cannot match the hash and public key: %v", req)
	}
}

func TestBillingResponse_MarshalUnmarshalBinary(t *testing.T) {
	resp, err := generateRandomBillingResponse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	enc, err := utils.EncodeMsgPack(resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	dec := &BillingResponse{}
	err = utils.DecodeMsgPack(enc.Bytes(), dec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
