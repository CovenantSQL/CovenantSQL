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

package main

import (
	"encoding"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"reflect"
	"time"

	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

var (
	// ErrInvalidType defines invalid type.
	ErrInvalidType = errors.New("invalid type")
)

const (
	pooledBufferLength    = hash.HashSize
	maxPooledBufferNumber = 1024
	maxSliceLength        = 4 << 10
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

	// ErrSliceLengthExceedLimit indicates that a slice length exceeds limit during
	// deserialization.
	ErrSliceLengthExceedLimit = errors.New("slice length exceeds limit")

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

func (s simpleSerializer) readFloat64(r io.Reader, order binary.ByteOrder) (ret float64, err error) {
	buffer := s.borrowBuffer(8)
	defer s.returnBuffer(buffer)

	if _, err = io.ReadFull(r, buffer); err == nil {
		val := order.Uint64(buffer)
		ret = math.Float64frombits(val)
	}

	return
}

func (s simpleSerializer) readPublicKey(r io.Reader, order binary.ByteOrder) (val *asymmetric.PublicKey, err error) {
	var buffer []byte

	if err = serializer.readBytes(r, order, &buffer); err == nil && len(buffer) > 0 {
		val, err = asymmetric.ParsePubKey(buffer)
	} else {
		val = nil
	}
	return
}

func (s simpleSerializer) readPublicKeys(r io.Reader, order binary.ByteOrder, val *[]*asymmetric.PublicKey) (err error) {
	lenBuffer := s.borrowBuffer(4)
	defer s.returnBuffer(lenBuffer)

	if _, err = io.ReadFull(r, lenBuffer); err != nil {
		return
	}

	retLen := order.Uint32(lenBuffer)

	if retLen > maxBufferLength {
		err = ErrBufferLengthExceedLimit
		return
	} else if retLen == 0 {
		// Always return nil slice for a zero-length
		*val = nil
		return
	}

	if *val == nil || cap(*val) < int(retLen) {
		*val = make([]*asymmetric.PublicKey, retLen)
	} else {
		*val = (*val)[:retLen]
	}

	for i := range *val {
		(*val)[i], err = s.readPublicKey(r, order)
		if err != nil {
			return
		}
	}

	return
}

func (s simpleSerializer) readSignature(r io.Reader, order binary.ByteOrder) (val *asymmetric.Signature, err error) {
	var buffer []byte

	if err = serializer.readBytes(r, order, &buffer); err == nil && len(buffer) > 0 {
		val, err = asymmetric.ParseSignature(buffer)
	} else {
		val = nil
	}
	return
}

func (s simpleSerializer) readSignatures(r io.Reader, order binary.ByteOrder, val *[]*asymmetric.Signature) (err error) {
	lenBuffer := s.borrowBuffer(4)
	defer s.returnBuffer(lenBuffer)

	if _, err = io.ReadFull(r, lenBuffer); err != nil {
		return
	}

	retLen := order.Uint32(lenBuffer)

	if retLen > maxBufferLength {
		err = ErrBufferLengthExceedLimit
		return
	} else if retLen == 0 {
		// Always return nil slice for a zero-length
		*val = nil
		return
	}

	if *val == nil || cap(*val) < int(retLen) {
		*val = make([]*asymmetric.Signature, retLen)
	} else {
		*val = (*val)[:retLen]
	}

	for i := range *val {
		(*val)[i], err = s.readSignature(r, order)
	}

	return
}

// readAddrAndGas reads AddrAndGas from reader with the following format:
//
// 0             32             64          72
// +--------------+--------------+----------+
// |     hash     |     hash     |  uint64  |
// +--------------+--------------+----------+
//
func (s simpleSerializer) readAddrAndGas(r io.Reader, order binary.ByteOrder, ret *proto.AddrAndGas) (err error) {
	addr := hash.Hash{}
	err = s.readFixedSizeBytes(r, hash.HashSize, addr[:])
	if err != nil {
		return err
	}

	nodeID := hash.Hash{}
	err = s.readFixedSizeBytes(r, hash.HashSize, nodeID[:])
	if err != nil {
		return err
	}

	gasAmount, err := s.readUint64(r, order)
	if err != nil {
		return err
	}

	ret.GasAmount = gasAmount
	ret.RawNodeID = proto.RawNodeID{Hash: nodeID}
	ret.AccountAddress = proto.AccountAddress(addr)

	return nil
}

// readString reads string from reader with the following format:
//
// 0     4                                 4+len
// +-----+---------------------------------+
// | len |             string              |
// +-----+---------------------------------+
//
func (s simpleSerializer) readString(r io.Reader, order binary.ByteOrder, ret *string) (
	err error) {
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
	} else if retLen == 0 {
		// Always return nil slice for a zero-length
		*ret = nil
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

// readUint32s reads bytes from reader with the following format:
//
// 0     4                                 4+len
// +-----+---------------------------------+
// | len |             uint32s             |
// +-----+---------------------------------+
//
func (s simpleSerializer) readUint32s(r io.Reader, order binary.ByteOrder, ret *[]uint32) (err error) {
	lenBuffer := s.borrowBuffer(4)
	defer s.returnBuffer(lenBuffer)

	if _, err = io.ReadFull(r, lenBuffer); err != nil {
		return
	}

	retLen := order.Uint32(lenBuffer)

	if retLen > maxBufferLength {
		err = ErrBufferLengthExceedLimit
		return
	} else if retLen == 0 {
		// Always return nil slice for a zero-length
		*ret = nil
		return
	}

	retBuffer := s.borrowBuffer(int(retLen) * 4)
	defer s.returnBuffer(retBuffer)

	if _, err = io.ReadFull(r, retBuffer); err == nil {
		if *ret == nil || cap(*ret) < int(retLen) {
			*ret = make([]uint32, retLen)
		} else {
			*ret = (*ret)[:retLen]
		}

		for i := range *ret {
			(*ret)[i] = order.Uint32(retBuffer[i*4 : i*4+4])
		}
	}

	return
}

// readUint64s reads bytes from reader with the following format:
//
// 0     4                                 4+len
// +-----+---------------------------------+
// | len |             uint64s             |
// +-----+---------------------------------+
//
func (s simpleSerializer) readUint64s(r io.Reader, order binary.ByteOrder, ret *[]uint64) (err error) {
	lenBuffer := s.borrowBuffer(4)
	defer s.returnBuffer(lenBuffer)

	if _, err = io.ReadFull(r, lenBuffer); err != nil {
		return
	}

	retLen := order.Uint32(lenBuffer)

	if retLen > maxBufferLength {
		err = ErrBufferLengthExceedLimit
		return
	} else if retLen == 0 {
		// Always return nil slice for a zero-length
		*ret = nil
		return
	}

	retBuffer := s.borrowBuffer(int(retLen) * 8)
	defer s.returnBuffer(retBuffer)

	if _, err = io.ReadFull(r, retBuffer); err == nil {
		if *ret == nil || cap(*ret) < int(retLen) {
			*ret = make([]uint64, retLen)
		} else {
			*ret = (*ret)[:retLen]
		}

		for i := range *ret {
			(*ret)[i] = order.Uint64(retBuffer[i*8 : i*8+8])
		}
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

// readStrings reads strings from reader with the following format:
//
// 0          4       8          8+len_0
// +----------+-------+----------+---------+-------+----------+
// | sliceLen | len_0 | string_0 |   ...   | len_n | string_n |
// +----------+-------+----------+---------+-------+----------+
//
func (s simpleSerializer) readStrings(r io.Reader, order binary.ByteOrder, ret *[]string) (
	err error) {
	var retLen uint32

	if retLen, err = s.readUint32(r, order); err != nil {
		return
	}

	if retLen > maxSliceLength {
		err = ErrSliceLengthExceedLimit
		return
	} else if retLen == 0 {
		// Always return nil slice for a zero-length
		*ret = nil
		return
	}

	if *ret == nil || cap(*ret) < int(retLen) {
		*ret = make([]string, retLen)
	} else {
		*ret = (*ret)[:retLen]
	}

	for i := range *ret {
		if err = s.readString(r, order, &((*ret)[i])); err != nil {
			break
		}
	}

	return
}

// readDatabaseIDs reads databaseIDs from reader with the following format:
//
// 0          4       8          8+len_0
// +----------+-------+--------------+---------+-------+--------------+
// | sliceLen | len_0 | databaseID_0 |   ...   | len_n | databaseID_n |
// +----------+-------+--------------+---------+-------+--------------+
//
func (s simpleSerializer) readDatabaseIDs(r io.Reader, order binary.ByteOrder, ret *[]proto.DatabaseID) (
	err error) {
	var retLen uint32

	if retLen, err = s.readUint32(r, order); err != nil {
		return
	}

	if retLen > maxSliceLength {
		err = ErrSliceLengthExceedLimit
		return
	} else if retLen == 0 {
		// Always return nil slice for a zero-length
		*ret = nil
		return
	}

	if *ret == nil || cap(*ret) < int(retLen) {
		*ret = make([]proto.DatabaseID, retLen)
	} else {
		*ret = (*ret)[:retLen]
	}

	for i := range *ret {
		if err = s.readString(r, order, (*string)(&((*ret)[i]))); err != nil {
			break
		}
	}

	return
}

// readAddrAndGases reads hashes from reader with the following format:
//
// 0          4
// +----------+----------------+---------------+-------+----------------+
// | sliceLen |  AddrAndGas_0  |  AddrAndGas_1 |  ...  |  AddrAndGas_2  |
// +----------+----------------+---------------+-------+----------------+
//
func (s simpleSerializer) readAddrAndGases(r io.Reader, order binary.ByteOrder, ret *[]*proto.AddrAndGas) error {
	retLen, err := s.readUint32(r, order)
	if err != nil {
		return err
	}

	if retLen > maxSliceLength {
		return ErrSliceLengthExceedLimit
	} else if retLen == 0 {
		*ret = nil
		return nil
	}

	if *ret == nil || cap(*ret) < int(retLen) {
		*ret = make([]*proto.AddrAndGas, retLen)
	} else {
		*ret = (*ret)[:retLen]
	}

	for i := range *ret {
		(*ret)[i] = new(proto.AddrAndGas)

		err = s.readAddrAndGas(r, order, (*ret)[i])
		if err != nil {
			return err
		}
	}

	return nil
}

// readHashes reads hashes from reader with the following format:
//
// 0          4            4+hashsize   4+2*hashsize ...          4+(n+1)*hashsize
// +----------+------------+------------+------------+------------+
// | sliceLen |   hash_0   |   hash_1   |    ...     |   hash_n   |
// +----------+------------+------------+------------+------------+
//
func (s simpleSerializer) readHashes(r io.Reader, order binary.ByteOrder, ret *[]*hash.Hash) (
	err error) {
	var retLen uint32

	if retLen, err = s.readUint32(r, order); err != nil {
		return
	}

	if retLen > maxSliceLength {
		err = ErrSliceLengthExceedLimit
		return
	} else if retLen == 0 {
		// Always return nil slice for a zero-length
		*ret = nil
		return
	}

	if *ret == nil || cap(*ret) < int(retLen) {
		*ret = make([]*hash.Hash, retLen)
	} else {
		*ret = (*ret)[:retLen]
	}

	for i := range *ret {
		(*ret)[i] = new(hash.Hash)

		if err = s.readFixedSizeBytes(r, hash.HashSize, ((*ret)[i])[:]); err != nil {
			break
		}
	}

	return
}

// readAccountAddresses reads hashes from reader with the following format:
//
// 0          4            4+hashsize   4+2*hashsize ...          4+(n+1)*hashsize
// +----------+------------+------------+------------+------------+
// | sliceLen |   hash_0   |   hash_1   |    ...     |   hash_n   |
// +----------+------------+------------+------------+------------+
//
func (s simpleSerializer) readAccountAddresses(r io.Reader, order binary.ByteOrder, ret *[]*proto.AccountAddress) (
	err error) {
	var retLen uint32

	if retLen, err = s.readUint32(r, order); err != nil {
		return
	}

	if retLen > maxSliceLength {
		err = ErrSliceLengthExceedLimit
		return
	} else if retLen == 0 {
		// Always return nil slice for a zero-length
		*ret = nil
		return
	}

	if *ret == nil || cap(*ret) < int(retLen) {
		*ret = make([]*proto.AccountAddress, retLen)
	} else {
		*ret = (*ret)[:retLen]
	}

	for i := range *ret {
		(*ret)[i] = new(proto.AccountAddress)

		if err = s.readFixedSizeBytes(r, hash.HashSize, ((*ret)[i])[:]); err != nil {
			break
		}
	}

	return
}

func (s simpleSerializer) writeUint8(w io.Writer, val uint8) (err error) {
	buffer := s.borrowBuffer(1)
	defer s.returnBuffer(buffer)

	buffer[0] = val
	_, err = w.Write(buffer)
	return
}

func (s simpleSerializer) writeUint16(w io.Writer, order binary.ByteOrder, val uint16) (
	err error) {
	buffer := s.borrowBuffer(2)
	defer s.returnBuffer(buffer)

	order.PutUint16(buffer, val)
	_, err = w.Write(buffer)
	return
}

func (s simpleSerializer) writeUint32(w io.Writer, order binary.ByteOrder, val uint32) (
	err error) {
	buffer := s.borrowBuffer(4)
	defer s.returnBuffer(buffer)

	order.PutUint32(buffer, val)
	_, err = w.Write(buffer)
	return
}

func (s simpleSerializer) writeUint64(w io.Writer, order binary.ByteOrder, val uint64) (
	err error) {
	buffer := s.borrowBuffer(8)
	defer s.returnBuffer(buffer)

	order.PutUint64(buffer, val)
	_, err = w.Write(buffer)
	return
}

func (s simpleSerializer) writeFloat64(w io.Writer, order binary.ByteOrder, val float64) (
	err error) {
	buffer := s.borrowBuffer(8)
	defer s.returnBuffer(buffer)

	valToUint64 := math.Float64bits(val)
	order.PutUint64(buffer, valToUint64)
	_, err = w.Write(buffer)
	return
}

func (s simpleSerializer) writeSignature(w io.Writer, order binary.ByteOrder, val *asymmetric.Signature) (err error) {
	if val == nil {
		err = s.writeBytes(w, order, nil)
	} else {
		err = s.writeBytes(w, order, val.Serialize())
	}
	return
}

func (s simpleSerializer) writePublicKey(w io.Writer, order binary.ByteOrder, val *asymmetric.PublicKey) (err error) {
	if val == nil {
		err = s.writeBytes(w, order, nil)
	} else {
		err = s.writeBytes(w, order, val.Serialize())
	}
	return
}

// writeString writes string to writer with the following format:
//
// 0     4                                 4+len
// +-----+---------------------------------+
// | len |             string              |
// +-----+---------------------------------+
//
func (s simpleSerializer) writeString(w io.Writer, order binary.ByteOrder, val *string) (
	err error) {
	buffer := s.borrowBuffer(4 + len(*val))
	defer s.returnBuffer(buffer)

	valLen := uint32(len(*val))
	order.PutUint32(buffer, valLen)
	copy(buffer[4:], []byte(*val))
	_, err = w.Write(buffer)
	return
}

// writeAddrAndGas writes AddrAndGas to writer with the following format:
//
// 0             32             64          72
// +--------------+--------------+----------+
// |     hash     |     hash     |  uint64  |
// +--------------+--------------+----------+
//
func (s simpleSerializer) writeAddrAndGas(w io.Writer, order binary.ByteOrder, val *proto.AddrAndGas) (err error) {
	err = s.writeFixedSizeBytes(w, hash.HashSize, val.AccountAddress[:])
	if err != nil {
		return
	}
	err = s.writeFixedSizeBytes(w, hash.HashSize, val.RawNodeID.Hash[:])
	if err != nil {
		return
	}
	err = s.writeUint64(w, binary.BigEndian, val.GasAmount)
	if err != nil {
		return
	}

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

// writeUint32s writes bytes to writer with the following format:
//
// 0     4                                 4+len
// +-----+---------------------------------+
// | len |             uint32s             |
// +-----+---------------------------------+
//
func (s simpleSerializer) writeUint32s(w io.Writer, order binary.ByteOrder, val []uint32) (err error) {
	buffer := s.borrowBuffer(4 + len(val)*4)
	defer s.returnBuffer(buffer)

	valLen := uint32(len(val))
	order.PutUint32(buffer, valLen)
	for i := range val {
		order.PutUint32(buffer[4+i*4:], val[i])
	}
	_, err = w.Write(buffer)
	return
}

// writeUint64 writes bytes to writer with the following format:
//
// 0     4                                 4+len
// +-----+---------------------------------+
// | len |             uint64s             |
// +-----+---------------------------------+
//
func (s simpleSerializer) writeUint64s(w io.Writer, order binary.ByteOrder, val []uint64) (err error) {
	buffer := s.borrowBuffer(4 + len(val)*8)
	defer s.returnBuffer(buffer)

	valLen := uint32(len(val))
	order.PutUint32(buffer, valLen)
	for i := range val {
		order.PutUint64(buffer[4+i*8:], val[i])
	}
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

// writeStrings writes strings to writer with the following format:
//
// 0          4       8          8+len_0
// +----------+-------+----------+---------+-------+----------+
// | sliceLen | len_0 | string_0 |   ...   | len_n | string_n |
// +----------+-------+----------+---------+-------+----------+
//
func (s simpleSerializer) writeStrings(w io.Writer, order binary.ByteOrder, val []string) (
	err error) {
	if err = s.writeUint32(w, order, uint32(len(val))); err != nil {
		return
	}

	for i := range val {
		if err = s.writeString(w, order, &val[i]); err != nil {
			break
		}
	}

	return
}

// writeDatabaseIDs writes databaseIDs to writer with the following format:
//
// 0          4       8          8+len_0
// +----------+-------+--------------+---------+-------+--------------+
// | sliceLen | len_0 | databaseID_0 |   ...   | len_n | databaseID_n |
// +----------+-------+--------------+---------+-------+--------------+
//
func (s simpleSerializer) writeDatabaseIDs(w io.Writer,
	order binary.ByteOrder,
	val []proto.DatabaseID) (err error) {
	if err = s.writeUint32(w, order, uint32(len(val))); err != nil {
		return
	}

	for i := range val {
		if err = s.writeString(w, order, (*string)(&val[i])); err != nil {
			break
		}
	}

	return
}

// writeSignatures writes signatures to writer with the following format:
//
// 0          4            4+signaturesize   4+2*signaturesize ...          4+(n+1)*signaturesize
// +----------+-----------------+-----------------+------------+-----------------+
// | sliceLen |   signature_0   |   signature_1   |    ...     |   signature_n   |
// +----------+-----------------+-----------------+------------+-----------------+
//
func (s simpleSerializer) writeSignatures(w io.Writer, order binary.ByteOrder, val []*asymmetric.Signature) (
	err error) {
	if err = s.writeUint32(w, order, uint32(len(val))); err != nil {
		return
	}

	for _, v := range val {
		if err = s.writeSignature(w, order, v); err != nil {
			break
		}
	}

	return
}

// writePublicKeys writes public key to writer with the following format:
//
// 0          4            4+pubkeysize   4+2*pubkeysize ...          4+(n+1)*pubkeysize
// +----------+-----------------+-----------------+------------+-----------------+
// | sliceLen |   publickey_0   |   publickey_1   |    ...     |   publickey_n   |
// +----------+-----------------+-----------------+------------+-----------------+
//
func (s simpleSerializer) writePublicKeys(w io.Writer, order binary.ByteOrder, val []*asymmetric.PublicKey) (
	err error) {
	if err = s.writeUint32(w, order, uint32(len(val))); err != nil {
		return
	}

	for _, v := range val {
		if err = s.writePublicKey(w, order, v); err != nil {
			break
		}
	}

	return
}

// writeHashes writes hashes to writer with the following format:
//
// 0          4            4+hashsize   4+2*hashsize ...          4+(n+1)*hashsize
// +----------+------------+------------+------------+------------+
// | sliceLen |   hash_0   |   hash_1   |    ...     |   hash_n   |
// +----------+------------+------------+------------+------------+
//
func (s simpleSerializer) writeHashes(w io.Writer, order binary.ByteOrder, val []*hash.Hash) (
	err error) {
	if err = s.writeUint32(w, order, uint32(len(val))); err != nil {
		return
	}

	for _, v := range val {
		if err = s.writeFixedSizeBytes(w, hash.HashSize, v[:]); err != nil {
			break
		}
	}

	return
}

// writeAccountAddresses writes hashes to writer with the following format:
//
// 0          4            4+hashsize   4+2*hashsize ...          4+(n+1)*hashsize
// +----------+------------+------------+------------+------------+
// | sliceLen |   hash_0   |   hash_1   |    ...     |   hash_n   |
// +----------+------------+------------+------------+------------+
//
func (s simpleSerializer) writeAccountAddresses(w io.Writer, order binary.ByteOrder, val []*proto.AccountAddress) (err error) {
	if err = s.writeUint32(w, order, uint32(len(val))); err != nil {
		return
	}

	for _, v := range val {
		if err = s.writeFixedSizeBytes(w, hash.HashSize, v[:]); err != nil {
			break
		}
	}

	return
}

// writeAddrAndGases writes hashes to writer with the following format:
//
// 0          4
// +----------+----------------+---------------+-------+----------------+
// | sliceLen |  AddrAndGas_0  |  AddrAndGas_1 |  ...  |  AddrAndGas_2  |
// +----------+----------------+---------------+-------+----------------+
//
func (s simpleSerializer) writeAddrAndGases(w io.Writer, order binary.ByteOrder, val []*proto.AddrAndGas) error {
	if err := s.writeUint32(w, order, uint32(len(val))); err != nil {
		return err
	}

	for _, v := range val {
		if err := s.writeAddrAndGas(w, order, v); err != nil {
			return err
		}
	}

	return nil
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

	case *float64:
		*e, err = serializer.readFloat64(r, order)

	case *time.Time:
		var ret uint64

		if ret, err = serializer.readUint64(r, order); err == nil {
			*e = time.Unix(0, int64(ret)).UTC()
		}

	case *string:
		err = serializer.readString(r, order, e)

	case *[]byte:
		err = serializer.readBytes(r, order, e)

	case *[]uint32:
		err = serializer.readUint32s(r, order, e)

	case *[]uint64:
		err = serializer.readUint64s(r, order, e)

	case *proto.NodeID:
		err = serializer.readString(r, order, (*string)(e))

	case *proto.DatabaseID:
		err = serializer.readString(r, order, (*string)(e))

	case *hash.Hash:
		err = serializer.readFixedSizeBytes(r, hash.HashSize, (*e)[:])

	case *proto.AccountAddress:
		err = serializer.readFixedSizeBytes(r, hash.HashSize, (*e)[:])

	case *proto.AddrAndGas:
		err = serializer.readAddrAndGas(r, order, e)

	case **asymmetric.PublicKey:
		*e, err = serializer.readPublicKey(r, order)

	case **asymmetric.Signature:
		*e, err = serializer.readSignature(r, order)

	case *[]*asymmetric.PublicKey:
		serializer.readPublicKeys(r, order, e)

	case *[]*asymmetric.Signature:
		serializer.readSignatures(r, order, e)

	case *[]string:
		err = serializer.readStrings(r, order, e)

	case *[]*hash.Hash:
		err = serializer.readHashes(r, order, e)

	case *[]*proto.AccountAddress:
		err = serializer.readAccountAddresses(r, order, e)

	case *[]proto.DatabaseID:
		err = serializer.readDatabaseIDs(r, order, e)

	case *[]*proto.AddrAndGas:
		err = serializer.readAddrAndGases(r, order, e)

	default:
		// Fallback to BinaryUnmarshaler interface
		if i, ok := e.(encoding.BinaryUnmarshaler); ok {
			var buffer []byte

			if err = serializer.readBytes(r, order, &buffer); err != nil {
				return
			}

			return i.UnmarshalBinary(buffer)
		}

		log.Debugf("element type is: %s", reflect.TypeOf(e))
		return ErrInvalidType
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

	case float64:
		err = serializer.writeFloat64(w, order, e)

	case *float64:
		err = serializer.writeFloat64(w, order, *e)

	case string:
		err = serializer.writeString(w, order, &e)

	case *string:
		err = serializer.writeString(w, order, e)

	case []byte:
		err = serializer.writeBytes(w, order, e)

	case *[]byte:
		err = serializer.writeBytes(w, order, *e)

	case []uint32:
		err = serializer.writeUint32s(w, order, e)

	case *[]uint32:
		err = serializer.writeUint32s(w, order, *e)

	case []uint64:
		err = serializer.writeUint64s(w, order, e)

	case *[]uint64:
		err = serializer.writeUint64s(w, order, *e)

	case time.Time:
		err = serializer.writeUint64(w, order, (uint64)(e.UnixNano()))

	case *time.Time:
		err = serializer.writeUint64(w, order, (uint64)(e.UnixNano()))

	case proto.NodeID:
		err = serializer.writeString(w, order, (*string)(&e))

	case *proto.NodeID:
		err = serializer.writeString(w, order, (*string)(e))

	case proto.DatabaseID:
		err = serializer.writeString(w, order, (*string)(&e))

	case *proto.DatabaseID:
		err = serializer.writeString(w, order, (*string)(e))

	case hash.Hash:
		err = serializer.writeFixedSizeBytes(w, hash.HashSize, e[:])

	case *hash.Hash:
		err = serializer.writeFixedSizeBytes(w, hash.HashSize, (*e)[:])

	case proto.AccountAddress:
		err = serializer.writeFixedSizeBytes(w, hash.HashSize, e[:])

	case *proto.AccountAddress:
		err = serializer.writeFixedSizeBytes(w, hash.HashSize, (*e)[:])

	case *proto.AddrAndGas:
		serializer.writeAddrAndGas(w, order, e)

	case proto.AddrAndGas:
		serializer.writeAddrAndGas(w, order, &e)

	case *asymmetric.PublicKey:
		serializer.writePublicKey(w, order, e)

	case **asymmetric.PublicKey:
		serializer.writePublicKey(w, order, *e)

	case *asymmetric.Signature:
		serializer.writeSignature(w, order, e)

	case **asymmetric.Signature:
		serializer.writeSignature(w, order, *e)

	case []*asymmetric.Signature:
		err = serializer.writeSignatures(w, order, e)

	case *[]*asymmetric.Signature:
		err = serializer.writeSignatures(w, order, *e)

	case []*asymmetric.PublicKey:
		err = serializer.writePublicKeys(w, order, e)

	case *[]*asymmetric.PublicKey:
		err = serializer.writePublicKeys(w, order, *e)

	case []string:
		err = serializer.writeStrings(w, order, e)

	case *([]string):
		err = serializer.writeStrings(w, order, *e)

	case []*hash.Hash:
		err = serializer.writeHashes(w, order, e)

	case *[]*hash.Hash:
		err = serializer.writeHashes(w, order, *e)

	case []*proto.AccountAddress:
		err = serializer.writeAccountAddresses(w, order, e)

	case *[]*proto.AccountAddress:
		err = serializer.writeAccountAddresses(w, order, *e)

	case []proto.DatabaseID:
		err = serializer.writeDatabaseIDs(w, order, e)

	case *[]proto.DatabaseID:
		err = serializer.writeDatabaseIDs(w, order, *e)

	case []*proto.AddrAndGas:
		err = serializer.writeAddrAndGases(w, order, e)

	case *[]*proto.AddrAndGas:
		err = serializer.writeAddrAndGases(w, order, *e)

	default:
		// Fallback to BinaryMarshaler interface
		if i, ok := e.(encoding.BinaryMarshaler); ok {
			var data []byte

			if data, err = i.MarshalBinary(); err == nil {
				err = serializer.writeBytes(w, order, data)
			}

			return
		}

		log.Debugf("element type is: %s", reflect.TypeOf(e))
		return ErrInvalidType
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
