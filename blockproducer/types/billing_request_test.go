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
	"reflect"
	"testing"

	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

var (
	peerNum = 32
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
	_, sign, err := req.SignRequestHeader(priv, false)
	if err != nil || !sign.Verify(req.RequestHash[:], pub) {
		t.Fatalf("signature cannot match the hash and public key: %v", req)
	}
}

func TestBillingRequest_SignRequestHeader2(t *testing.T) {
	header := generateRandomBillingRequestHeader()
	req := &BillingRequest{
		Header: *header,
	}

	priv, _, err := asymmetric.GenSecp256k1KeyPair()
	signee, sign, err := req.SignRequestHeader(priv, true)
	if err != nil || !sign.Verify(req.RequestHash[:], signee) {
		t.Fatalf("signature cannot match the hash and public key: %v", req)
	}
}

func TestBillingRequest_AddSignature(t *testing.T) {
	header := generateRandomBillingRequestHeader()
	req := &BillingRequest{
		Header: *header,
	}

	priv, _, err := asymmetric.GenSecp256k1KeyPair()
	signee, sign, err := req.SignRequestHeader(priv, true)
	if err != nil || !sign.Verify(req.RequestHash[:], signee) {
		t.Fatalf("signature cannot match the hash and public key, req: %v, err: %v", req, err)
	}

	// clear previous signees and signatures
	req.Signees = req.Signees[:0]
	req.Signatures = req.Signatures[:0]

	if err := req.AddSignature(signee, sign, false); err != nil {
		t.Fatalf("add signature failed, req: %v, err: %v", req, err)
	}
}

func TestBillingRequest_AddSignature2(t *testing.T) {
	header := generateRandomBillingRequestHeader()
	req := &BillingRequest{
		Header: *header,
	}

	priv, _, err := asymmetric.GenSecp256k1KeyPair()
	signee, sign, err := req.SignRequestHeader(priv, true)
	if err != nil || !sign.Verify(req.RequestHash[:], signee) {
		t.Fatalf("signature cannot match the hash and public key, req: %v, err: %v", req, err)
	}

	// clear previous signees and signatures
	req.RequestHash = hash.Hash{}
	req.Signees = req.Signees[:0]
	req.Signatures = req.Signatures[:0]

	if err := req.AddSignature(signee, sign, true); err != nil {
		t.Fatalf("add signature failed, req: %v, err: %v", req, err)
	}
}

func TestBillingRequest_AddSignature3(t *testing.T) {
	header := generateRandomBillingRequestHeader()
	req := &BillingRequest{
		Header: *header,
	}

	priv, _, err := asymmetric.GenSecp256k1KeyPair()
	signee, sign, err := req.SignRequestHeader(priv, true)
	if err != nil || !sign.Verify(req.RequestHash[:], signee) {
		t.Fatalf("signature cannot match the hash and public key, req: %v, err: %v", req, err)
	}

	// clear previous signees and signatures
	req.RequestHash = hash.Hash{}
	req.Signees = req.Signees[:0]
	req.Signatures = req.Signatures[:0]

	_, signee, _ = asymmetric.GenSecp256k1KeyPair()
	if err := req.AddSignature(signee, sign, true); err != ErrSignVerification {
		t.Fatalf("add signature should failed, req: %v, err: %v", req, err)
	}
}

func TestBillingRequest_VerifySignatures(t *testing.T) {
	header := generateRandomBillingRequestHeader()
	req := &BillingRequest{
		Header: *header,
	}

	addSignature := func(calcHash bool) {
		priv, _, err := asymmetric.GenSecp256k1KeyPair()
		_, _, err = req.SignRequestHeader(priv, calcHash)
		if err != nil {
			t.Fatalf("sign request failed, req: %v, err: %v", req, err)
		}
	}

	// add 3 signatures
	addSignature(true)
	addSignature(false)
	addSignature(false)

	if err := req.VerifySignatures(); err != nil {
		t.Fatalf("verify signature failed, req: %v, err: %v", req, err)
	}
}

func TestBillingRequest_VerifySignatures2(t *testing.T) {
	header := generateRandomBillingRequestHeader()
	req := &BillingRequest{
		Header: *header,
	}

	addSignature := func(calcHash bool) {
		priv, _, err := asymmetric.GenSecp256k1KeyPair()
		_, _, err = req.SignRequestHeader(priv, calcHash)
		if err != nil {
			t.Fatalf("sign request failed, req: %v, err: %v", req, err)
		}
	}

	// add 3 signatures
	addSignature(true)
	addSignature(false)
	addSignature(false)

	// length invalidation
	req.Signees = req.Signees[:0]

	if err := req.VerifySignatures(); err != ErrSignVerification {
		t.Fatalf("verify should be failed, req: %v, err: %v", req, err)
	}
}

func TestBillingRequest_VerifySignatures3(t *testing.T) {
	header := generateRandomBillingRequestHeader()
	req := &BillingRequest{
		Header: *header,
	}

	addSignature := func(calcHash bool) {
		priv, _, err := asymmetric.GenSecp256k1KeyPair()
		_, _, err = req.SignRequestHeader(priv, calcHash)
		if err != nil {
			t.Fatalf("sign request failed, req: %v, err: %v", req, err)
		}
	}

	// add 3 signatures
	addSignature(true)
	addSignature(false)
	addSignature(false)

	// length invalidation
	req.RequestHash = hash.Hash{}

	if err := req.VerifySignatures(); err != ErrSignVerification {
		t.Fatalf("verify should be failed, req: %v, err: %v", req, err)
	}
}

func TestBillingRequest_VerifySignatures4(t *testing.T) {
	header := generateRandomBillingRequestHeader()
	req := &BillingRequest{
		Header: *header,
	}

	addSignature := func(calcHash bool) {
		priv, _, err := asymmetric.GenSecp256k1KeyPair()
		_, _, err = req.SignRequestHeader(priv, calcHash)
		if err != nil {
			t.Fatalf("sign request failed, req: %v, err: %v", req, err)
		}
	}

	// add 3 signatures
	addSignature(true)
	addSignature(false)
	addSignature(false)

	// length invalidation
	_, req.Signees[0], _ = asymmetric.GenSecp256k1KeyPair()

	if err := req.VerifySignatures(); err == nil || err != ErrSignVerification {
		t.Fatalf("verify should be failed, req: %v, err: %v", req, err)
	}
}

func TestBillingRequest_Compare(t *testing.T) {
	req, _ := generateRandomBillingRequest()

	if err := req.Compare(req); err != nil {
		t.Fatalf("compare failed, req: %v, err: %v", req, err)
	}

	var req2 BillingRequest
	req2 = *req

	req2.Header.LowBlock = hash.Hash{}

	if err := req.Compare(&req2); err != ErrBillingNotMatch {
		t.Fatalf("compare should be failed, req: %v, req2: %v, err: %v", req, req2, err)
	}
}

func TestBillingRequest_Compare2(t *testing.T) {
	req, _ := generateRandomBillingRequest()
	var req2 BillingRequest
	req2 = *req

	var gasAmount proto.AddrAndGas
	gasAmount = *req.Header.GasAmounts[0]
	gasAmount.GasAmount += 10
	req2.Header.GasAmounts = nil
	req2.Header.GasAmounts = append(req2.Header.GasAmounts, &gasAmount)
	req2.Header.GasAmounts = append(req2.Header.GasAmounts, req.Header.GasAmounts[1:]...)

	if err := req.Compare(&req2); err != ErrBillingNotMatch {
		t.Fatalf("compare should be failed, req: %v, req2: %v, err: %v", req, req2, err)
	}
}
