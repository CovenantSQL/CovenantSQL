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
	"gitlab.com/thunderdb/ThunderDB/crypto/asymmetric"
	hash2 "gitlab.com/thunderdb/ThunderDB/crypto/hash"
	"reflect"
	"testing"
)

var (
	peerNum int32 = 32
)

func TestBillingRequestHeader_MarshalUnmarshalBinary(t *testing.T) {
	reqHeader := generateRandomBillingRequestHeader()
	b, err := reqHeader.MarshalBinary()
	if err != nil {
		t.Fatalf("unexpect error when marshal request header: %v", err)
	}

	newReqHeader := &BillingRequestHeader{}
	err = newReqHeader.UnmarshalBinary(b)
	if err != nil {
		t.Fatalf("unexpect error when unmashll request header: %v", err)
	}

	if !reflect.DeepEqual(reqHeader, newReqHeader) {
		t.Fatalf("values not match:\n\tv0=%+v\n\tv1=%+v", reqHeader, newReqHeader)
	}
}

func TestBillingRequestHeader_PackAndSignRequest(t *testing.T) {
	// Generate key pair
	priv, pub, err := asymmetric.GenSecp256k1KeyPair()
	if err != nil {
		t.Fatalf("generate key pair failed: %v", err)
	}
	reqHeader := generateRandomBillingRequestHeader()
	hash, signature, err := reqHeader.PackAndSignRequestHeader(priv)

	b, err := reqHeader.MarshalBinary()
	if err != nil {
		t.Fatalf("unexpect error when marshal request header: %v", err)
	}
	newHash := hash2.THashH(b)
	if !hash.IsEqual(&newHash) {
		t.Fatalf("values not match: \n\thash0=%+v\n\thash1=%+v", hash, newHash)
	}
	if !signature.Verify(hash[:], pub) {
		t.Fatalf("signature not match the hash and public key")
	}
}
