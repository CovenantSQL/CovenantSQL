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
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"testing"
	"time"
	"unsafe"

	"github.com/CovenantSQL/CovenantSQL/utils/log"
	. "github.com/smartystreets/goconvey/convey"
)

const testNum  = 200

func init() {
	rand.Seed(time.Now().UnixNano())
}

func TestUint256(t *testing.T) {
	Convey("uint256 len", t, func() {
		i := Uint256{}
		So(unsafe.Sizeof(Uint256{}), ShouldEqual, 32)
		So(len(i.Bytes()), ShouldEqual, 32)
	})
	Convey("convert byte", t, func() {
		i := Uint256{math.MaxUint64, 3, 444, 1230}
		log.Print(i.Bytes())
		j, err := Uint256FromBytes(i.Bytes())

		So(err, ShouldBeNil)
		So(j.A == math.MaxUint64, ShouldBeTrue)
		So(j.B, ShouldEqual, 3)
		So(j.C, ShouldEqual, 444)
		So(j.D, ShouldEqual, 1230)
	})
	Convey("convert byte error", t, func() {
		i, err := Uint256FromBytes([]byte("aaa"))
		So(err, ShouldEqual, ErrBytesLen)
		So(i, ShouldBeNil)
	})
	Convey("zero and max", t, func() {
		z := Zero()
		m := MaxUint256()
		So(m.Inc().String(), ShouldEqual, z.String())
		one := Zero()
		one.Inc()
		z.Sub(one)
		So(z.String(), ShouldEqual, MaxUint256().String())
	})
	Convey("convert string", t, func() {
		for i := 0; i < testNum; i++ {
			a := rand.Uint64() & 0x7fffffffffffffff | 0x7000000000000000
			b := rand.Uint64() & 0x7fffffffffffffff | 0x7000000000000000
			c := rand.Uint64() & 0x7fffffffffffffff | 0x7000000000000000
			d := rand.Uint64() & 0x7fffffffffffffff | 0x7000000000000000
			m := &Uint256{a, b, c, d}
			n, _ := big.NewInt(0).SetString(fmt.Sprintf("%x%x%x%x", d, c, b, a), 16)
			So(m.String(), ShouldEqual, n.String())
		}
		m1 := &Uint256{math.MaxUint64, 0, 0, 0}
		So(m1.String(), ShouldEqual, "18446744073709551615")
		m2 := &Uint256{1, math.MaxUint64, 0, 0}
		So(m2.String(), ShouldEqual, "340282366920938463444927863358058659841")
	})
	Convey("equal or not equal", t, func() {
		for i := 0; i < testNum; i++ {
			var m, n *Uint256
			var bigM, bigN *big.Int
			if i % 2 == 0 {
				m = notRandomUint256()
				n = notRandomUint256()
				bigM = m.BigInt()
				bigN = n.BigInt()
			} else {
				m = notRandomUint256()
				n = &Uint256{}
				n.Copy(m)
				bigM = m.BigInt()
				bigN = n.BigInt()
			}
			So(m.Equal(n), ShouldEqual, bigM.Cmp(bigN) == 0)
		}
	})
	Convey("compare two number", t, func() {
		for i := 0; i < testNum; i++ {
			m := notRandomUint256()
			n := notRandomUint256()
			bigM := m.BigInt()
			bigN := n.BigInt()
			log.Infof("m = {A: %x, B: %x, C: %x, D: %x}, %v", m.A, m.B, m.C, m.D, m.Bytes())
			log.Infof("n = {A: %x, B: %x, C: %x, D: %x}", n.A, n.B, n.C, n.D)
			log.Infof("bigM = %s", bigM.String())
			log.Infof("BigN = %s", bigN.String())
			So(m.Compare(n), ShouldEqual, bigM.Cmp(bigN))
		}
	})
	Convey("copy number", t, func() {
		i0 := &Uint256{1,2,3,4}
		i1 := &Uint256{}
		i1.Copy(i0)
		So(i1.Equal(i0), ShouldBeTrue)
	})
	Convey("convert to BigInt", t, func() {
		for i := 0; i < testNum; i++ {
			m := notRandomUint256()
			So(m.BigInt().String(), ShouldEqual, m.String())
		}
	})
}

