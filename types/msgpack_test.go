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

	. "github.com/smartystreets/goconvey/convey"

	pi "github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	"github.com/CovenantSQL/CovenantSQL/utils"
)

type complexStructure struct {
	Tx   pi.Transaction
	Txs  []pi.Transaction
	Maps map[string]pi.Transaction
}

func TestEncodeDecodeTransactions(t *testing.T) {
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
		t = append(t, NewCreateDatabase(&CreateDatabaseHeader{}))

		buf, err := utils.EncodeMsgPack(t)
		So(err, ShouldBeNil)

		var out []pi.Transaction
		err = utils.DecodeMsgPack(buf.Bytes(), &out)
		So(err, ShouldBeNil)
		So(out, ShouldNotBeNil)
		So(out, ShouldHaveLength, len(t))

		for i := range t {
			So(out[i], ShouldNotBeNil)
			So(out[i].GetTransactionType(), ShouldEqual, t[i].GetTransactionType())
			So(reflect.TypeOf(out[i]).String(), ShouldContainSubstring, "TransactionWrapper")
		}
	})
	Convey("test encode/decode complex structure containing transactions", t, func() {
		var t complexStructure
		t.Tx = NewBaseAccount(&Account{})
		t.Txs = append(t.Txs, NewBaseAccount(&Account{}))
		t.Txs = append(t.Txs, NewTransfer(&TransferHeader{}))
		t.Txs = append(t.Txs, NewCreateDatabase(&CreateDatabaseHeader{}))
		t.Maps = make(map[string]pi.Transaction)
		t.Maps["BaseAccount"] = NewBaseAccount(&Account{})
		t.Maps["Transfer"] = NewTransfer(&TransferHeader{})
		t.Maps["CreateDatabase"] = NewCreateDatabase(&CreateDatabaseHeader{})
		buf, err := utils.EncodeMsgPack(t)
		So(err, ShouldBeNil)

		var out complexStructure
		err = utils.DecodeMsgPack(buf.Bytes(), &out)
		So(err, ShouldBeNil)
		So(out.Tx, ShouldNotBeNil)
		So(out.Txs, ShouldNotBeNil)
		So(out.Txs, ShouldHaveLength, len(t.Txs))
		So(out.Maps, ShouldNotBeNil)
		So(out.Maps, ShouldHaveLength, len(t.Maps))

		So(out.Tx.GetTransactionType(), ShouldEqual, pi.TransactionTypeBaseAccount)
		So(reflect.TypeOf(out.Tx).String(), ShouldContainSubstring, "TransactionWrapper")

		for i := range out.Txs {
			So(out.Txs[i], ShouldNotBeNil)
			So(out.Txs[i].GetTransactionType(), ShouldEqual, t.Txs[i].GetTransactionType())
			So(reflect.TypeOf(out.Txs[i]).String(), ShouldContainSubstring, "TransactionWrapper")
		}

		for k, v := range out.Maps {
			So(v, ShouldNotBeNil)
			So(out.Maps[k].GetTransactionType(), ShouldEqual, t.Maps[k].GetTransactionType())
			So(reflect.TypeOf(out.Maps[k]).String(), ShouldContainSubstring, "TransactionWrapper")
		}
	})
	Convey("decode invalid data", t, func() {
		var testTypes = []interface{}{
			"1",
			1,
			0.1,
			[]int{},
			[]int{1, 2},
			[]int{1, 2, 3},
			struct{ A int }{A: 1},
			struct{ TxType int }{TxType: int(pi.TransactionTypeNumber) + 1},
			struct{ TxType struct{} }{},
			struct{}{},
		}

		for _, val := range testTypes {
			buf, err := utils.EncodeMsgPack(val)
			So(err, ShouldBeNil)

			var out pi.Transaction
			err = utils.DecodeMsgPack(buf.Bytes(), &out)
			So(err, ShouldNotBeNil)
		}
	})
	Convey("decode wrapper with nil transaction", t, func() {
		var t pi.Transaction
		t = pi.WrapTransaction(nil)
		buf, err := utils.EncodeMsgPack(t)
		So(err, ShouldBeNil)

		var out pi.Transaction
		err = utils.DecodeMsgPack(buf.Bytes(), &out)
		So(err, ShouldBeNil)
		So(out, ShouldBeNil)
	})
}
