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
	"math"
	"testing"
	"unsafe"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/CovenantSQL/CovenantSQL/utils/log"
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
