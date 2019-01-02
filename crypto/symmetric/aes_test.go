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

package symmetric

import (
	"bytes"
	"crypto/aes"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

const (
	password = "CovenantSQL.io"
	salt     = "auxten-key-salt-auxten"
)

func TestEncryptDecryptWithPassword(t *testing.T) {
	Convey("encrypt & decrypt 0 length bytes with aes256", t, func() {
		enc, err := EncryptWithPassword([]byte(nil), []byte(password), []byte(salt))
		So(enc, ShouldNotBeNil)
		So(len(enc), ShouldEqual, 2*aes.BlockSize)
		So(err, ShouldBeNil)

		dec, err := DecryptWithPassword(enc, []byte(password), []byte(salt))
		So(dec, ShouldNotBeNil)
		So(len(dec), ShouldEqual, 0)
		So(err, ShouldBeNil)
	})

	Convey("encrypt & decrypt 1747 length bytes", t, func() {
		in := bytes.Repeat([]byte{0xff}, 1747)
		enc, err := EncryptWithPassword(in, []byte(password), []byte(salt))
		So(enc, ShouldNotBeNil)
		So(len(enc), ShouldEqual, (1747/aes.BlockSize+2)*aes.BlockSize)
		So(err, ShouldBeNil)

		dec, err := DecryptWithPassword(enc, []byte(password), []byte(salt))
		So(dec, ShouldNotBeNil)
		So(len(dec), ShouldEqual, 1747)
		So(err, ShouldBeNil)
	})

	Convey("decrypt error length bytes", t, func() {
		in := bytes.Repeat([]byte{0xff}, 1747)
		dec, err := DecryptWithPassword(in, []byte(password), []byte(salt))
		So(dec, ShouldBeNil)
		So(err, ShouldEqual, ErrInputSize)
	})
}
