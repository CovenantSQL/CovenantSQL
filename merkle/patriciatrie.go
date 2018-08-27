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
	"errors"

	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/tchap/go-patricia/patricia"
)

// Trie is a patricia trie
type Trie struct {
	trie *patricia.Trie
}

// NewPatricia is patricia construction
func NewPatricia() *Trie {
	trie := patricia.NewTrie(patricia.MaxPrefixPerNode(16), patricia.MaxChildrenPerSparseNode(17))
	return &Trie{trie}
}

// Insert serializes key into binary and computes its hash,
// then stores the (hash(key), value) into the trie
func (trie *Trie) Insert(key []byte, value []byte) (inserted bool) {
	hashedKey := hash.HashB(key)

	inserted = trie.trie.Insert(hashedKey, value)
	return
}

// Get returns the value according to the key
func (trie *Trie) Get(key []byte) ([]byte, error) {
	hashedKey := hash.HashB(key)

	rawValue := trie.trie.Get(hashedKey)
	if rawValue == nil {
		return nil, errors.New("no such key")
	}
	value := rawValue.([]byte)

	return value, nil
}
