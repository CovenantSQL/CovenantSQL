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
	"sync"
	"testing"
	"time"

	"gitlab.com/thunderdb/ThunderDB/crypto/asymmetric"
	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
	"gitlab.com/thunderdb/ThunderDB/proto"
)

var (
	testGoRoutines = 10
	testRounds     = 10
)

func init() {
	for i := 0; i < maxPooledBufferNumber; i++ {
		serializer.returnBuffer(make([]byte, pooledBufferLength))
	}
}

type innerStruct struct {
	BoolField   bool
	Int8Field   int8
	Uint8Field  uint8
	Int16Field  int16
	Uint16Field uint16
	Int32Field  int32
	Uint32Field uint32
}

type testStruct struct {
	innerStruct
	Int64Field            int64
	Uint64Field           uint64
	Float64Field          float64
	StringField           string
	BytesField            []byte
	Uint32sField          []uint32
	Uint64sField          []uint64
	TimeField             time.Time
	NodeIDField           proto.NodeID
	DatabaseIDField       proto.DatabaseID
	AddrAndGasField       proto.AddrAndGas
	HashField             hash.Hash
	PublicKeyField        *asymmetric.PublicKey
	SignatureField        *asymmetric.Signature
	StringsField          []string
	HashesField           []*hash.Hash
	AccountAddressesField []*proto.AccountAddress
	DatabaseIDsField      []proto.DatabaseID
	AddrAndGaseseField    []*proto.AddrAndGas
	PublicKeysField       []*asymmetric.PublicKey
	SignaturesField       []*asymmetric.Signature
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
	s.Float64Field = rand.Float64()

	// Randomize StringField
	slen := rand.Intn(2 * pooledBufferLength)
	buff := make([]byte, slen)
	rand.Read(buff)
	s.StringField = string(buff)

	// Randomize BytesField
	slen = rand.Intn(2 * pooledBufferLength)

	if slen == 0 {
		s.BytesField = nil // Watch out, a zero-length slice will is not deep equal to nil
	} else {
		s.BytesField = make([]byte, slen)
		rand.Read(s.BytesField)
	}

	s.TimeField = time.Unix(0, rand.Int63()).UTC()

	// Randomize uint32s field
	slen = rand.Intn(1024)
	if slen == 0 {
		s.Uint32sField = nil
	} else {
		s.Uint32sField = make([]uint32, slen)
	}

	for i := range s.Uint32sField {
		s.Uint32sField[i] = rand.Uint32()
	}

	// Randomize uint64s field
	slen = rand.Intn(1024)
	if slen == 0 {
		s.Uint64sField = nil
	} else {
		s.Uint64sField = make([]uint64, slen)
	}

	for i := range s.Uint64sField {
		s.Uint64sField[i] = rand.Uint64()
	}

	// Randomize NodeIDField
	slen = rand.Intn(2 * pooledBufferLength)
	buff = make([]byte, slen)
	rand.Read(buff)
	s.NodeIDField = proto.NodeID(buff)

	// Randomize DatabaseIDField
	slen = rand.Intn(2 * pooledBufferLength)
	buff = make([]byte, slen)
	rand.Read(buff)
	s.DatabaseIDField = proto.DatabaseID(buff)

	// Randomize hash field
	rand.Read(s.HashField[:])

	// Randomize AddrAndGas field
	s.AddrAndGasField = proto.AddrAndGas{}
	rand.Read(s.AddrAndGasField.AccountAddress[:])
	rand.Read(s.AddrAndGasField.RawNodeID.Hash[:])
	s.AddrAndGasField.GasAmount = rand.Uint64()

	// Randomize PublicKeyField and SignatureField
	priv, pub, err := asymmetric.GenSecp256k1KeyPair()

	if err != nil {
		panic(err)
	}

	s.PublicKeyField = pub
	s.SignatureField, err = priv.Sign(s.HashField[:])

	if err != nil {
		panic(err)
	}

	// Randomize strings field
	slen = rand.Intn(1024)

	if slen == 0 {
		s.StringsField = nil // Watch out, a zero-length slice will is not deep equal to nil
	} else {
		s.StringsField = make([]string, slen)
	}

	for i := range s.StringsField {
		slen = rand.Intn(2 * pooledBufferLength)
		buff = make([]byte, slen)
		rand.Read(buff)
		s.StringsField[i] = string(buff)
	}

	// Randomize hashes field
	slen = rand.Intn(1024)

	if slen == 0 {
		s.HashesField = nil // Watch out, a zero-length slice will is not deep equal to nil
	} else {
		s.HashesField = make([]*hash.Hash, slen)
	}

	for i := range s.HashesField {
		s.HashesField[i] = new(hash.Hash)
		rand.Read(s.HashesField[i][:])
	}

	// Randomize accountAddresses field
	slen = rand.Intn(1024)

	if slen == 0 {
		s.AccountAddressesField = nil // Watch out, a zero-length slice will is not deep equal to nil
	} else {
		s.AccountAddressesField = make([]*proto.AccountAddress, slen)
	}

	for i := range s.AccountAddressesField {
		s.AccountAddressesField[i] = new(proto.AccountAddress)
		rand.Read(s.AccountAddressesField[i][:])
	}

	// Randomize DatabaseIDs field
	slen = rand.Intn(1024)
	if slen == 0 {
		s.DatabaseIDsField = nil
	} else {
		s.DatabaseIDsField = make([]proto.DatabaseID, slen)
	}

	for i := range s.DatabaseIDsField {
		slen = rand.Intn(2 * pooledBufferLength)
		buff = make([]byte, slen)
		rand.Read(buff)
		s.DatabaseIDsField[i] = proto.DatabaseID(string(buff))
	}

	// Randomize AddrAndGases field
	slen = rand.Intn(1024)
	if slen == 0 {
		s.AddrAndGaseseField = nil
	} else {
		s.AddrAndGaseseField = make([]*proto.AddrAndGas, slen)
	}
	for i := range s.AddrAndGaseseField {
		aag := proto.AddrAndGas{}
		rand.Read(aag.AccountAddress[:])
		rand.Read(aag.RawNodeID.Hash[:])
		aag.GasAmount = rand.Uint64()
		s.AddrAndGaseseField[i] = &aag
	}

	// Randomize PublicKeys filed
	slen = rand.Intn(1024)
	if slen == 0 {
		s.PublicKeysField = nil
		s.SignaturesField = nil
	} else {
		s.PublicKeysField = make([]*asymmetric.PublicKey, slen)
		s.SignaturesField = make([]*asymmetric.Signature, slen)
	}

	for i := range s.PublicKeysField {
		priv, pub, err := asymmetric.GenSecp256k1KeyPair()

		if err != nil {
			panic(err)
		}

		s.PublicKeysField[i] = pub
		s.SignaturesField[i], err = priv.Sign(s.HashField[:])

		if err != nil {
			panic(err)
		}
	}

}

