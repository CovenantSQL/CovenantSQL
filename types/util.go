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
	"crypto/rand"
	"time"

	"github.com/pkg/errors"

	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/crypto/verifier"
	"github.com/CovenantSQL/CovenantSQL/proto"
)

var genesisHash = hash.Hash{}

type canMarshalHash interface {
	MarshalHash() ([]byte, error)
}

func verifyHash(data canMarshalHash, h *hash.Hash) (err error) {
	var newHash hash.Hash
	if err = buildHash(data, &newHash); err != nil {
		return
	}
	if !newHash.IsEqual(h) {
		return errors.Cause(verifier.ErrHashValueNotMatch)
	}
	return
}

func buildHash(data canMarshalHash, h *hash.Hash) (err error) {
	var hashBytes []byte
	if hashBytes, err = data.MarshalHash(); err != nil {
		return
	}
	newHash := hash.THashH(hashBytes)
	copy(h[:], newHash[:])
	return
}

// CreateRandomBlock create a new random block
func CreateRandomBlock(parent hash.Hash, isGenesis bool) (b *Block, err error) {
	// Generate key pair
	priv, _, err := asymmetric.GenSecp256k1KeyPair()

	if err != nil {
		return
	}

	h := hash.Hash{}
	rand.Read(h[:])

	b = &Block{
		SignedHeader: SignedHeader{
			Header: Header{
				Version:     0x01000000,
				Producer:    proto.NodeID(h.String()),
				GenesisHash: genesisHash,
				ParentHash:  parent,
				Timestamp:   time.Now().UTC(),
			},
		},
	}

	if isGenesis {
		emptyNode := &proto.RawNodeID{}
		b.SignedHeader.ParentHash = hash.Hash{}
		b.SignedHeader.GenesisHash = hash.Hash{}
		b.SignedHeader.Producer = emptyNode.ToNodeID()
		b.SignedHeader.MerkleRoot = hash.Hash{}

		err = b.PackAsGenesis()
		return
	}

	err = b.PackAndSignBlock(priv)
	return
}
