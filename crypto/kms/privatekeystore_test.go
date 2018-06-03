/*
 * Copyright 2018 The ThunderDB Authors.
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

package kms

import (
	"testing"

	"os"

	"io/ioutil"

	"bytes"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/thunderdb/ThunderDB/crypto/symmetric"
)

const (
	privateKeyPath = "./.testprivatekey"
	password       = "auxten"
)

func TestLoadPrivateKey(t *testing.T) {
	Convey("save and load", t, func() {
		mk, err := GeneratePrivateKey()
		So(mk, ShouldNotBeNil)
		So(err, ShouldBeNil)
		err = SavePrivateKey(privateKeyPath, mk, []byte(password))
		So(err, ShouldBeNil)

		lk, err := LoadPrivateKey(privateKeyPath, []byte(password))
		So(err, ShouldBeNil)
		So(string(mk.Serialize()), ShouldEqual, string(lk.Serialize()))
		os.Remove(privateKeyPath)
	})
	Convey("load error", t, func() {
		lk, err := LoadPrivateKey("/path/not/exist", []byte(password))
		So(err, ShouldNotBeNil)
		So(lk, ShouldBeNil)
	})
	Convey("empty key file", t, func() {
		os.Create("./.empty")
		lk, err := LoadPrivateKey("./.empty", []byte(password))
		So(err, ShouldEqual, symmetric.ErrInputSize)
		So(lk, ShouldBeNil)
		os.Remove("./.empty")
	})
	Convey("not key file1", t, func() {
		lk, err := LoadPrivateKey("doc.go", []byte(password))
		So(err, ShouldEqual, symmetric.ErrInputSize)
		So(lk, ShouldBeNil)
	})
	Convey("not key file2", t, func() {
		enc, _ := symmetric.EncryptWithPassword([]byte("aa"), []byte(password))
		ioutil.WriteFile("./.notkey", enc, 0600)
		lk, err := LoadPrivateKey("./.notkey", []byte(password))
		So(err, ShouldEqual, ErrNotKeyFile)
		So(lk, ShouldBeNil)
		os.Remove("./.notkey")
	})
	Convey("hash not match", t, func() {
		enc, _ := symmetric.EncryptWithPassword(bytes.Repeat([]byte("a"), 64), []byte(password))
		ioutil.WriteFile("./.HashNotMatch", enc, 0600)
		lk, err := LoadPrivateKey("./.HashNotMatch", []byte(password))
		So(err, ShouldEqual, ErrHashNotMatch)
		So(lk, ShouldBeNil)
		os.Remove("./.HashNotMatch")
	})

}
