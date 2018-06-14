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

package utils

import (
	"bytes"
	"encoding/binary"
	"math/rand"
	"reflect"
	"testing"

	"github.com/thunderdb/ThunderDB/crypto/hash"
	"github.com/thunderdb/ThunderDB/proto"
)

type testStruct struct {
	BoolField   bool
	Int8Field   int8
	Uint8Field  uint8
	Int16Field  int16
	Uint16Field uint16
	Int32Field  int32
	Uint32Field uint32
	Int64Field  int64
	Uint64Field uint64
	StringField string
	NodeIDField proto.NodeID
	HashField   hash.Hash
}

func (s *testStruct) randomize() {
	s.BoolField = (rand.Int()%2 == 0)
	s.Int8Field = (int8)(rand.Int())
	s.Uint8Field = (uint8)(rand.Int())
	s.Int16Field = (int16)(rand.Int())
	s.Uint16Field = (uint16)(rand.Int())
	s.Int32Field = rand.Int31()
	s.Uint32Field = rand.Uint32()
	s.Int64Field = rand.Int63()
	s.Uint64Field = rand.Uint64()
	slen1 := rand.Intn(1024)
	sbuf1 := make([]byte, slen1)
	rand.Read(sbuf1)
	s.StringField = string(sbuf1)
	slen2 := rand.Intn(1024)
	sbuf2 := make([]byte, slen2)
	rand.Read(sbuf2)
	s.NodeIDField = proto.NodeID(sbuf2)
	rand.Read(s.HashField[:])
}

func (s *testStruct) MarshalBinary() ([]byte, error) {
	buffer := bytes.NewBuffer(nil)

	if err := WriteElements(buffer, binary.BigEndian,
		s.BoolField,
		s.Int8Field,
		s.Uint8Field,
		s.Int16Field,
		s.Uint16Field,
		s.Int32Field,
		s.Uint32Field,
		s.Int64Field,
		s.Uint64Field,
		s.StringField,
		s.NodeIDField,
		s.HashField,
	); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func (s *testStruct) MarshalBinary2() ([]byte, error) {
	buffer := bytes.NewBuffer(nil)

	if err := WriteElements(buffer, binary.BigEndian,
		&s.BoolField,
		&s.Int8Field,
		&s.Uint8Field,
		&s.Int16Field,
		&s.Uint16Field,
		&s.Int32Field,
		&s.Uint32Field,
		&s.Int64Field,
		&s.Uint64Field,
		&s.StringField,
		&s.NodeIDField,
		&s.HashField,
	); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func (s *testStruct) UnmarshalBinary(b []byte) error {
	reader := bytes.NewReader(b)

	return ReadElements(reader, binary.BigEndian,
		&s.BoolField,
		&s.Int8Field,
		&s.Uint8Field,
		&s.Int16Field,
		&s.Uint16Field,
		&s.Int32Field,
		&s.Uint32Field,
		&s.Int64Field,
		&s.Uint64Field,
		&s.StringField,
		&s.NodeIDField,
		&s.HashField,
	)
}

func TestSerialization(t *testing.T) {
	for i := 0; i < 100; i++ {
		ots := &testStruct{}
		ots.randomize()
		oenc, err := ots.MarshalBinary()

		if err != nil {
			t.Fatalf("Error occurred: %v", err)
		}

		ohash := hash.HashH(oenc)
		rts := &testStruct{}

		if err = rts.UnmarshalBinary(oenc); err != nil {
			t.Fatalf("Error occurred: %v", err)
		}

		if !reflect.DeepEqual(ots, rts) {
			t.Fatalf("Result not match: t1=%+v t2=%+v", ots, rts)
		}

		renc, err := rts.MarshalBinary2()

		if err != nil {
			t.Fatalf("Error occurred: %v", err)
		}

		rhash := hash.HashH(renc)

		if rhash != ohash {
			t.Fatalf("Hash result not match: %s v.s. %s", ohash.String(), rhash.String())
		}
	}
}
