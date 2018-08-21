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
	"github.com/CovenantSQL/CovenantSQL/utils"
)

func TestTxContent_GetHashAndGetType(t *testing.T) {
	tc, err := generateRandomTxContent()
	if err != nil {
		t.Fatalf("Unexpeted error: %v", err)
	}

	h, err := tc.GetHash()
	if err != nil {
		t.Fatalf("Unexpeted error: %v", err)
	}

	enc, err := tc.MarshalHash()
	encHash := hash.THashH(enc)
	if !h.IsEqual(&encHash) {
		t.Fatalf("Hash not match: \n\tv1=%v,\n\tv2=%v", h, encHash)
	}
}

func TestTxContent_MarshalUnmarshalBinary(t *testing.T) {
	tc, err := generateRandomTxContent()
	if err != nil {
		t.Fatalf("Unexpeted error: %v", err)
	}

	enc, err := utils.EncodeMsgPack(tc)
	if err != nil {
		t.Fatalf("Unexpeted error: %v", err)
	}

	dec := &TxContent{}
	err = utils.DecodeMsgPack(enc.Bytes(), dec)
	if err != nil {
		t.Fatalf("Unexpeted error: %v", err)
	}

	if tc.SequenceID != dec.SequenceID {
		t.Fatalf("Value not match: \n\tv1=%v\n\tv2=%v", tc.SequenceID, tc.SequenceID)
	}
	if tc.BillingRequest.RequestHash != dec.BillingRequest.RequestHash {
		t.Fatalf("Value not match: \n\tv1=%v\n\tv2=%v", tc.BillingRequest.RequestHash, tc.BillingRequest.RequestHash)
	}
	if !tc.BillingRequest.Signatures[0].IsEqual(dec.BillingRequest.Signatures[0]) {
		t.Fatalf("Value not match: \n\tv1=%v\n\tv2=%v", tc.BillingRequest.Signatures[0], dec.BillingRequest.Signatures[0])
	}
	for i := range tc.Receivers {
		if !reflect.DeepEqual(tc.Receivers[i], dec.Receivers[i]) {
			t.Fatalf("Value not match: \n\ttc.Receivers[%d]=%v\n\tReceive[%d]=%v", i, i, tc.Receivers[i], tc.Receivers[0])
		}
		if tc.Rewards[i] != dec.Rewards[i] {
			t.Fatalf("Value not match: \n\ttc.Rewards[%d]=%v\n\tRewards[%d]=%v", i, i, tc.Rewards[i], tc.Rewards[0])
		}
		if tc.Fees[i] != dec.Fees[i] {
			t.Fatalf("Value not match: \n\ttc.Fees[%d]=%v\n\tFees[%d]=%v", i, i, tc.Fees[i], tc.Fees[0])
		}
	}
}

func TestTxBilling_SerializeDeserialize(t *testing.T) {
	tb, err := generateRandomTxBilling()
	if err != nil {
		t.Fatalf("Unexpeted error: %v", err)
	}

	enc, err := tb.Serialize()
	if err != nil {
		t.Fatalf("Unexpeted error: %v", err)
	}

	dec := TxBilling{}
	err = dec.Deserialize(enc)
	if err != nil {
		t.Fatalf("Unexpeted error: %v", err)
	}

	if !tb.SignedBlock.IsEqual(dec.SignedBlock) {
		t.Fatalf("Value not match: \n\tv1=%v\n\tv2=%v", tb.SignedBlock, tb.SignedBlock)
	}
	if !tb.Signature.IsEqual(dec.Signature) {
		t.Fatalf("Value not match: \n\tv1=%v\n\tv2=%v", tb.Signature, tb.Signature)
	}
	if !tb.Signee.IsEqual(dec.Signee) {
		t.Fatalf("Value not match: \n\tv1=%v\n\tv2=%v", tb.Signee, tb.Signee)
	}
	if !tb.TxHash.IsEqual(dec.TxHash) {
		t.Fatalf("Value not match: \n\tv1=%v\n\tv2=%v", tb.TxHash, tb.TxHash)
	}
	if !reflect.DeepEqual(tb.TxContent.BillingResponse, dec.TxContent.BillingResponse) {
		t.Fatalf("Value not match: \n\tv1=%v\n\tv2=%v", tb.TxContent.BillingResponse, tb.TxContent.BillingResponse)
	}
}

func TestTxBilling_GetSet(t *testing.T) {
	tb, err := generateRandomTxBilling()
	if err != nil {
		t.Fatalf("Unexpeted error: %v", err)
	}

	tb.GetSequenceID()
	if tb.GetDatabaseID() == nil {
		t.Fatalf("String should not be nil: %v", tb)
	}

	if !tb.IsSigned() {
		t.Fatalf("BlockHash should not be nil: %v", tb)
	}
	h := generateRandomHash()
	tb.SetSignedBlock(&h)
	if !h.IsEqual(tb.GetSignedBlock()) {
		t.Fatalf("BlockHash should be the same: v1=%s, v2=%s", h.String(), tb.GetSignedBlock().String())
	}
}

func TestTxBilling_PackAndSignTx(t *testing.T) {
	tb, err := generateRandomTxBilling()
	if err != nil {
		t.Fatalf("Unexpeted error: %v", err)
	}

	priv, _, err := asymmetric.GenSecp256k1KeyPair()
	if err != nil {
		t.Fatalf("Unexpeted error: %v", err)
	}
	tb.PackAndSignTx(priv)
	enc, err := tb.TxContent.MarshalHash()
	if err != nil {
		t.Fatalf("Unexpeted error: %v", err)
	}
	h := hash.THashH(enc[:])
	sign, err := priv.Sign(h[:])
	if err != nil {
		t.Fatalf("Unexpeted error: %v", err)
	}
	if !sign.IsEqual(tb.Signature) {
		t.Fatalf("Value not match: \n\tv1=%v\n\tv2=%v", sign, tb.Signature)
	}

	err = tb.Verify()
	if err != nil {
		t.Fatalf("Verify signature failed: %v", err)
	}
}
