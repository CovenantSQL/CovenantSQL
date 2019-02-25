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

package cpuminer

import (
	"bytes"
	"encoding/binary"
	"errors"
	"net"

	hsp "github.com/CovenantSQL/HashStablePack/marshalhash"
)

var (
	// ErrBytesLen is an error type
	ErrBytesLen = errors.New("byte length should be 32 for Uint256")
	// ErrEmptyIPv6Addr is an error type
	ErrEmptyIPv6Addr = errors.New("nil or zero length IPv6")
)

// Uint256 is an unsigned 256 bit integer.
type Uint256 struct {
	A uint64 // Bits 63..0.
	B uint64 // Bits 127..64.
	C uint64 // Bits 191..128.
	D uint64 // Bits 255..192.
}

// Inc makes i = i + 1.
func (i *Uint256) Inc() (ret *Uint256) {
	if i.A++; i.A == 0 {
		if i.B++; i.B == 0 {
			if i.C++; i.C == 0 {
				i.D++
			}
		}
	}
	return i
}

// Bytes converts Uint256 to []byte.
func (i *Uint256) Bytes() []byte {
	var binBuf bytes.Buffer
	binary.Write(&binBuf, binary.BigEndian, i)
	return binBuf.Bytes()
}

// MarshalHash marshals for hash.
func (i *Uint256) MarshalHash() (o []byte, err error) {
	return i.Bytes(), nil
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message.
func (i *Uint256) Msgsize() (s int) {
	return hsp.BytesPrefixSize + 32
}

// Uint256FromBytes converts []byte to Uint256.
func Uint256FromBytes(b []byte) (*Uint256, error) {
	if len(b) != 32 {
		return nil, ErrBytesLen
	}
	i := Uint256{}
	binary.Read(bytes.NewBuffer(b), binary.BigEndian, &i)
	return &i, nil
}

// ToIPv6 converts Uint256 to 2 IPv6 addresses.
func (i *Uint256) ToIPv6() (ab, cd net.IP, err error) {
	buf := i.Bytes()
	ab = make(net.IP, 0, net.IPv6len)
	cd = make(net.IP, 0, net.IPv6len)
	ab = append(ab, buf[:16]...)
	cd = append(cd, buf[16:]...)
	return
}

//FromIPv6 converts 2 IPv6 addresses to Uint256
func FromIPv6(ab, cd net.IP) (ret *Uint256, err error) {
	if ab == nil || cd == nil || len(ab) == 0 || len(cd) == 0 {
		return nil, ErrEmptyIPv6Addr
	}
	buf := make([]byte, 0, 32)
	buf = append(buf, ab...)
	buf = append(buf, cd...)
	return Uint256FromBytes(buf)
}