func TestUint256_Inc(t *testing.T) {
	Convey("uint256 inc", t, func() {
		i := Uint256{}
		i.Inc()
		So(i.A, ShouldEqual, 1)
		So(i.B, ShouldEqual, 0)
		So(i.C, ShouldEqual, 0)
		So(i.D, ShouldEqual, 0)
	})
	Convey("uint256 inc", t, func() {
		i := Uint256{math.MaxUint64, 0, 0, 0}
		i.Inc()
		So(i.A, ShouldEqual, 0)
		So(i.B, ShouldEqual, 1)
		So(i.C, ShouldEqual, 0)
		So(i.D, ShouldEqual, 0)
	})
	Convey("uint256 inc", t, func() {
		i := Uint256{math.MaxUint64, math.MaxUint64, 0, 0}
		i.Inc()
		So(i.A, ShouldEqual, 0)
		So(i.B, ShouldEqual, 0)
		So(i.C, ShouldEqual, 1)
		So(i.D, ShouldEqual, 0)
	})
	Convey("uint256 inc", t, func() {
		i := Uint256{math.MaxUint64, math.MaxUint64, math.MaxUint64, 0}
		i.Inc()
		So(i.A, ShouldEqual, 0)
		So(i.B, ShouldEqual, 0)
		So(i.C, ShouldEqual, 0)
		So(i.D, ShouldEqual, 1)
	})
	Convey("uint256 inc", t, func() {
		i := Uint256{math.MaxUint64, math.MaxUint64, math.MaxUint64, math.MaxUint64}
		i.Inc()
		So(i.A, ShouldEqual, 0)
		So(i.B, ShouldEqual, 0)
		So(i.C, ShouldEqual, 0)
		So(i.D, ShouldEqual, 0)
	})
}

func TestUint256_ToIPv6(t *testing.T) {
	Convey("uint256 to IPv6", t, func() {
		src := Uint256{}
		ab, cd, err := src.ToIPv6()
		So(ab.IsUnspecified(), ShouldBeTrue)
		So(cd.IsUnspecified(), ShouldBeTrue)

		i, err := FromIPv6(ab, cd)
		So(err, ShouldBeNil)
		So(i.A, ShouldEqual, 0)
		So(i.B, ShouldEqual, 0)
		So(i.C, ShouldEqual, 0)
		So(i.D, ShouldEqual, 0)

		src = Uint256{math.MaxUint64, math.MaxUint64, math.MaxUint64, math.MaxUint64}
		ab, cd, err = src.ToIPv6()
		So(err, ShouldBeNil)
		So(ab.String(), ShouldEqual, "ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff")
		So(cd.String(), ShouldEqual, "ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff")

		src = Uint256{math.MaxUint64, 0, math.MaxUint64, 0x10}
		ab, cd, err = src.ToIPv6()
		So(err, ShouldBeNil)
		So(ab.String(), ShouldEqual, "ffff:ffff:ffff:ffff::")
		So(cd.String(), ShouldEqual, "ffff:ffff:ffff:ffff::10")

		//00000bef611d346c0cbe1beaa76e7f0ed705a194fdf9ac3a248ec70e9c198bf9
		src = Uint256{313283, 0, 0, 0}
		ab, cd, err = src.ToIPv6()
		So(err, ShouldBeNil)
		So(ab.String(), ShouldEqual, "0:0:4:c7c3::")
		So(cd.String(), ShouldEqual, "::")

		//00000381d46fd6cf7742d7fb94e2422033af989c0e348b5781b3219599a3af35
		src = Uint256{478373, 0, 0, 2305843009893772025}
		ab, cd, err = src.ToIPv6()
		So(err, ShouldBeNil)
		So(ab.String(), ShouldEqual, "0:0:7:4ca5::")
		So(cd.String(), ShouldEqual, "::2000:0:2889:2af9")

		//000000172580063ded88e010556b0aca2851265be8845b1ef397e8fce6ab5582
		src = Uint256{259939, 0, 0, 2305843012544226372}
		ab, cd, err = src.ToIPv6()
		So(err, ShouldBeNil)
		So(ab.String(), ShouldEqual, "0:0:3:f763::")
		So(cd.String(), ShouldEqual, "::2000:0:c683:e444")
	})
}

