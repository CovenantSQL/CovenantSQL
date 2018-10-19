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

package types

import (
	"reflect"
	"testing"

	pi "github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	"github.com/CovenantSQL/CovenantSQL/utils"
	. "github.com/smartystreets/goconvey/convey"
)

func TestEncodeSingleTransaction(t *testing.T) {
	Convey("test encode/decode single transaction", t, func() {
		var t pi.Transaction
		t = NewBaseAccount(&Account{})
		buf, err := utils.EncodeMsgPack(t)
		So(err, ShouldBeNil)

		var out pi.Transaction
		err = utils.DecodeMsgPack(buf.Bytes(), &out)
		So(err, ShouldBeNil)
		So(out, ShouldNotBeNil)
		So(reflect.TypeOf(out).String(), ShouldContainSubstring, "TransactionWrapper")
		So(out.GetTransactionType(), ShouldEqual, t.GetTransactionType())
	})

	Convey("test encode/decode series of transactions", t, func() {
		var t []pi.Transaction
		t = append(t, NewBaseAccount(&Account{}))
		t = append(t, NewTransfer(&TransferHeader{}))
		t = append(t, NewBilling(&BillingHeader{}))
		t = append(t, NewCreateDatabase(&CreateDatabaseHeader{}))

		buf, err := utils.EncodeMsgPack(t)
		So(err, ShouldBeNil)

		var out []pi.Transaction
		err = utils.DecodeMsgPack(buf.Bytes(), &out)
		So(err, ShouldBeNil)
		So(out, ShouldNotBeNil)
		So(out, ShouldHaveLength, len(t))

		for i := range t {
			So(out[i].GetTransactionType(), ShouldEqual, t[i].GetTransactionType())
			So(reflect.TypeOf(out[i]).String(), ShouldContainSubstring, "TransactionWrapper")
		}
	})
}
