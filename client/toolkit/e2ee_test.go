/*
 * Copyright 2019 The CovenantSQL Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package toolkit

import (
	"bytes"
	"crypto/aes"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

const (
	password = "CovenantSQL.io"
)

func TestEncryptDecrypt(t *testing.T) {
	Convey("encrypt & decrypt 0 length string with aes128", t, func() {
		enc, err := Encrypt([]byte(""), []byte(password))
		So(enc, ShouldNotBeNil)
		So(len(enc), ShouldEqual, 2*aes.BlockSize)
		So(err, ShouldBeNil)

		dec, err := Decrypt(enc, []byte(password))
		So(dec, ShouldNotBeNil)
		So(len(dec), ShouldEqual, 0)
		So(err, ShouldBeNil)
	})

	Convey("encrypt & decrypt 0 length bytes with aes128", t, func() {
		enc, err := Encrypt([]byte(nil), []byte(password))
		So(enc, ShouldNotBeNil)
		So(len(enc), ShouldEqual, 2*aes.BlockSize)
		So(err, ShouldBeNil)

		dec, err := Decrypt(enc, []byte(password))
		So(dec, ShouldNotBeNil)
		So(len(dec), ShouldEqual, 0)
		So(err, ShouldBeNil)
	})

	Convey("encrypt & decrypt 1 byte with aes128", t, func() {
		enc, err := Encrypt([]byte{0x11}, []byte(password))
		So(enc, ShouldNotBeNil)
		So(len(enc), ShouldEqual, 2*aes.BlockSize)
		So(err, ShouldBeNil)

		dec, err := Decrypt(enc, []byte(password))
		So(dec, ShouldResemble, []byte{0x11})
		So(len(dec), ShouldEqual, 1)
		So(err, ShouldBeNil)
	})

	Convey("encrypt & decrypt 1747 length bytes", t, func() {
		in := bytes.Repeat([]byte{0xff}, 1747)
		enc, err := Encrypt(in, []byte(password))
		So(enc, ShouldNotBeNil)
		So(len(enc), ShouldEqual, (1747/aes.BlockSize+2)*aes.BlockSize)
		So(err, ShouldBeNil)

		dec, err := Decrypt(enc, []byte(password))
		So(dec, ShouldResemble, in)
		So(len(dec), ShouldEqual, 1747)
		So(err, ShouldBeNil)
	})

	Convey("encrypt & decrypt 32 length bytes", t, func() {
		in := bytes.Repeat([]byte{0xcc}, 32)
		enc, err := Encrypt(in, []byte(password))
		So(enc, ShouldNotBeNil)
		So(len(enc), ShouldEqual, (32/aes.BlockSize+2)*aes.BlockSize)
		So(err, ShouldBeNil)

		dec, err := Decrypt(enc, []byte(password))
		So(dec, ShouldResemble, in)
		So(len(dec), ShouldEqual, 32)
		So(err, ShouldBeNil)
	})

	Convey("decrypt error length bytes", t, func() {
		in := bytes.Repeat([]byte{0xaa}, 1747)
		dec, err := Decrypt(in, []byte(password))
		So(dec, ShouldBeNil)
		So(err.Error(), ShouldEqual, "cipher data size not match")
	})
}
