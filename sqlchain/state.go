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

package sqlchain

import (
	"bytes"
	"encoding/binary"
	"sync"

	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
	"gitlab.com/thunderdb/ThunderDB/utils"
)

// state represents a snapshot of current best chain.
type state struct {
	mu     sync.Mutex
	node   *blockNode
	Head   hash.Hash
	Height int32
}

// MarshalBinary implements BinaryMarshaler.
func (s *state) MarshalBinary() ([]byte, error) {
	buffer := bytes.NewBuffer(nil)

	if err := utils.WriteElements(buffer, binary.BigEndian,
		s.Head,
		s.Height,
	); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

// UnmarshalBinary implements BinaryUnmarshaler.
func (s *state) UnmarshalBinary(b []byte) (err error) {
	reader := bytes.NewReader(b)
	return utils.ReadElements(reader, binary.BigEndian,
		&s.Head,
		&s.Height,
	)
}
