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

package interfaces

import (
	"time"

	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
	"gitlab.com/thunderdb/ThunderDB/proto"
)

type Serializer interface {
	Serialize() ([]byte, error)
}

type Deserializer interface {
	Deserialize(enc []byte) error
}

type Transaction interface {
	Serializer
	Deserializer
	GetDatabaseID() *proto.DatabaseID
	GetHash() hash.Hash
	GetIndexKey() interface{}
	GetPersistenceKey() []byte
	GetSequenceID() uint32
	GetTime() time.Time
	Verify() error
}
