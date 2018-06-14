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
	"github.com/thunderdb/ThunderDB/crypto/asymmetric"
	"github.com/thunderdb/ThunderDB/crypto/symmetric"
)

const (
	privateKeyPath = "./.testprivatekey"
	password       = "auxten"
)

//func TestSaveLoadPrivateKey(t *testing.T) {
//	Convey("save and load", t, func() {
//		pass := ""
//		privateKeyBytes, _ := hex.DecodeString("")
//		privateKey, _ := ec.PrivKeyFromBytes(ec.S256(), privateKeyBytes)
//		SavePrivateKey("../../keys/private.key", privateKey, []byte(pass))
//		pl, _ := LoadPrivateKey("../../keys/private.key", []byte(pass))
//		So(bytes.Compare(pl.Serialize(), privateKey.Serialize()), ShouldBeZeroValue)
//		So(bytes.Compare(privateKeyBytes, privateKey.Serialize()), ShouldBeZeroValue)
//		publicKeyBytes, _ := hex.DecodeString(BPPublicKeyStr)
//		publicKey, _ := ec.ParsePubKey(publicKeyBytes, ec.S256())
//		So(pl.PubKey().IsEqual(publicKey), ShouldBeTrue)
//	})
//}

func TestLoadPrivateKey(t *testing.T) {
	Convey("save and load", t, func() {
		defer os.Remove(privateKeyPath)
		pk, _, err := asymmetric.GenSecp256k1KeyPair()
		So(pk, ShouldNotBeNil)
		So(err, ShouldBeNil)
		err = SavePrivateKey(privateKeyPath, pk, []byte(password))
		So(err, ShouldBeNil)

		lk, err := LoadPrivateKey(privateKeyPath, []byte(password))
		So(err, ShouldBeNil)
		So(string(pk.Serialize()), ShouldEqual, string(lk.Serialize()))
	})
	Convey("load error", t, func() {
		lk, err := LoadPrivateKey("/path/not/exist", []byte(password))
		So(err, ShouldNotBeNil)
		So(lk, ShouldBeNil)
	})
	Convey("empty key file", t, func() {
		defer os.Remove("./.empty")
		os.Create("./.empty")
		lk, err := LoadPrivateKey("./.empty", []byte(password))
		So(err, ShouldEqual, symmetric.ErrInputSize)
		So(lk, ShouldBeNil)
	})
	Convey("not key file1", t, func() {
		lk, err := LoadPrivateKey("doc.go", []byte(password))
		So(err, ShouldEqual, symmetric.ErrInputSize)
		So(lk, ShouldBeNil)
	})
	Convey("not key file2", t, func() {
		defer os.Remove("./.notkey")
		enc, _ := symmetric.EncryptWithPassword([]byte("aa"), []byte(password))
		ioutil.WriteFile("./.notkey", enc, 0600)
		lk, err := LoadPrivateKey("./.notkey", []byte(password))
		So(err, ShouldEqual, ErrNotKeyFile)
		So(lk, ShouldBeNil)
	})
	Convey("hash not match", t, func() {
		defer os.Remove("./.HashNotMatch")
		enc, _ := symmetric.EncryptWithPassword(bytes.Repeat([]byte("a"), 64), []byte(password))
		ioutil.WriteFile("./.HashNotMatch", enc, 0600)
		lk, err := LoadPrivateKey("./.HashNotMatch", []byte(password))
		So(err, ShouldEqual, ErrHashNotMatch)
		So(lk, ShouldBeNil)
	})
}

func TestInitLocalKeyPair(t *testing.T) {
	Convey("InitLocalKeyPair", t, func() {
		defer os.Remove(privateKeyPath)
		err := InitLocalKeyPair(privateKeyPath, []byte(password))
		So(err, ShouldBeNil)
		gotPrivate1, _ := GetLocalPrivateKey()
		gotPublic1, _ := GetLocalPublicKey()
		So(gotPrivate1, ShouldNotBeNil)
		So(gotPublic1, ShouldNotBeNil)

		err = InitLocalKeyPair(privateKeyPath, []byte(password))
		So(err, ShouldBeNil)
		err = InitLocalKeyPair("/path/not/exist", []byte(password))
		So(err, ShouldNotBeNil)

		err = InitLocalKeyPair(privateKeyPath, []byte("aa"))
		So(err, ShouldNotBeNil)
		gotPrivate2, _ := GetLocalPrivateKey()
		gotPublic2, _ := GetLocalPublicKey()
		So(gotPrivate2, ShouldNotBeNil)
		So(gotPublic2, ShouldNotBeNil)

		So(bytes.Compare(gotPrivate1.Serialize(), gotPrivate2.Serialize()), ShouldBeZeroValue)
		So(gotPublic1.IsEqual(gotPublic2), ShouldBeTrue)
		So(gotPrivate1.PubKey().IsEqual(gotPublic2), ShouldBeTrue)
	})
}

func TestInitLocalKeyPair_error(t *testing.T) {
	Convey("hash not match", t, func() {
		defer os.Remove("./.HashNotMatch")
		enc, _ := symmetric.EncryptWithPassword(bytes.Repeat([]byte("a"), 64), []byte(password))
		ioutil.WriteFile("./.HashNotMatch", enc, 0600)
		err := InitLocalKeyPair("./.HashNotMatch", []byte(password))
		So(err, ShouldEqual, ErrHashNotMatch)
	})
	Convey("ErrNotKeyFile", t, func() {
		defer os.Remove("./.ErrNotKeyFile")
		enc, _ := symmetric.EncryptWithPassword(bytes.Repeat([]byte("a"), 65), []byte(password))
		ioutil.WriteFile("./.ErrNotKeyFile", enc, 0600)
		err := InitLocalKeyPair("./.ErrNotKeyFile", []byte(password))
		So(err, ShouldEqual, ErrNotKeyFile)
	})
}