func (s *innerStruct) MarshalBinary() ([]byte, error) {
	buffer := bytes.NewBuffer(nil)

	if err := WriteElements(buffer, binary.BigEndian,
		s.BoolField,
		s.Int8Field,
		s.Uint8Field,
		s.Int16Field,
		s.Uint16Field,
		s.Int32Field,
		s.Uint32Field,
	); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func (s *innerStruct) UnmarshalBinary(b []byte) error {
	reader := bytes.NewReader(b)
	return ReadElements(reader, binary.BigEndian,
		&s.BoolField,
		&s.Int8Field,
		&s.Uint8Field,
		&s.Int16Field,
		&s.Uint16Field,
		&s.Int32Field,
		&s.Uint32Field,
	)
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
		s.Float64Field,
		s.StringField,
		s.BytesField,
		s.Uint32sField,
		s.Uint64sField,
		s.TimeField,
		s.NodeIDField,
		s.DatabaseIDField,
		s.AddrAndGasField,
		s.HashField,
		s.PublicKeyField,
		s.SignatureField,
		s.StringsField,
		s.HashesField,
		s.DatabaseIDsField,
		s.AddrAndGaseseField,
		s.PublicKeysField,
		s.SignaturesField,
		s.AccountAddressesField,
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
		&s.Float64Field,
		&s.StringField,
		&s.BytesField,
		&s.Uint32sField,
		&s.Uint64sField,
		&s.TimeField,
		&s.NodeIDField,
		&s.DatabaseIDField,
		&s.AddrAndGasField,
		&s.HashField,
		&s.PublicKeyField,
		&s.SignatureField,
		&s.StringsField,
		&s.HashesField,
		&s.DatabaseIDsField,
		&s.AddrAndGaseseField,
		&s.PublicKeysField,
		&s.SignaturesField,
		&s.AccountAddressesField,
	); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func (s *testStruct) MarshalBinary3() ([]byte, error) {
	buffer := bytes.NewBuffer(nil)

	if err := WriteElements(buffer, binary.BigEndian,
		&s.innerStruct,
		&s.Int64Field,
		&s.Uint64Field,
		&s.Float64Field,
		&s.StringField,
		&s.BytesField,
		&s.Uint32sField,
		&s.Uint64sField,
		&s.TimeField,
		&s.NodeIDField,
		&s.DatabaseIDField,
		&s.AddrAndGasField,
		&s.HashField,
		&s.PublicKeyField,
		&s.SignatureField,
		&s.StringsField,
		&s.HashesField,
		&s.DatabaseIDsField,
		&s.AddrAndGaseseField,
		&s.PublicKeysField,
		&s.SignaturesField,
		&s.AccountAddressesField,
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
		&s.Float64Field,
		&s.StringField,
		&s.BytesField,
		&s.Uint32sField,
		&s.Uint64sField,
		&s.TimeField,
		&s.NodeIDField,
		&s.DatabaseIDField,
		&s.AddrAndGasField,
		&s.HashField,
		&s.PublicKeyField,
		&s.SignatureField,
		&s.StringsField,
		&s.HashesField,
		&s.DatabaseIDsField,
		&s.AddrAndGaseseField,
		&s.PublicKeysField,
		&s.SignaturesField,
		&s.AccountAddressesField,
	)
}

func (s *testStruct) UnmarshalBinary2(b []byte) error {
	reader := bytes.NewReader(b)
	return ReadElements(reader, binary.BigEndian,
		&s.innerStruct,
		&s.Int64Field,
		&s.Uint64Field,
		&s.Float64Field,
		&s.StringField,
		&s.BytesField,
		&s.Uint32sField,
		&s.Uint64sField,
		&s.TimeField,
		&s.NodeIDField,
		&s.DatabaseIDField,
		&s.AddrAndGasField,
		&s.HashField,
		&s.PublicKeyField,
		&s.SignatureField,
		&s.StringsField,
		&s.HashesField,
		&s.DatabaseIDsField,
		&s.AddrAndGaseseField,
		&s.PublicKeysField,
		&s.SignaturesField,
		&s.AccountAddressesField,
	)
}

func TestNullValueSerialization(t *testing.T) {
	ots := &testStruct{}
	// XXX(leventeliu): beware of the zero value flaw -- time.Time zero value (January 1, year 1,
	// 00:00:00.000000000 UTC) is out of range of the int64 (or uint64) Unix time.
	ots.TimeField = time.Unix(0, 0).UTC()
	rts := &testStruct{}

	for i := 0; i < testRounds; i++ {
		oenc, err := ots.MarshalBinary()

		if err != nil {
			t.Fatalf("Error occurred: %v", err)
		}

		ohash := hash.HashH(oenc)

		if err = rts.UnmarshalBinary(oenc); err != nil {
			t.Fatalf("Error occurred: %v", err)
		}

		if !reflect.DeepEqual(ots, rts) {
			t.Fatalf("Result not match: \n\tt1=%+v\n\tt2=%+v", ots, rts)
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

func TestSerialization(t *testing.T) {
	wg := &sync.WaitGroup{}

	for i := 0; i < testGoRoutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			ots := &testStruct{}
			rts := &testStruct{}

			for i := 0; i < testRounds; i++ {
				ots.randomize()

				// Test MarshalBinary/MarshalBinary2 <==> UnmarshalBinary
				oenc, err := ots.MarshalBinary()

				if err != nil {
					t.Errorf("Error occurred: %v", err)
				}

				ohash := hash.HashH(oenc)

				if err = rts.UnmarshalBinary(oenc); err != nil {
					t.Errorf("Error occurred: %v", err)
				}

				if !rts.SignatureField.Verify(rts.HashField[:], rts.PublicKeyField) {
					t.Errorf("Failed to verify signature: hash=%s, sign=%+v, pub=%+v",
						rts.HashField.String(),
						rts.SignatureField,
						rts.PublicKeyField,
					)
				}

				if !reflect.DeepEqual(ots, rts) {
					t.Errorf("Result not match: t1=%+v, t2=%+v", ots.Float64Field, rts.Float64Field)
					t.Errorf("Result not match")
				}

				renc, err := ots.MarshalBinary2()

				if err != nil {
					t.Errorf("Error occurred: %v", err)
				}

				rhash := hash.HashH(renc)

				if rhash != ohash {
					t.Errorf("Hash result not match: %s v.s. %s", ohash.String(), rhash.String())
				}

				// Test MarshalBinary3 <==> UnmarshalBinary2
				oenc, err = ots.MarshalBinary3()

				if err != nil {
					t.Errorf("Error occurred: %v", err)
				}

				ohash = hash.HashH(oenc)

				if err = rts.UnmarshalBinary2(oenc); err != nil {
					t.Errorf("Error occurred: %v", err)
				}

				if !rts.SignatureField.Verify(rts.HashField[:], rts.PublicKeyField) {
					t.Errorf("Failed to verify signature: hash=%s, sign=%+v, pub=%+v",
						rts.HashField.String(),
						rts.SignatureField,
						rts.PublicKeyField,
					)
				}

				if !reflect.DeepEqual(ots, rts) {
					// t.Errorf("Result not match: t1=%+v, t2=%+v", ots, rts)

					t.Errorf("Result not match")
				}

				renc, err = ots.MarshalBinary3()

				if err != nil {
					t.Errorf("Error occurred: %v", err)
				}

				rhash = hash.HashH(renc)

				if rhash != ohash {
					t.Errorf("Hash result not match: %s v.s. %s", ohash.String(), rhash.String())
				}
			}
		}()
	}

	wg.Wait()
}

func TestLengthExceedLimitError(t *testing.T) {
	writeDummyLength := func(l uint32) ([]byte, error) {
		buffer := bytes.NewBuffer(nil)

		if err := WriteElements(buffer, binary.BigEndian, l); err != nil {
			return nil, err
		}

		return buffer.Bytes(), nil
	}

	var buffer []byte
	var err error

	buffer, err = writeDummyLength(maxSliceLength + 1)
	err = func(b []byte) error {
		reader := bytes.NewReader(b)
		return ReadElements(reader, binary.BigEndian, &[]string{})
	}(buffer)

	if err == ErrSliceLengthExceedLimit {
		t.Logf("Error occurred as expected: %v", err)
	} else {
		t.Fatalf("Unexpected error type: %v", err)
	}

	buffer, err = writeDummyLength(maxBufferLength + 1)
	err = func(b []byte) error {
		reader := bytes.NewReader(b)
		return ReadElements(reader, binary.BigEndian, &[]byte{})
	}(buffer)

	if err == ErrBufferLengthExceedLimit {
		t.Logf("Error occurred as expected: %v", err)
	} else {
		t.Fatalf("Unexpected error type: %v", err)
	}
}

func BenchmarkMarshalBinary(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		st := &testStruct{}
		st.randomize()
		b.StartTimer()

		if _, err := st.MarshalBinary(); err != nil {
			b.Fatalf("Error occurred: %v", err)
		}

		b.StopTimer()
	}
}

func BenchmarkMarshalBinary2(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		st := &testStruct{}
		st.randomize()
		b.StartTimer()

		if _, err := st.MarshalBinary2(); err != nil {
			b.Fatalf("Error occurred: %v", err)
		}

		b.StopTimer()
	}
}

func BenchmarkMarshalBinary3(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		st := &testStruct{}
		st.randomize()
		b.StartTimer()

		if _, err := st.MarshalBinary3(); err != nil {
			b.Fatalf("Error occurred: %v", err)
		}

		b.StopTimer()
	}
}

func BenchmarkUnmarshalBinary(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		st := &testStruct{}
		st.randomize()
		enc, err := st.MarshalBinary2()

		if err != nil {
			b.Fatalf("Error occurred: %v", err)
		}

		b.StartTimer()

		if err = st.UnmarshalBinary(enc); err != nil {
			b.Fatalf("Error occurred: %v", err)
		}

		b.StopTimer()
	}
}

func BenchmarkUnmarshalBinary2(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		st := &testStruct{}
		st.randomize()
		enc, err := st.MarshalBinary3()

		if err != nil {
			b.Fatalf("Error occurred: %v", err)
		}

		b.StartTimer()

		if err = st.UnmarshalBinary2(enc); err != nil {
			b.Fatalf("Error occurred: %v", err)
		}

		b.StopTimer()
	}
}