func TestUint256_AddAndSafeAdd(t *testing.T) {
	Convey("add two uint256", t, func() {
		mask := big.NewInt(1)
		mask.Lsh(mask, 256)
		for i := 0; i < testNum; i++ {
			m := notRandomUint256()
			n := notRandomUint256()
			bigM := m.BigInt()
			bigN := n.BigInt()
			m.Add(n)
			bigM.Add(bigM, bigN)
			bigM.Mod(bigM, mask)
			So(m.String(), ShouldEqual, bigM.String())
		}
	})
	Convey("safeadd two uint256", t, func() {
		mask := big.NewInt(1)
		mask.Lsh(mask, 256)
		for i := 0; i < testNum; i++ {
			m := notRandomUint256()
			n := notRandomUint256()
			bigM := m.BigInt()
			bigN := n.BigInt()
			bigM.Add(bigM, bigN)
			if bigM.Cmp(mask) < 0 {
				err := m.SafeAdd(n)
				So(err, ShouldBeNil)
				So(m.String(), ShouldEqual, bigM.String())
			} else {
				err := m.SafeAdd(n)
				So(err, ShouldNotBeNil)
			}
		}
	})
}

func TestUint256_SubAndSafeSub(t *testing.T) {
	Convey("sub two uint256", t, func() {
		mask := big.NewInt(1)
		mask.Lsh(mask, 256)
		for i := 0; i < testNum; i++ {
			m := notRandomUint256()
			n := notRandomUint256()
			bigM := m.BigInt()
			bigN := n.BigInt()
			m.Sub(n)
			bigM.Sub(bigM, bigN)
			if bigM.Cmp(big.NewInt(0)) < 0 {
				bigM.Add(bigM, mask)
			}
			So(m.String(), ShouldEqual, bigM.String())
		}
	})
	Convey("safeadd two uint256", t, func() {
		mask := big.NewInt(1)
		mask.Lsh(mask, 256)
		for i := 0; i < testNum; i++ {
			m := notRandomUint256()
			n := notRandomUint256()
			bigM := m.BigInt()
			bigN := n.BigInt()
			bigM.Sub(bigM, bigN)
			log.Infof("m = {A: %x, B: %x, C: %x, D: %x}", m.A, m.B, m.C, m.D)
			log.Infof("n = {A: %x, B: %x, C: %x, D: %x}", n.A, n.B, n.C, n.D)
			if bigM.Cmp(big.NewInt(0)) >= 0 {
				err := m.SafeSub(n)
				So(err, ShouldBeNil)
				So(m.String(), ShouldEqual, bigM.String())
			} else {
				err := m.SafeSub(n)
				So(err, ShouldNotBeNil)
			}
		}
	})
}

func notRandomUint256() *Uint256 {
	var m *Uint256
	dice := rand.Int31n(25)
	if dice == 9 {
		m = &Uint256{0, 0, 0, 0}
		return m
	}
	dice %= 4
	if dice == 0 {
		m = &Uint256{rand.Uint64(), rand.Uint64(), rand.Uint64(), rand.Uint64()}
	} else if dice == 1 {
		m = &Uint256{0, rand.Uint64(), rand.Uint64(), rand.Uint64()}
	} else if dice == 2 {
		m = &Uint256{0, 0, rand.Uint64(), rand.Uint64()}
	} else {
		m = &Uint256{0, 0, 0, rand.Uint64()}
	}
	return m
}
