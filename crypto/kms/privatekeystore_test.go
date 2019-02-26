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

package kms

import (
	"bytes"
	"encoding/hex"
	"io/ioutil"
	"os"
	"testing"

	"github.com/btcsuite/btcutil/base58"
	"github.com/pkg/errors"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/crypto/symmetric"
)

const (
	privateKeyPath = "./.testprivatekey"
	password       = "auxten"
	salt           = "auxten-key-salt-auxten"
)

func TestSaveLoadPrivateKey(t *testing.T) {
	Convey("save and load", t, func() {
		pass := ""
		privateKeyBytes, _ := hex.DecodeString("f7c0bc718eb0df81e796a11e6f62e23cd2be0a4bdcca30df40d4d915cc3be3ff")
		privateKey, _ := asymmetric.PrivKeyFromBytes(privateKeyBytes)
		SavePrivateKey("private.key", privateKey, []byte(pass))
		pl, _ := LoadPrivateKey("private.key", []byte(pass))
		So(bytes.Compare(pl.Serialize(), privateKey.Serialize()), ShouldBeZeroValue)
		So(bytes.Compare(privateKeyBytes, privateKey.Serialize()), ShouldBeZeroValue)
		publicKeyBytes, _ := hex.DecodeString("02c76216704d797c64c58bc11519fb68582e8e63de7e5b3b2dbbbe8733efe5fd24")
		publicKey, _ := asymmetric.ParsePubKey(publicKeyBytes)
		So(pl.PubKey().IsEqual(publicKey), ShouldBeTrue)
	})
}

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
		So(errors.Cause(err), ShouldEqual, symmetric.ErrInputSize)
		So(lk, ShouldBeNil)
	})
	Convey("not key file1", t, func() {
		lk, err := LoadPrivateKey("doc.go", []byte(password))
		So(errors.Cause(err), ShouldEqual, symmetric.ErrInputSize)
		So(lk, ShouldBeNil)
	})
	Convey("not key file2", t, func() {
		defer os.Remove("./.notkey")
		enc, _ := symmetric.EncryptWithPassword([]byte("aa"), []byte(password), []byte(salt))
		ioutil.WriteFile("./.notkey", enc, 0600)
		lk, err := LoadPrivateKey("./.notkey", []byte(password))
		So(errors.Cause(err), ShouldEqual, ErrNotKeyFile)
		So(lk, ShouldBeNil)
	})
	Convey("hash not match", t, func() {
		defer os.Remove("./.HashNotMatch")
		enc, _ := symmetric.EncryptWithPassword(bytes.Repeat([]byte("a"), 64), []byte(password), []byte(salt))
		ioutil.WriteFile("./.HashNotMatch", enc, 0600)
		lk, err := LoadPrivateKey("./.HashNotMatch", []byte(password))
		So(err, ShouldEqual, ErrHashNotMatch)
		So(lk, ShouldBeNil)
	})
	Convey("invalid base58 version", t, func() {
		defer os.Remove("./.Base58VersionNotMatch")
		var invalidPrivateKeyStoreVersion byte = 0x1
		privateKeyBytes, _ := hex.DecodeString("f7c0bc718eb0df81e796a11e6f62e23cd2be0a4bdcca30df40d4d915cc3be3ff")
		privateKey, _ := asymmetric.PrivKeyFromBytes(privateKeyBytes)
		serializedKey := privateKey.Serialize()
		keyHash := hash.DoubleHashB(serializedKey)
		rawData := append(keyHash, serializedKey...)
		encKey, _ := symmetric.EncryptWithPassword(rawData, []byte(password), []byte(salt))
		invalidBase58EncKey := base58.CheckEncode(encKey, invalidPrivateKeyStoreVersion)
		ioutil.WriteFile("./.Base58VersionNotMatch", []byte(invalidBase58EncKey), 0600)
		lk, err := LoadPrivateKey("./.Base58VersionNotMatch", []byte(password))
		So(err, ShouldEqual, ErrInvalidBase58Version)
		So(lk, ShouldBeNil)
	})
	Convey("invalid base58 checksum", t, func() {
		defer os.Remove("./.Base58InvalidChecksum")
		invalidBase58Str := "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
		ioutil.WriteFile("./.Base58InvalidChecksum", []byte(invalidBase58Str), 0600)
		lk, err := LoadPrivateKey("./.Base58InvalidChecksum", []byte(password))
		So(err, ShouldEqual, base58.ErrChecksum)
		So(lk, ShouldBeNil)
	})
}

func TestInitLocalKeyPair(t *testing.T) {
	Convey("InitLocalKeyPair", t, func() {
		conf.GConf.GenerateKeyPair = true
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
		enc, _ := symmetric.EncryptWithPassword(bytes.Repeat([]byte("a"), 64), []byte(password), []byte(salt))
		ioutil.WriteFile("./.HashNotMatch", enc, 0600)
		err := InitLocalKeyPair("./.HashNotMatch", []byte(password))
		So(err, ShouldEqual, ErrHashNotMatch)
	})
	Convey("ErrNotKeyFile", t, func() {
		defer os.Remove("./.ErrNotKeyFile")
		enc, _ := symmetric.EncryptWithPassword(bytes.Repeat([]byte("a"), 65), []byte(password), []byte(salt))
		ioutil.WriteFile("./.ErrNotKeyFile", enc, 0600)
		err := InitLocalKeyPair("./.ErrNotKeyFile", []byte(password))
		So(errors.Cause(err), ShouldEqual, ErrNotKeyFile)
	})
}
