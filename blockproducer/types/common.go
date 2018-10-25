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
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
)

type marshalHasher interface {
	MarshalHash() ([]byte, error)
}

// DefaultHashSignVerifierImpl defines a default implementation of hashSignVerifier.
type DefaultHashSignVerifierImpl struct {
	Hash      hash.Hash
	Signee    *asymmetric.PublicKey
	Signature *asymmetric.Signature
}

// GetHash implements hashSignVerifier.GetHash.
func (i *DefaultHashSignVerifierImpl) GetHash() hash.Hash {
	return i.Hash
}

// Sign implements hashSignVerifier.Sign.
func (i *DefaultHashSignVerifierImpl) Sign(
	obj marshalHasher, signer *asymmetric.PrivateKey) (err error,
) {
	var enc []byte
	if enc, err = obj.MarshalHash(); err != nil {
		return
	}
	var h = hash.THashH(enc)
	if i.Signature, err = signer.Sign(h[:]); err != nil {
		return
	}
	i.Hash = h
	i.Signee = signer.PubKey()
	return
}

// Verify implements hashSignVerifier.Verify.
func (i *DefaultHashSignVerifierImpl) Verify(obj marshalHasher) (err error) {
	var enc []byte
	if enc, err = obj.MarshalHash(); err != nil {
		return
	}
	var h = hash.THashH(enc)
	if !i.Hash.IsEqual(&h) {
		err = ErrSignVerification
		return
	}
	if !i.Signature.Verify(h[:], i.Signee) {
		err = ErrSignVerification
		return
	}
	return
}
