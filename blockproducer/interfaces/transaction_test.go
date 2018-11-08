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

package interfaces

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestTypes(t *testing.T) {
	Convey("Transaction types should be consistent to convert to/from bytes", t, func() {
		for tt := TransactionType(0); tt < TransactionTypeNumber; tt++ {
			So(tt, ShouldEqual, FromBytes(tt.Bytes()))
		}
	})
	Convey("Transaction types should be hash stable", t, func() {
		var (
			h1, h2 []byte
			err    error
		)
		for tt := TransactionType(0); tt < TransactionTypeNumber; tt++ {
			h1, err = tt.MarshalHash()
			So(err, ShouldBeNil)
			h2, err = tt.MarshalHash()
			So(err, ShouldBeNil)
			So(h1, ShouldResemble, h2)
		}
	})
	Convey("Nonce should be hash stable", t, func() {
		var (
			h1, h2 []byte
			err    error
		)
		for n := AccountNonce(0); n < AccountNonce(10); n++ {
			h1, err = n.MarshalHash()
			So(err, ShouldBeNil)
			h2, err = n.MarshalHash()
			So(err, ShouldBeNil)
			So(h1, ShouldResemble, h2)
		}
	})
	Convey("test string", t, func() {
		for i := TransactionTypeBilling; i != TransactionTypeNumber+1; i++ {
			So(i.String(), ShouldNotBeEmpty)
		}
	})
}
