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

package cpuminer

import (
	"math"
	"testing"

	"gitlab.com/thunderdb/ThunderDB/utils/log"

	"unsafe"

	. "github.com/smartystreets/goconvey/convey"
)

func TestUint256(t *testing.T) {
	Convey("uint256 len", t, func() {
		i := Uint256{}
		So(unsafe.Sizeof(Uint256{}), ShouldEqual, 32)
		So(len(i.Bytes()), ShouldEqual, 32)
	})
	Convey("convert", t, func() {
		i := Uint256{math.MaxUint64, 3, 444, 1230}
		log.Print(i.Bytes())
		j, err := Uint256FromBytes(i.Bytes())

		So(err, ShouldBeNil)
		So(j.A == math.MaxUint64, ShouldBeTrue)
		So(j.B, ShouldEqual, 3)
		So(j.C, ShouldEqual, 444)
		So(j.D, ShouldEqual, 1230)
	})
	Convey("convert error", t, func() {
		i, err := Uint256FromBytes([]byte("aaa"))
		So(err, ShouldEqual, ErrBytesLen)
		So(i, ShouldBeNil)
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

		src = Uint256{14396347928, 0, 0, 6148914694092305796}
		ab, cd, err = src.ToIPv6()
		So(err, ShouldBeNil)
		So(ab.String(), ShouldEqual, "0:3:5a16:d618::")
		So(cd.String(), ShouldEqual, "::5555:5555:ff8d:3584")
	})
}
