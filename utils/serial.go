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
	"encoding/binary"
	"errors"
	"io"

	"github.com/thunderdb/ThunderDB/crypto/hash"
	"github.com/thunderdb/ThunderDB/proto"
)

const (
	maxPooledBufferLength = hash.HashSize
	maxPooledBufferNumber = 1024
	maxStringLength       = 1 << 20
)

type simpleSerializer chan []byte

var (
	serializer simpleSerializer = make(chan []byte, maxPooledBufferNumber)

	// ErrStringLengthExceed indicates that a string length exceeds limit during deserialization.
	ErrStringLengthExceed = errors.New("string length exceeds limit")

	// ErrInsufficientBuffer indicates that the given buffer space is insufficient during
	// deserialization.
	ErrInsufficientBuffer = errors.New("insufficient buffer space")
)

func (s simpleSerializer) borrowBuffer(len int) []byte {
	if len > maxPooledBufferLength {
		return make([]byte, len)
	}

	select {
	case buffer := <-s:
		return buffer[:len]
	default:
	}

	return make([]byte, len, maxPooledBufferLength)
}

func (s simpleSerializer) returnBuffer(buffer []byte) {
	if cap(buffer) != maxPooledBufferLength {
		return
	}

	select {
	case s <- buffer[:maxPooledBufferLength]:
	default:
	}
}

func (s simpleSerializer) readUint8(r io.Reader) (ret uint8, err error) {
	buffer := s.borrowBuffer(1)
	defer s.returnBuffer(buffer)

	if _, err = io.ReadFull(r, buffer); err == nil {
		ret = buffer[0]
	}

	return
}

func (s simpleSerializer) readUint16(r io.Reader, bo binary.ByteOrder) (ret uint16, err error) {
	buffer := s.borrowBuffer(2)
	defer s.returnBuffer(buffer)

	if _, err = io.ReadFull(r, buffer); err == nil {
		ret = bo.Uint16(buffer)
	}

	return
}

func (s simpleSerializer) readUint32(r io.Reader, bo binary.ByteOrder) (ret uint32, err error) {
	buffer := s.borrowBuffer(4)
	defer s.returnBuffer(buffer)

	if _, err = io.ReadFull(r, buffer); err == nil {
		ret = bo.Uint32(buffer)
	}

	return
}

func (s simpleSerializer) readUint64(r io.Reader, bo binary.ByteOrder) (ret uint64, err error) {
	buffer := s.borrowBuffer(8)
	defer s.returnBuffer(buffer)

	if _, err = io.ReadFull(r, buffer); err == nil {
		ret = bo.Uint64(buffer)
	}

	return
}

func (s simpleSerializer) readString(r io.Reader, bo binary.ByteOrder, ret *string) (err error) {
	lenBuffer := s.borrowBuffer(4)
	defer s.returnBuffer(lenBuffer)

	if _, err = io.ReadFull(r, lenBuffer); err != nil {
		return
	}

	retLen := bo.Uint32(lenBuffer)

	if retLen > maxStringLength {
		err = ErrStringLengthExceed
		return
	}

	strBuffer := s.borrowBuffer(int(retLen))
	defer s.returnBuffer(strBuffer)

	if _, err = io.ReadFull(r, strBuffer); err == nil {
		*ret = string(strBuffer[:])
	}

	return
}

func (s simpleSerializer) readBytes(r io.Reader, lenToRead int, ret []byte) (err error) {
	if len(ret) != lenToRead {
		return ErrInsufficientBuffer
	}

	_, err = io.ReadFull(r, ret)
	return
}

func (s simpleSerializer) putUint8(w io.Writer, val uint8) (err error) {
	buffer := s.borrowBuffer(1)
	defer s.returnBuffer(buffer)

	buffer[0] = val
	_, err = w.Write(buffer)
	return
}

func (s simpleSerializer) putUint16(w io.Writer, bo binary.ByteOrder, val uint16) (err error) {
	buffer := s.borrowBuffer(2)
	defer s.returnBuffer(buffer)

	bo.PutUint16(buffer, val)
	_, err = w.Write(buffer)
	return
}

func (s simpleSerializer) putUint32(w io.Writer, bo binary.ByteOrder, val uint32) (err error) {
	buffer := s.borrowBuffer(4)
	defer s.returnBuffer(buffer)

	bo.PutUint32(buffer, val)
	_, err = w.Write(buffer)
	return
}

func (s simpleSerializer) putUint64(w io.Writer, bo binary.ByteOrder, val uint64) (err error) {
	buffer := s.borrowBuffer(8)
	defer s.returnBuffer(buffer)

	bo.PutUint64(buffer, val)
	_, err = w.Write(buffer)
	return
}

