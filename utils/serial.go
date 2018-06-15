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
	"time"

	"github.com/thunderdb/ThunderDB/crypto/asymmetric"
	"github.com/thunderdb/ThunderDB/crypto/hash"
	"github.com/thunderdb/ThunderDB/proto"
)

const (
	pooledBufferLength    = hash.HashSize
	maxPooledBufferNumber = 1024
	maxBufferLength       = 1 << 20
)

// simpleSerializer is just a simple serializer with its own []byte pool, which is done by a
// buffered []byte channel.
type simpleSerializer chan []byte

var (
	serializer simpleSerializer = make(chan []byte, maxPooledBufferNumber)

	// ErrBufferLengthExceedLimit indicates that a string length exceeds limit during
	// deserialization.
	ErrBufferLengthExceedLimit = errors.New("buffer length exceeds limit")

	// ErrInsufficientBuffer indicates that the given buffer space is insufficient during
	// deserialization.
	ErrInsufficientBuffer = errors.New("insufficient buffer space")

	// ErrUnexpectedBufferLength indicates that the given buffer doesn't have length as specified.
	ErrUnexpectedBufferLength = errors.New("unexpected buffer length")
)

func (s simpleSerializer) borrowBuffer(len int) []byte {
	if len > pooledBufferLength {
		return make([]byte, len)
	}

	select {
	case buffer := <-s:
		return buffer[:len]
	default:
	}

	return make([]byte, len, pooledBufferLength)
}

