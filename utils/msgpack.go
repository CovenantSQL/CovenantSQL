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

package utils

import (
	"bytes"
	"reflect"

	"github.com/ugorji/go/codec"
)

var (
	msgpackHandle = &codec.MsgpackHandle{
		WriteExt:    true,
		RawToString: true,
	}
)

// RegisterInterfaceToMsgPack binds interface decode/encode to specified implementation.
func RegisterInterfaceToMsgPack(intf, impl reflect.Type) (err error) {
	return msgpackHandle.Intf2Impl(intf, impl)
}

// DecodeMsgPack reverses the encode operation on a byte slice input.
func DecodeMsgPack(buf []byte, out interface{}) error {
	r := bytes.NewBuffer(buf)
	dec := codec.NewDecoder(r, msgpackHandle)
	return dec.Decode(out)
}

// DecodeMsgPackPlain reverses the encode operation on a byte slice input without RawToString setting.
func DecodeMsgPackPlain(buf []byte, out interface{}) error {
	r := bytes.NewBuffer(buf)
	hd := &codec.MsgpackHandle{
		WriteExt: true,
	}
	dec := codec.NewDecoder(r, hd)
	return dec.Decode(out)
}

// EncodeMsgPack writes an encoded object to a new bytes buffer.
func EncodeMsgPack(in interface{}) (*bytes.Buffer, error) {
	buf := bytes.NewBuffer(nil)
	enc := codec.NewEncoder(buf, msgpackHandle)
	err := enc.Encode(in)
	return buf, err
}
