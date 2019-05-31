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

const (
	// msgpack constants, copied from go/codec/msgpack.go
	valueTypeMap   = 9
	valueTypeArray = 10
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
	// ErrMsgPackVersionMismatch represents the msgpack library abi has changed.
	ErrMsgPackVersionMismatch = errors.New("msgpack library version mismatch")
)

func init() {
	// detect msgpack version
	if codec.GenVersion < 8 {
		panic(ErrMsgPackVersionMismatch)
	}

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
	helperEncoder, encDriver := codec.GenHelperEncoder(e)

	if w == nil || w.Transaction == nil {
		encDriver.EncodeNil()
		return
	}

	// translate wrapper to two fields array wrapped by map
	encDriver.WriteArrayStart(2)
	encDriver.WriteArrayElem()
	encDriver.EncodeUint(uint64(w.GetTransactionType()))
	encDriver.WriteArrayElem()
	helperEncoder.EncFallback(w.Transaction)
	encDriver.WriteArrayEnd()
}

// CodecDecodeSelf implements codec.Selfer interface.
func (w *TransactionWrapper) CodecDecodeSelf(d *codec.Decoder) {
	_, decodeDriver := codec.GenHelperDecoder(d)

	// clear fields
	w.Transaction = nil
	ct := decodeDriver.ContainerType()

	switch ct {
	case valueTypeArray:
		w.decodeFromWrapper(d)
	case valueTypeMap:
		w.decodeFromRaw(d)
	default:
		panic(errors.Wrapf(ErrInvalidContainerType, "type %v applied", ct))
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

func (w *TransactionWrapper) decodeFromWrapper(d *codec.Decoder) {
	helperDecoder, decodeDriver := codec.GenHelperDecoder(d)
	containerLen := decodeDriver.ReadArrayStart()

	for i := 0; i < containerLen; i++ {
		if decodeDriver.CheckBreak() {
			break
		}

		decodeDriver.ReadArrayElem()

		if i == 0 {
			if decodeDriver.TryDecodeAsNil() {
				// invalid type, can not instantiate transaction
				panic(ErrInvalidTransactionType)
			} else {
				var txType TransactionType
				helperDecoder.DecFallback(&txType, true)

				var err error
				if w.Transaction, err = NewTransaction(txType); err != nil {
					panic(err)
				}
			}
		} else if i == 1 {
			if ct := decodeDriver.ContainerType(); ct != valueTypeMap {
				panic(errors.Wrapf(ErrInvalidContainerType, "type %v applied", ct))
			}

			if !decodeDriver.TryDecodeAsNil() {
				// the container type should be struct
				helperDecoder.DecFallback(&w.Transaction, true)
			}
		} else {
			helperDecoder.DecStructFieldNotFound(i, "")
		}
	}

	decodeDriver.ReadArrayEnd()

	if containerLen < 2 {
		panic(ErrInvalidTransactionType)
	}
}

func (w *TransactionWrapper) decodeFromRaw(d *codec.Decoder) {
	helperDecoder, _ := codec.GenHelperDecoder(d)

	// read all container as raw
	rawBytes := helperDecoder.DecRaw()

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