func (s simpleSerializer) returnBuffer(buffer []byte) {
	// This guarantees all the buffers in free list are of the same size pooledBufferLength.
	if cap(buffer) != pooledBufferLength {
		return
	}

	select {
	case s <- buffer:
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

func (s simpleSerializer) readUint16(r io.Reader, order binary.ByteOrder) (ret uint16, err error) {
	buffer := s.borrowBuffer(2)
	defer s.returnBuffer(buffer)

	if _, err = io.ReadFull(r, buffer); err == nil {
		ret = order.Uint16(buffer)
	}

	return
}

func (s simpleSerializer) readUint32(r io.Reader, order binary.ByteOrder) (ret uint32, err error) {
	buffer := s.borrowBuffer(4)
	defer s.returnBuffer(buffer)

	if _, err = io.ReadFull(r, buffer); err == nil {
		ret = order.Uint32(buffer)
	}

	return
}

func (s simpleSerializer) readUint64(r io.Reader, order binary.ByteOrder) (ret uint64, err error) {
	buffer := s.borrowBuffer(8)
	defer s.returnBuffer(buffer)

	if _, err = io.ReadFull(r, buffer); err == nil {
		ret = order.Uint64(buffer)
	}

	return
}

// readString reads string from reader with the following format:
//
// 0     4                                 4+len
// +-----+---------------------------------+
// | len |             string              |
// +-----+---------------------------------+
//
func (s simpleSerializer) readString(r io.Reader, order binary.ByteOrder, ret *string) (err error) {
	lenBuffer := s.borrowBuffer(4)
	defer s.returnBuffer(lenBuffer)

	if _, err = io.ReadFull(r, lenBuffer); err != nil {
		return
	}

	retLen := order.Uint32(lenBuffer)

	if retLen > maxBufferLength {
		err = ErrBufferLengthExceedLimit
		return
	}

	strBuffer := s.borrowBuffer(int(retLen))
	defer s.returnBuffer(strBuffer)

	if _, err = io.ReadFull(r, strBuffer); err == nil {
		*ret = string(strBuffer[:])
	}

	return
}

// readBytes reads bytes from reader with the following format:
//
// 0     4                                 4+len
// +-----+---------------------------------+
// | len |             bytes               |
// +-----+---------------------------------+
//
func (s simpleSerializer) readBytes(r io.Reader, order binary.ByteOrder, ret *[]byte) (err error) {
	lenBuffer := s.borrowBuffer(4)
	defer s.returnBuffer(lenBuffer)

	if _, err = io.ReadFull(r, lenBuffer); err != nil {
		return
	}

	retLen := order.Uint32(lenBuffer)

	if retLen > maxBufferLength {
		err = ErrBufferLengthExceedLimit
		return
	}

	retBuffer := s.borrowBuffer(int(retLen))
	defer s.returnBuffer(retBuffer)

	if _, err = io.ReadFull(r, retBuffer); err == nil {
		if *ret == nil || cap(*ret) < int(retLen) {
			*ret = make([]byte, retLen)
		} else {
			*ret = (*ret)[:retLen]
		}

		copy(*ret, retBuffer)
	}

	return
}

// readFixedSizeBytes reads fixed-size bytes from reader. It's used to read fixed-size array such
// as Hash, which is a [32]byte array.
func (s simpleSerializer) readFixedSizeBytes(r io.Reader, lenToRead int, ret []byte) (err error) {
	if len(ret) != lenToRead {
		return ErrInsufficientBuffer
	}

	_, err = io.ReadFull(r, ret)
	return
}

func (s simpleSerializer) writeUint8(w io.Writer, val uint8) (err error) {
	buffer := s.borrowBuffer(1)
	defer s.returnBuffer(buffer)

	buffer[0] = val
	_, err = w.Write(buffer)
	return
}

func (s simpleSerializer) writeUint16(w io.Writer, order binary.ByteOrder, val uint16) (err error) {
	buffer := s.borrowBuffer(2)
	defer s.returnBuffer(buffer)

	order.PutUint16(buffer, val)
	_, err = w.Write(buffer)
	return
}

func (s simpleSerializer) writeUint32(w io.Writer, order binary.ByteOrder, val uint32) (err error) {
	buffer := s.borrowBuffer(4)
	defer s.returnBuffer(buffer)

	order.PutUint32(buffer, val)
	_, err = w.Write(buffer)
	return
}

func (s simpleSerializer) writeUint64(w io.Writer, order binary.ByteOrder, val uint64) (err error) {
	buffer := s.borrowBuffer(8)
	defer s.returnBuffer(buffer)

	order.PutUint64(buffer, val)
	_, err = w.Write(buffer)
	return
}

// writeString writes string to writer with the following format:
//
//  0     4                                 4+len
// +-----+---------------------------------+
// | len |             string              |
// +-----+---------------------------------+
//
func (s simpleSerializer) writeString(w io.Writer, order binary.ByteOrder, val *string) (err error) {
	buffer := s.borrowBuffer(4 + len(*val))
	defer s.returnBuffer(buffer)

	valLen := uint32(len(*val))
	order.PutUint32(buffer, valLen)
	copy(buffer[4:], []byte(*val))
	_, err = w.Write(buffer)
	return
}

// writeBytes writes bytes to writer with the following format:
//
// 0     4                                 4+len
// +-----+---------------------------------+
// | len |             bytes               |
// +-----+---------------------------------+
//
func (s simpleSerializer) writeBytes(w io.Writer, order binary.ByteOrder, val []byte) (err error) {
	buffer := s.borrowBuffer(4 + len(val))
	defer s.returnBuffer(buffer)

	valLen := uint32(len(val))
	order.PutUint32(buffer, valLen)
	copy(buffer[4:], []byte(val))
	_, err = w.Write(buffer)
	return
}

// writeFixedSizeBytes writes fixed-size bytes to wirter. It's used to write fixed-size array such
// as Hash, which is a [32]byte array.
func (s simpleSerializer) writeFixedSizeBytes(w io.Writer, lenToPut int, val []byte) (err error) {
	if len(val) != lenToPut {
		return ErrUnexpectedBufferLength
	}

	_, err = w.Write(val)
	return
}

func readElement(r io.Reader, order binary.ByteOrder, element interface{}) (err error) {
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

		if ret, err = serializer.readUint16(r, order); err == nil {
			*e = int16(ret)
		}

	case *uint16:
		*e, err = serializer.readUint16(r, order)

	case *int32:
		var ret uint32

		if ret, err = serializer.readUint32(r, order); err == nil {
			*e = int32(ret)
		}

	case *uint32:
		*e, err = serializer.readUint32(r, order)

	case *int64:
		var ret uint64

		if ret, err = serializer.readUint64(r, order); err == nil {
			*e = int64(ret)
		}

	case *uint64:
		*e, err = serializer.readUint64(r, order)

	case *time.Time:
		var ret uint64

		if ret, err = serializer.readUint64(r, order); err == nil {
			*e = time.Unix(0, int64(ret)).UTC()
		}

	case *string:
		err = serializer.readString(r, order, e)

	case *[]byte:
		err = serializer.readBytes(r, order, e)

	case *proto.NodeID:
		err = serializer.readString(r, order, (*string)(e))

	case *hash.Hash:
		err = serializer.readFixedSizeBytes(r, hash.HashSize, (*e)[:])

	case **asymmetric.PublicKey:
		var buffer []byte

		if err = serializer.readBytes(r, order, &buffer); err == nil {
			*e, err = asymmetric.ParsePubKey(buffer)
		}

	case **asymmetric.Signature:
		var buffer []byte

		if err = serializer.readBytes(r, order, &buffer); err == nil {
			*e, err = asymmetric.ParseSignature(buffer)
		}

	default:
		return binary.Read(r, order, element)
	}

	return
}

// ReadElements reads the element list in order from the given reader.
func ReadElements(r io.Reader, order binary.ByteOrder, elements ...interface{}) (err error) {
	for _, element := range elements {
		if err = readElement(r, order, element); err != nil {
			break
		}
	}

	return
}

func writeElement(w io.Writer, order binary.ByteOrder, element interface{}) (err error) {
	switch e := element.(type) {
	case bool:
		err = serializer.writeUint8(w, func() uint8 {
			if e {
				return uint8(0x01)
			}

			return uint8(0x00)
		}())

	case *bool:
		err = serializer.writeUint8(w, func() uint8 {
			if *e {
				return uint8(0x01)
			}

			return uint8(0x00)
		}())

	case int8:
		err = serializer.writeUint8(w, uint8(e))

	case *int8:
		err = serializer.writeUint8(w, uint8(*e))

	case uint8:
		err = serializer.writeUint8(w, e)

	case *uint8:
		err = serializer.writeUint8(w, *e)

	case int16:
		err = serializer.writeUint16(w, order, uint16(e))

	case *int16:
		err = serializer.writeUint16(w, order, uint16(*e))

	case uint16:
		err = serializer.writeUint16(w, order, e)

	case *uint16:
		err = serializer.writeUint16(w, order, *e)

	case int32:
		err = serializer.writeUint32(w, order, uint32(e))

	case *int32:
		err = serializer.writeUint32(w, order, uint32(*e))

	case uint32:
		err = serializer.writeUint32(w, order, e)

	case *uint32:
		err = serializer.writeUint32(w, order, *e)

	case int64:
		err = serializer.writeUint64(w, order, uint64(e))

	case *int64:
		err = serializer.writeUint64(w, order, uint64(*e))

	case uint64:
		err = serializer.writeUint64(w, order, e)

	case *uint64:
		err = serializer.writeUint64(w, order, *e)

	case string:
		err = serializer.writeString(w, order, &e)

	case *string:
		err = serializer.writeString(w, order, e)

	case []byte:
		err = serializer.writeBytes(w, order, e)

	case *[]byte:
		err = serializer.writeBytes(w, order, *e)

	case time.Time:
		err = serializer.writeUint64(w, order, (uint64)(e.UnixNano()))

	case *time.Time:
		err = serializer.writeUint64(w, order, (uint64)(e.UnixNano()))

	case proto.NodeID:
		err = serializer.writeString(w, order, (*string)(&e))

	case *proto.NodeID:
		err = serializer.writeString(w, order, (*string)(e))

	case hash.Hash:
		err = serializer.writeFixedSizeBytes(w, hash.HashSize, e[:])

	case *hash.Hash:
		err = serializer.writeFixedSizeBytes(w, hash.HashSize, (*e)[:])

	case *asymmetric.PublicKey:
		err = serializer.writeBytes(w, order, e.Serialize())

	case **asymmetric.PublicKey:
		err = serializer.writeBytes(w, order, (*e).Serialize())

	case *asymmetric.Signature:
		err = serializer.writeBytes(w, order, e.Serialize())

	case **asymmetric.Signature:
		err = serializer.writeBytes(w, order, (*e).Serialize())

	default:
		return binary.Write(w, order, element)
	}

	return
}

// WriteElements writes the element list in order to the given writer.
func WriteElements(w io.Writer, order binary.ByteOrder, elements ...interface{}) (err error) {
	for _, element := range elements {
		if err = writeElement(w, order, element); err != nil {
			break
		}
	}

	return
}
