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

	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/proto"
)

//go:generate hsp

// BillingRequestHeader includes contents that need to be signed. Billing blocks should be within
// height range [low, high] (inclusive).
type BillingRequestHeader struct {
	DatabaseID proto.DatabaseID
	// sqlchain block hash and its height
	LowBlock   hash.Hash
	LowHeight  int32
	HighBlock  hash.Hash
	HighHeight int32
	GasAmounts []*proto.AddrAndGas
}

// BillingRequest defines periodically Billing sync.
type BillingRequest struct {
	Header      BillingRequestHeader
	RequestHash hash.Hash
	Signees     []*asymmetric.PublicKey
	Signatures  []*asymmetric.Signature
}

// PackRequestHeader computes the hash of header.
func (br *BillingRequest) PackRequestHeader() (h *hash.Hash, err error) {
	var enc []byte
	if enc, err = br.Header.MarshalHash(); err != nil {
		return
	}

	br.RequestHash = hash.THashH(enc)
	h = &br.RequestHash
	return
}

// SignRequestHeader first computes the hash of BillingRequestHeader, then signs the request.
func (br *BillingRequest) SignRequestHeader(signer *asymmetric.PrivateKey, calcHash bool) (
	signee *asymmetric.PublicKey, signature *asymmetric.Signature, err error) {
	if calcHash {
		if _, err = br.PackRequestHeader(); err != nil {
			return
		}
	}

	if signature, err = signer.Sign(br.RequestHash[:]); err == nil {
		// append to current signatures
		signee = signer.PubKey()
		br.Signees = append(br.Signees, signee)
		br.Signatures = append(br.Signatures, signature)
	}

	return
}

// AddSignature add existing signature to BillingRequest, requires the structure to be packed first.
func (br *BillingRequest) AddSignature(
	signee *asymmetric.PublicKey, signature *asymmetric.Signature, calcHash bool) (err error) {
	if calcHash {
		if _, err = br.PackRequestHeader(); err != nil {
			return
		}
	}

	if !signature.Verify(br.RequestHash[:], signee) {
		err = ErrSignVerification
		return
	}

	// append
	br.Signees = append(br.Signees, signee)
	br.Signatures = append(br.Signatures, signature)

	return
}

// VerifySignatures verify existing signatures.
func (br *BillingRequest) VerifySignatures() (err error) {
	if len(br.Signees) != len(br.Signatures) {
		return ErrSignVerification
	}

	var enc []byte
	if enc, err = br.Header.MarshalHash(); err != nil {
		return
	}

	h := hash.THashH(enc)
	if !br.RequestHash.IsEqual(&h) {
		return ErrSignVerification
	}

	if len(br.Signees) == 0 {
		return
	}

	for idx, signee := range br.Signees {
		if !br.Signatures[idx].Verify(br.RequestHash[:], signee) {
			return ErrSignVerification
		}
	}

	return
}

// Compare returns if two billing records are identical.
func (br *BillingRequest) Compare(r *BillingRequest) (err error) {
	if !br.Header.LowBlock.IsEqual(&r.Header.LowBlock) ||
		!br.Header.HighBlock.IsEqual(&br.Header.HighBlock) {
		err = ErrBillingNotMatch
		return
	}

	reqMap := make(map[proto.AccountAddress]*proto.AddrAndGas)
	locMap := make(map[proto.AccountAddress]*proto.AddrAndGas)

	for _, v := range br.Header.GasAmounts {
		reqMap[v.AccountAddress] = v
	}

	for _, v := range r.Header.GasAmounts {
		locMap[v.AccountAddress] = v
	}

	if !reflect.DeepEqual(reqMap, locMap) {
		err = ErrBillingNotMatch
		return
	}

	return
}
