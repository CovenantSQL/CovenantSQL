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
"math/big"
"net"

	hsp "github.com/CovenantSQL/HashStablePack/marshalhash"
	"github.com/ethereum/go-ethereum/common/math"
)

var (
	// ErrBytesLen is an error type.
	ErrBytesLen = errors.New("byte length should be 32 for Uint256")
	// ErrEmptyIPv6Addr is an error type.
	ErrEmptyIPv6Addr = errors.New("nil or zero length IPv6")
	// ErrInsufficientBalance indicates that an account has insufficient balance for spending.
	ErrInsufficientBalance = errors.New("insufficient balance")
)

// Uint256 is an unsigned 256 bit integer.
type Uint256 struct {
	A uint64 // Bits 63..0.
	B uint64 // Bits 127..64.
	C uint64 // Bits 191..128.
	D uint64 // Bits 255..192.
}

// Zero returns zero number of Uint256.
func Zero() *Uint256 {
	return &Uint256{0,0,0,0}
}

// MaxUint256 returns max number of Uint256
func MaxUint256() *Uint256 {
	return &Uint256{math.MaxUint64, math.MaxUint64, math.MaxUint64, math.MaxUint64}
}

// Equal returns if two number is equal.
func (i *Uint256) Equal(j *Uint256) bool {
	return (i.A == j.A) && (i.B == j.B) && (i.C == j.C) && (i.D == j.D)
}

// Compare compares two number.
// If i > j then returns 1.
// If i == j then returns 0.
// If i < j then retuns -1.
func (i *Uint256) Compare(j *Uint256) int8 {
	if res := compareHelper(i.D, j.D); res != 0 {
		return res
	}
	if res := compareHelper(i.C, j.C); res != 0 {
		return res
	}
	if res := compareHelper(i.B, j.B); res != 0 {
		return res
	}
	if res := compareHelper(i.A, j.A); res != 0 {
		return res
	}

	return 0
}

func compareHelper(i, j uint64) int8 {
	if i > j {
		return 1
	} else if i < j {
		return -1
	} else {
		return 0
	}
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

// String converts Uint256 to string.
func (i *Uint256) String() string {
	n := big.NewInt(0).SetUint64(i.D)
	n.Lsh(n, 64)
	n.Add(n, big.NewInt(0).SetUint64(i.C))
	n.Lsh(n, 64)
	n.Add(n, big.NewInt(0).SetUint64(i.B))
	n.Lsh(n, 64)
	n.Add(n, big.NewInt(0).SetUint64(i.A))
	return n.String()
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

//FromIPv6 converts 2 IPv6 addresses to Uint256.
func FromIPv6(ab, cd net.IP) (ret *Uint256, err error) {
	if ab == nil || cd == nil || len(ab) == 0 || len(cd) == 0 {
		return nil, ErrEmptyIPv6Addr
	}
	buf := make([]byte, 0, 32)
	buf = append(buf, ab...)
	buf = append(buf, cd...)
	return Uint256FromBytes(buf)
}

// SafeSub provides a safe sub method with lower overflow check for uint256.
func (i *Uint256) SafeSub(j *Uint256) (err error) {
	if i.Compare(j) < 0 {
		return ErrInsufficientBalance
	}

	i.Sub(j)
	return
}

// Sub provides a subtraction for Uint256.
func (i *Uint256) Sub(j *Uint256) {
	borrow := i.A < j.A
	i.A = i.A - j.A
	borrow = subWithBorrow(borrow, &(i.B), &(j.B))
	borrow = subWithBorrow(borrow, &(i.C), &(j.C))
	borrow = subWithBorrow(borrow, &(i.D), &(j.D))
}

func subWithBorrow(borrow bool, i, j *uint64) bool {
	if *i > *j {
		*i -= *j
		if borrow {
			*i -= 1
		}
		return false
	} else {
		*i -= *j
		if borrow {
			*i -= 1
			return true
		}
		return *i != 0
	}
}

// SafeAdd provides a safe add method with upper overflow check for uint256.
func (i *Uint256) SafeAdd(j *Uint256) (err error) {
	sumIJ := &Uint256{}
	sumIJ.Copy(i)
	sumIJ.Add(j)
	if sumIJ.Compare(i) < 0 {
		return ErrInsufficientBalance
	}

	i.Copy(sumIJ)
	return nil
}

// Add provides a addition for Uint256.
func (i *Uint256) Add(j *Uint256) {
	carry := false
	carry = addWithCarry(carry, &(i.A), &(j.A))
	carry = addWithCarry(carry, &(i.B), &(j.B))
	carry = addWithCarry(carry, &(i.C), &(j.C))
	carry = addWithCarry(carry, &(i.D), &(j.D))
}

func addWithCarry(carry bool, i, j *uint64) bool {
	if *i + *j < *i {
		*i += *j
		if carry {
			*i += 1
		}
		return true
	} else {
		*i += *j
		if carry {
			if *i + 1 < *i {
				*i += 1
				return true
			} else {
				*i += 1
				return false
			}
		}
		return false
	}
}

// Copy sets i to j and return i. j is not changed even if i and j are the same.
func (i *Uint256) Copy(j *Uint256) *Uint256 {
	if i != j {
		i.A = j.A
		i.B = j.B
		i.C = j.C
		i.D = j.D
	}
	return i
}

// BigInt converts Uint256 to BigInt
func (i *Uint256) BigInt() *big.Int {
	var binBuf bytes.Buffer
	binary.Write(&binBuf, binary.BigEndian, i.D)
	binary.Write(&binBuf, binary.BigEndian, i.C)
	binary.Write(&binBuf, binary.BigEndian, i.B)
	binary.Write(&binBuf, binary.BigEndian, i.A)

	n := big.NewInt(0).SetBytes(binBuf.Bytes())
	return n
}