func (s simpleSerializer) putString(w io.Writer, bo binary.ByteOrder, val *string) (err error) {
	buffer := s.borrowBuffer(4 + len(*val))
	defer s.returnBuffer(buffer)

	valLen := uint32(len(*val))
	bo.PutUint32(buffer, valLen)
	copy(buffer[4:], []byte(*val))
	_, err = w.Write(buffer)
	return
}

func (s simpleSerializer) putBytes(w io.Writer, lenToPut int, val []byte) (err error) {
	if len(val) != lenToPut {
		return ErrInsufficientBuffer
	}

	_, err = w.Write(val)
	return
}

func readElement(r io.Reader, bo binary.ByteOrder, element interface{}) (err error) {
	switch e := element.(type) {
	case *bool:
		var ret uint8

		if ret, err = serializer.readUint8(r); err == nil {
			*e = (ret != 0x00)
		}

	case *int8:
		var ret uint8

		if ret, err = serializer.readUint8(r); err == nil {
			*e = int8(ret)
		}

	case *uint8:
		*e, err = serializer.readUint8(r)

	case *int16:
		var ret uint16

		if ret, err = serializer.readUint16(r, bo); err == nil {
			*e = int16(ret)
		}

	case *uint16:
		*e, err = serializer.readUint16(r, bo)

	case *int32:
		var ret uint32

		if ret, err = serializer.readUint32(r, bo); err == nil {
			*e = int32(ret)
		}

	case *uint32:
		*e, err = serializer.readUint32(r, bo)

	case *int64:
		var ret uint64

		if ret, err = serializer.readUint64(r, bo); err == nil {
			*e = int64(ret)
		}

	case *uint64:
		*e, err = serializer.readUint64(r, bo)

	case *string:
		err = serializer.readString(r, bo, e)

	case *proto.NodeID:
		err = serializer.readString(r, bo, (*string)(e))

	case *hash.Hash:
		err = serializer.readBytes(r, hash.HashSize, (*e)[:])

	default:
		return binary.Read(r, bo, element)
	}

	return
}

// ReadElements reads the element list in order from the given reader.
func ReadElements(r io.Reader, bo binary.ByteOrder, elements ...interface{}) (err error) {
	for _, element := range elements {
		if err = readElement(r, bo, element); err != nil {
			break
		}
	}

	return
}

func writeElement(w io.Writer, bo binary.ByteOrder, element interface{}) (err error) {
	switch e := element.(type) {
	case *bool:
		err = serializer.putUint8(w, func() uint8 {
			if *e {
				return uint8(0x01)
			}

			return uint8(0x00)
		}())

	case bool:
		err = serializer.putUint8(w, func() uint8 {
			if e {
				return uint8(0x01)
			}

			return uint8(0x00)
		}())

	case *int8:
		err = serializer.putUint8(w, uint8(*e))

	case int8:
		err = serializer.putUint8(w, uint8(e))

	case *uint8:
		err = serializer.putUint8(w, *e)

	case uint8:
		err = serializer.putUint8(w, e)

	case *int16:
		err = serializer.putUint16(w, bo, uint16(*e))

	case int16:
		err = serializer.putUint16(w, bo, uint16(e))

	case *uint16:
		err = serializer.putUint16(w, bo, *e)

	case uint16:
		err = serializer.putUint16(w, bo, e)

	case *int32:
		err = serializer.putUint32(w, bo, uint32(*e))

	case int32:
		err = serializer.putUint32(w, bo, uint32(e))

	case *uint32:
		err = serializer.putUint32(w, bo, *e)

	case uint32:
		err = serializer.putUint32(w, bo, e)

	case *int64:
		err = serializer.putUint64(w, bo, uint64(*e))

	case int64:
		err = serializer.putUint64(w, bo, uint64(e))

	case *uint64:
		err = serializer.putUint64(w, bo, *e)

	case uint64:
		err = serializer.putUint64(w, bo, e)

	case *string:
		err = serializer.putString(w, bo, e)

	case string:
		err = serializer.putString(w, bo, &e)

	case *proto.NodeID:
		err = serializer.putString(w, bo, (*string)(e))

	case proto.NodeID:
		err = serializer.putString(w, bo, (*string)(&e))

	case *hash.Hash:
		err = serializer.putBytes(w, hash.HashSize, (*e)[:])

	case hash.Hash:
		err = serializer.putBytes(w, hash.HashSize, e[:])

	default:
		return binary.Write(w, bo, element)
	}

	return
}

// WriteElements writes the element list in order to the given writer.
func WriteElements(w io.Writer, bo binary.ByteOrder, elements ...interface{}) (err error) {
	for _, element := range elements {
		if err = writeElement(w, bo, element); err != nil {
			break
		}
	}

	return
}
