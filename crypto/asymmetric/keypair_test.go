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

package asymmetric

import (
	"bytes"
	"strings"
	"testing"
	"time"

	ec "github.com/btcsuite/btcd/btcec"
	. "github.com/smartystreets/goconvey/convey"
	yaml "gopkg.in/yaml.v2"

	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

func TestGenSecp256k1Keypair(t *testing.T) {
	privateKey, publicKey, err := GenSecp256k1KeyPair()
	log.Debugf("privateKey: %x", privateKey.Serialize())
	log.Debugf("publicKey: %x", publicKey.Serialize())
	if err != nil {
		t.Fatal("failed to generate private key")
	}

	in := []byte("Hey there dude. How are you doing? This is a test.")

	out, err := ec.Encrypt((*ec.PublicKey)(publicKey), in)
	if err != nil {
		t.Fatal("failed to encrypt:", err)
	}

	dec, err := ec.Decrypt((*ec.PrivateKey)(privateKey), out)
	if err != nil {
		t.Fatal("failed to decrypt:", err)
	}

	if !bytes.Equal(in, dec) {
		t.Error("decrypted data doesn't match original")
	}
}

func TestGenECDHSharedSecret(t *testing.T) {
	privateKey1, publicKey1, _ := GenSecp256k1KeyPair()
	privateKey2, publicKey2, _ := GenSecp256k1KeyPair()
	shared1 := GenECDHSharedSecret(privateKey1, publicKey2)
	shared2 := GenECDHSharedSecret(privateKey2, publicKey1)
	if len(shared1) <= 0 {
		t.Errorf("shared length should not be %d", len(shared1))
	}

	for i, b := range shared1 {
		if b != shared2[i] {
			t.Error("shared1 and shared2 should be equel")
		}
	}
	//t.Log(shared1)
}

func TestGetPubKeyNonce(t *testing.T) {
	Convey("translate key error", t, func() {
		privateKey, publicKey, err := GenSecp256k1KeyPair()
		if err != nil {
			t.Fatal("failed to generate private key")
		}
		log.Infof("privateKey: %x", privateKey.Serialize())
		log.Infof("publicKey: %x", publicKey.Serialize())

		nonce := GetPubKeyNonce(publicKey, 20, 500*time.Millisecond, nil)
		log.Infof("nonce: %v", nonce)
		// sometimes nonce difficulty can be little bit higher than expected
		So(nonce.Difficulty, ShouldBeLessThanOrEqualTo, 120)
	})
	Convey("translate key error", t, func() {

	})
}

func TestGetThePubKeyNonce(t *testing.T) {
	Convey("translate key error", t, func() {
		publicKey, _ := ParsePubKey([]byte{
			0x02,
			0xc1, 0xdb, 0x96, 0xf2, 0xba, 0x7e, 0x1c, 0xb4,
			0xe9, 0x82, 0x2d, 0x12, 0xde, 0x0f, 0x63, 0xfb,
			0x66, 0x6f, 0xeb, 0x82, 0x8c, 0x7f, 0x50, 0x9e,
			0x81, 0xfa, 0xb9, 0xbd, 0x7a, 0x34, 0x03, 0x9c,
		})
		nonce := GetPubKeyNonce(publicKey, 256, 1*time.Second, nil)

		log.Infof("nonce: %v", nonce)
		// sometimes nonce difficulty can be little bit higher than expected
		So(nonce.Difficulty, ShouldBeLessThanOrEqualTo, 40)
	})
}

func TestPubKey(t *testing.T) {
	pubKeyTests := []struct {
		name string
		key  []byte
	}{
		{
			name: "Test serialize",
			key: []byte{
				0x02,
				0xc1, 0xdb, 0x96, 0xf2, 0xba, 0x7e, 0x1c, 0xb4,
				0xe9, 0x82, 0x2d, 0x12, 0xde, 0x0f, 0x63, 0xfb,
				0x66, 0x6f, 0xeb, 0x82, 0x8c, 0x7f, 0x50, 0x9e,
				0x81, 0xfa, 0xb9, 0xbd, 0x7a, 0x34, 0x03, 0x9c,
			},
		},
	}

	for _, test := range pubKeyTests {
		pubKey, err := ParsePubKey(test.key)

		if err != nil {
			t.Errorf("%s could not parse public key: %v", test.name, err)
			continue
		}

		serializedKey := pubKey.Serialize()

		if !bytes.Equal(test.key, serializedKey) {
			t.Errorf("%s unexpected serialized bytes - got: %x, want: %x", test.name,
				serializedKey, test.key)
		}
	}
}

func TestPublicKey_MarshalBinary(t *testing.T) {
	Convey("marshal unmarshal public key", t, func() {
		_, publicKey, _ := GenSecp256k1KeyPair()
		publicKey2 := new(PublicKey)
		buf, _ := publicKey.MarshalBinary()
		publicKey2.UnmarshalBinary(buf)

		So(publicKey.IsEqual(publicKey2), ShouldBeTrue)

		publicKey3 := new(PublicKey)
		publicKey3.UnmarshalBinary(buf)

		buf1, _ := publicKey.MarshalHash()
		buf2, _ := publicKey2.MarshalHash()
		buf3, _ := publicKey3.MarshalHash()

		So(buf1, ShouldResemble, buf2)
		So(buf1, ShouldResemble, buf3)
		So(publicKey.Msgsize(), ShouldEqual, publicKey2.Msgsize())
	})

	Convey("unmarshal from empty key bytes should fail", t, func() {
		pubKey := &PublicKey{}
		So(pubKey.UnmarshalBinary(nil), ShouldNotBeNil)
	})
}

func TestPrivateKey_Serialize(t *testing.T) {
	Convey("marshal unmarshal private key", t, func() {
		pk, _, _ := GenSecp256k1KeyPair()
		buf := pk.Serialize()
		priv2, pub2 := PrivKeyFromBytes(buf)

		So(priv2.PubKey().IsEqual(pub2), ShouldBeTrue)
	})
}

func TestPaddedAppend(t *testing.T) {
	Convey("paddedAppend", t, func() {
		src := []byte("aaaa")
		dst := make([]byte, 0, 16)
		dst = paddedAppend(16, dst, src)

		So(len(dst), ShouldEqual, 16)
		So(dst, ShouldResemble, append([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, src...))
	})
}

func unmarshalAndMarshal(str string) string {
	var key PublicKey
	yaml.Unmarshal([]byte(str), &key)
	ret, _ := yaml.Marshal(key)

	return strings.TrimSpace(string(ret))
}

func TestServerRole_MarshalYAML(t *testing.T) {
	Convey("marshal unmarshal yaml", t, func() {
		So(unmarshalAndMarshal("029e54e333da9ff38acb0f1afd8b425d57ba301539bc7b26a94f1ab663605efbcd"), ShouldEqual, "029e54e333da9ff38acb0f1afd8b425d57ba301539bc7b26a94f1ab663605efbcd")
		So(unmarshalAndMarshal("02c76216704d797c64c58bc11519fb68582e8e63de7e5b3b2dbbbe8733efe5fd24"), ShouldEqual, "02c76216704d797c64c58bc11519fb68582e8e63de7e5b3b2dbbbe8733efe5fd24")
	})
}
