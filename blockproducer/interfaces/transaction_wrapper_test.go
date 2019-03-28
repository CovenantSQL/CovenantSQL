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

package interfaces_test

import (
	"encoding/json"
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	pi "github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils"
)

type TestTransactionEncode struct {
	TestField int64
	pi.TransactionTypeMixin
}

func (e *TestTransactionEncode) GetAccountAddress() proto.AccountAddress {
	return proto.AccountAddress{}
}

func (e *TestTransactionEncode) GetAccountNonce() pi.AccountNonce {
	return pi.AccountNonce(0)
}

func (e *TestTransactionEncode) Hash() hash.Hash {
	return hash.Hash{}
}

func (e *TestTransactionEncode) Sign(signer *asymmetric.PrivateKey) error {
	return nil
}

func (e *TestTransactionEncode) Verify() error {
	return nil
}
func (e *TestTransactionEncode) MarshalHash() ([]byte, error) {
	return nil, nil
}

func (e *TestTransactionEncode) Msgsize() int {
	return 0
}

func init() {
	pi.RegisterTransaction(pi.TransactionTypeTransfer, (*TestTransactionEncode)(nil))
}

func TestTransactionWrapper(t *testing.T) {
	Convey("tx wrapper test", t, func() {
		w := &pi.TransactionWrapper{}
		So(w.Unwrap(), ShouldBeNil)

		// nil encode
		buf, err := utils.EncodeMsgPack(w)
		So(err, ShouldBeNil)
		var v interface{}
		err = utils.DecodeMsgPack(buf.Bytes(), &v)
		So(err, ShouldBeNil)
		So(v, ShouldBeNil)

		// encode test
		e := &TestTransactionEncode{}
		e.SetTransactionType(pi.TransactionTypeTransfer)
		buf, err = utils.EncodeMsgPack(e)
		So(err, ShouldBeNil)
		var v2 pi.Transaction
		err = utils.DecodeMsgPack(buf.Bytes(), &v2)
		So(err, ShouldBeNil)
		So(v2.GetTransactionType(), ShouldEqual, pi.TransactionTypeTransfer)

		// encode with wrapper test
		e2 := pi.WrapTransaction(e)
		buf, err = utils.EncodeMsgPack(e2)
		So(err, ShouldBeNil)
		var v3 pi.Transaction
		err = utils.DecodeMsgPack(buf.Bytes(), &v3)
		So(err, ShouldBeNil)
		So(v3.GetTransactionType(), ShouldEqual, pi.TransactionTypeTransfer)
		tw, ok := v3.(*pi.TransactionWrapper)
		So(ok, ShouldBeTrue)
		So(tw.Unwrap().GetTransactionType(), ShouldEqual, pi.TransactionTypeTransfer)

		// test encode non-existence type
		e3 := &TestTransactionEncode{}
		e3.SetTransactionType(pi.TransactionTypeCreateAccount)
		buf, err = utils.EncodeMsgPack(e3)
		So(err, ShouldBeNil)
		var v4 pi.Transaction
		err = utils.DecodeMsgPack(buf.Bytes(), &v4)
		So(err, ShouldNotBeNil)

		// test invalid decode, not enough length
		buf, err = utils.EncodeMsgPack([]uint64{})
		So(err, ShouldBeNil)
		var v5 pi.Transaction
		err = utils.DecodeMsgPack(buf.Bytes(), &v5)
		So(err, ShouldNotBeNil)

		// test invalid decode, invalid tx type
		buf, err = utils.EncodeMsgPack([]uint64{1})
		So(err, ShouldBeNil)
		var v6 pi.Transaction
		err = utils.DecodeMsgPack(buf.Bytes(), &v6)
		So(err, ShouldNotBeNil)

		// test invalid decode, nil type
		buf, err = utils.EncodeMsgPack([]interface{}{nil})
		So(err, ShouldBeNil)
		var v7 pi.Transaction
		err = utils.DecodeMsgPack(buf.Bytes(), &v7)
		So(err, ShouldNotBeNil)

		// test invalid decode, nil payload
		buf, err = utils.EncodeMsgPack([]interface{}{pi.TransactionTypeTransfer, nil})
		So(err, ShouldBeNil)
		var v8 pi.Transaction
		err = utils.DecodeMsgPack(buf.Bytes(), &v8)
		So(err, ShouldNotBeNil)

		// test invalid decode, invalid payload container type
		buf, err = utils.EncodeMsgPack([]interface{}{pi.TransactionTypeTransfer, []uint64{}})
		So(err, ShouldBeNil)
		var v9 pi.Transaction
		err = utils.DecodeMsgPack(buf.Bytes(), &v9)
		So(err, ShouldNotBeNil)

		// extra payload
		buf, err = utils.EncodeMsgPack([]interface{}{pi.TransactionTypeTransfer, e, 1, 2})
		So(err, ShouldBeNil)
		var v10 pi.Transaction
		err = utils.DecodeMsgPack(buf.Bytes(), &v10)
		So(err, ShouldBeNil)

		// test invalid type
		buf, err = utils.EncodeMsgPack(1)
		So(err, ShouldBeNil)
		var v11 pi.Transaction
		err = utils.DecodeMsgPack(buf.Bytes(), &v11)
		So(err, ShouldNotBeNil)

		// test invalid mixin
		buf, err = utils.EncodeMsgPack(map[string]interface{}{"TxType": "invalid type"})
		So(err, ShouldBeNil)
		var v12 pi.Transaction
		err = utils.DecodeMsgPack(buf.Bytes(), &v12)
		So(err, ShouldNotBeNil)

		// test invalid mixin type
		buf, err = utils.EncodeMsgPack(map[string]interface{}{"TxType": pi.TransactionTypeNumber})
		So(err, ShouldBeNil)
		var v13 pi.Transaction
		err = utils.DecodeMsgPack(buf.Bytes(), &v13)
		So(err, ShouldNotBeNil)

		// test tx data
		buf, err = utils.EncodeMsgPack(map[string]interface{}{"TxType": pi.TransactionTypeTransfer, "TestField": 1})
		So(err, ShouldBeNil)
		var v14 pi.Transaction
		err = utils.DecodeMsgPack(buf.Bytes(), &v14)
		So(err, ShouldBeNil)

		// test invalid tx data
		buf, err = utils.EncodeMsgPack(map[string]interface{}{"TxType": pi.TransactionTypeTransfer, "TestField": "happy"})
		So(err, ShouldBeNil)
		var v15 pi.Transaction
		err = utils.DecodeMsgPack(buf.Bytes(), &v15)
		So(err, ShouldNotBeNil)

		// test json marshal and unmarshal
		v16 := &TestTransactionEncode{TestField: 10}
		v16.SetTransactionType(pi.TransactionTypeTransfer)
		var v17 pi.Transaction = v16
		var jsonData []byte
		jsonData, err = json.Marshal(v17)
		So(string(jsonData), ShouldContainSubstring, "TestField")
		So(err, ShouldBeNil)

		var v18 pi.Transaction = &pi.TransactionWrapper{}
		err = json.Unmarshal(jsonData, &v18)
		So(err, ShouldBeNil)
		So(v18.(*pi.TransactionWrapper).Unwrap(), ShouldNotBeNil)
		So(v18.GetTransactionType(), ShouldEqual, pi.TransactionTypeTransfer)
		So(v18.(*pi.TransactionWrapper).Unwrap().(*TestTransactionEncode).TestField, ShouldEqual, 10)

		jsonData, err = json.Marshal(v18)
		So(string(jsonData), ShouldContainSubstring, "TestField")

		v18.(*pi.TransactionWrapper).Transaction = nil
		jsonData = []byte(`{"TxType": 1, "TestField": 11}`)
		err = json.Unmarshal(jsonData, &v18)
		So(err, ShouldBeNil)
		So(v18.GetTransactionType(), ShouldEqual, pi.TransactionTypeTransfer)
		So(v18.(*pi.TransactionWrapper).Unwrap().(*TestTransactionEncode).TestField, ShouldEqual, 11)

		// unmarshal fail cases
		v18.(*pi.TransactionWrapper).Transaction = nil
		jsonData = []byte(`{"TxType": {}, "TestField": 11}`)
		err = json.Unmarshal(jsonData, &v18)
		So(err, ShouldNotBeNil)

		v18.(*pi.TransactionWrapper).Transaction = nil
		jsonData = []byte(fmt.Sprintf(`{"TxType": %d, "TestField": 11}`, pi.TransactionTypeNumber))
		err = json.Unmarshal(jsonData, &v18)
		So(err, ShouldNotBeNil)

		v18.(*pi.TransactionWrapper).Transaction = nil
		jsonData = []byte(fmt.Sprintf(`{"TxType": %d, "TestField": 11}`, pi.TransactionTypeCreateAccount))
		err = json.Unmarshal(jsonData, &v18)
		So(err, ShouldNotBeNil)
	})
}

func TestRegisterTransaction(t *testing.T) {
	Convey("test registration", t, func() {
		So(func() { pi.RegisterTransaction(pi.TransactionTypeTransfer, nil) }, ShouldPanic)
		So(func() { pi.RegisterTransaction(pi.TransactionTypeBaseAccount, (*pi.TransactionWrapper)(nil)) }, ShouldPanic)
	})
}
