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

package blockproducer

import (
	"sync"

	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/utils"
)

// State store the node info of chain.
type State struct {
	sync.Mutex
	Node   *blockNode
	Head   hash.Hash
	Height uint32
}

// serialize serializes the state.
func (s *State) serialize() ([]byte, error) {
	buffer, err := utils.EncodeMsgPack(s)
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

// deserialize deserializes the state.
func (s *State) deserialize(b []byte) error {
	err := utils.DecodeMsgPack(b, s)
	return err
}

func (s *State) getNode() *blockNode {
	s.Lock()
	defer s.Unlock()
	return s.Node
}

func (s *State) getHeight() uint32 {
	s.Lock()
	defer s.Unlock()
	return s.Height
}

func (s *State) setHeight(h uint32) {
	s.Lock()
	defer s.Unlock()
	s.Height = h
}

func (s *State) increaseHeightByOne() {
	s.Lock()
	defer s.Unlock()
	s.Height++
}

func (s *State) getHeader() *hash.Hash {
	s.Lock()
	defer s.Unlock()
	return &s.Head
}
