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
	"encoding/json"
	"reflect"
	"sync"

	"github.com/pkg/errors"
	"github.com/ugorji/go/codec"

	"github.com/CovenantSQL/CovenantSQL/utils"
)

var (
	txTypeMapping sync.Map
	txType        = reflect.TypeOf((*Transaction)(nil)).Elem()
	txWrapperType = reflect.TypeOf((*TransactionWrapper)(nil))

	// ErrInvalidContainerType represents invalid container type read from msgpack bytes.
	ErrInvalidContainerType = errors.New("invalid container type for TransactionWrapper")
	// ErrInvalidTransactionType represents invalid transaction type read from msgpack bytes.
	ErrInvalidTransactionType = errors.New("invalid transaction type, can not instantiate transaction")
	// ErrTransactionRegistration represents invalid transaction object type being registered.
	ErrTransactionRegistration = errors.New("transaction register failed")

	// arrayHandle splits the [type, transaction] array into its elements. The
	// split decodes from the byte slice instead of going through
	// utils.DecodeMsgPack: a decoder reading from an io.Reader returns each
	// codec.Raw as a view of its buffer, which reading the next element
	// overwrites.
	arrayHandle = &codec.MsgpackHandle{WriteExt: true}
)

func init() {
	// register transaction wrapper to msgpack handler
	if err := utils.RegisterInterfaceToMsgPack(txType, txWrapperType); err != nil {
		panic(err)
	}
}

// TransactionWrapper is the wrapper for Transaction interface for serialization/deserialization purpose.
type TransactionWrapper struct {
	Transaction
}

// Unwrap returns transaction within wrapper.
func (w *TransactionWrapper) Unwrap() Transaction {
	return w.Transaction
}

// CodecEncodeSelf implements codec.Selfer interface.
func (w *TransactionWrapper) CodecEncodeSelf(e *codec.Encoder) {
	if w == nil || w.Transaction == nil {
		e.MustEncode(nil)
		return
	}

	// translate wrapper to two fields array: [type, transaction]
	e.MustEncode([]interface{}{uint64(w.GetTransactionType()), w.Transaction})
}

// CodecDecodeSelf implements codec.Selfer interface.
func (w *TransactionWrapper) CodecDecodeSelf(d *codec.Decoder) {
	// clear fields
	w.Transaction = nil

	// The decoder does not tell the type of the next value before decoding it,
	// so read the whole value and look at its first byte. raw is only valid
	// until d reads again.
	var raw codec.Raw
	d.MustDecode(&raw)

	switch {
	case isMsgpackArray(raw):
		w.decodeFromWrapper(raw)
	case isMsgpackMap(raw):
		w.decodeFromRaw(raw)
	default:
		panic(errors.Wrapf(ErrInvalidContainerType, "type %#x applied", raw[0]))
	}
}

// MarshalJSON implements json.Marshaler interface.
func (w TransactionWrapper) MarshalJSON() ([]byte, error) {
	return json.Marshal(w.Transaction)
}

// UnmarshalJSON implements json.Unmarshaler interface.
func (w *TransactionWrapper) UnmarshalJSON(data []byte) (err error) {
	// detect type from current bytes
	var typeDetector TransactionTypeMixin
	typeDetector.SetTransactionType(TransactionTypeNumber)

	if err = json.Unmarshal(data, &typeDetector); err != nil {
		err = errors.Wrap(err, "try decode transaction failed")
		return
	}

	txType := typeDetector.GetTransactionType()

	if txType == TransactionTypeNumber {
		err = errors.Wrapf(ErrInvalidTransactionType, "invalid tx type: %d", txType)
		return
	}

	if w.Transaction, err = NewTransaction(txType); err != nil {
		err = errors.Wrapf(err, "instantiate transaction type %s failed", txType.String())
		return
	}

	return json.Unmarshal(data, &w.Transaction)
}

func (w *TransactionWrapper) decodeFromWrapper(raw codec.Raw) {
	var fields []codec.Raw
	if err := codec.NewDecoderBytes(raw, arrayHandle).Decode(&fields); err != nil {
		panic(err)
	}

	// fields after the first two are ignored
	if len(fields) < 2 || fields[0] == nil {
		// invalid type, can not instantiate transaction
		panic(ErrInvalidTransactionType)
	}

	var txType TransactionType
	if err := utils.DecodeMsgPack(fields[0], &txType); err != nil {
		panic(err)
	}

	var err error
	if w.Transaction, err = NewTransaction(txType); err != nil {
		panic(err)
	}

	// the container type should be struct
	if !isMsgpackMap(fields[1]) {
		panic(errors.Wrap(ErrInvalidContainerType, "transaction is not a map"))
	}

	if err = utils.DecodeMsgPack(fields[1], &w.Transaction); err != nil {
		panic(err)
	}
}

func (w *TransactionWrapper) decodeFromRaw(rawBytes codec.Raw) {
	var typeDetector TransactionTypeMixin
	typeDetector.SetTransactionType(TransactionTypeNumber)

	var err error
	if err = utils.DecodeMsgPack(rawBytes, &typeDetector); err != nil {
		panic(err)
	}

	txType := typeDetector.GetTransactionType()

	if txType == TransactionTypeNumber {
		panic(ErrInvalidTransactionType)
	}

	if w.Transaction, err = NewTransaction(txType); err != nil {
		panic(err)
	}

	if err = utils.DecodeMsgPack(rawBytes, w.Transaction); err != nil {
		panic(err)
	}
}

// isMsgpackArray reports whether raw is a msgpack array (fixarray, array 16 or
// array 32, see https://github.com/msgpack/msgpack/blob/master/spec.md#formats).
func isMsgpackArray(raw codec.Raw) bool {
	return len(raw) > 0 && (raw[0]&0xf0 == 0x90 || raw[0] == 0xdc || raw[0] == 0xdd)
}

// isMsgpackMap reports whether raw is a msgpack map (fixmap, map 16 or map 32).
func isMsgpackMap(raw codec.Raw) bool {
	return len(raw) > 0 && (raw[0]&0xf0 == 0x80 || raw[0] == 0xde || raw[0] == 0xdf)
}

// RegisterTransaction registers transaction type to wrapper.
func RegisterTransaction(t TransactionType, tx Transaction) {
	if tx == nil {
		panic(ErrTransactionRegistration)
	}
	rt := reflect.TypeOf(tx)

	if rt == txWrapperType {
		panic(ErrTransactionRegistration)
	}

	txTypeMapping.Store(t, rt)
}

// NewTransaction instantiates new transaction object.
func NewTransaction(t TransactionType) (tx Transaction, err error) {
	var d interface{}
	var ok bool
	var rt reflect.Type

	if d, ok = txTypeMapping.Load(t); !ok {
		err = errors.Wrapf(ErrInvalidTransactionType, "transaction not registered")
		return
	}
	rt = d.(reflect.Type)

	if !rt.Implements(txType) || rt == txWrapperType {
		err = errors.Wrap(ErrInvalidTransactionType, "invalid transaction registered")
		return
	}

	var rv reflect.Value

	if rt.Kind() == reflect.Ptr {
		rv = reflect.New(rt.Elem())
	} else {
		rv = reflect.New(rt).Elem()
	}

	rawTx := rv.Interface()
	tx = rawTx.(Transaction)

	if txTypeAwareness, ok := rawTx.(ContainsTransactionTypeMixin); ok {
		txTypeAwareness.SetTransactionType(t)
	}

	return
}

// WrapTransaction wraps transaction in wrapper.
func WrapTransaction(tx Transaction) *TransactionWrapper {
	return &TransactionWrapper{
		Transaction: tx,
	}
}
