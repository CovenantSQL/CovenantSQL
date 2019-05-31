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

package merkle

import (
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
)

// Merkle is a merkle tree implementation (https://en.wikipedia.org/wiki/Merkle_tree).
type Merkle struct {
	tree []*hash.Hash
}

// we will not consider overflow because overflow means the length of slice is larger than 2^63
// Algorithm is from
// https://web.archive.org/web/20180327073507/graphics.stanford.edu/~seander/bithacks.html#RoundUpPowerOf2.
func upperPowOfTwo(n uint64) uint64 {
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n |= n >> 32
	n++
	return n
}

// NewMerkle generate a merkle tree according
// to some hashable values like transactions or blocks.
func NewMerkle(items []*hash.Hash) *Merkle {
	if len(items) == 0 {
		items = []*hash.Hash{{}}
	}

	// the max number of merkle tree node = len(items) * 2 + 2
	upperPoT := upperPowOfTwo(uint64(len(items)))
	maxMerkleSize := upperPoT*2 - 1
	hashArray := make([]*hash.Hash, maxMerkleSize)

	// generate merkle tree
	copy(hashArray, items)
	offset := upperPoT
	for i := uint64(0); i < maxMerkleSize-1; i += 2 {
		if hashArray[i] != nil && hashArray[i+1] != nil {
			hashArray[offset] = MergeTwoHash(hashArray[i], hashArray[i+1])
		} else if hashArray[i] != nil {
			// only left node
			hashArray[offset] = MergeTwoHash(hashArray[i], hashArray[i])
		} else {
			// left and right are both nil
			hashArray[offset] = nil
		}
		offset++
	}
	merkle := &Merkle{hashArray}
	return merkle
}

// GetRoot returns the root of merkle tree.
func (merkle *Merkle) GetRoot() *hash.Hash {
	return merkle.tree[len(merkle.tree)-1]
}

// MergeTwoHash computes the hash of the concatenate of two hash.
func MergeTwoHash(l *hash.Hash, r *hash.Hash) *hash.Hash {
	result := hash.THashH(append(append([]byte{}, (*l)[:]...), (*r)[:]...))
	return &result
}
