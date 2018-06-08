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

package asymmetric

import (
	"bytes"
	"testing"

	"time"

	ec "github.com/btcsuite/btcd/btcec"
	log "github.com/sirupsen/logrus"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/thunderdb/ThunderDB/crypto"
)

func TestGenSecp256k1Keypair(t *testing.T) {
	privateKey, publicKey, err := GenSecp256k1KeyPair()
	log.Debugf("privateKey: %x", privateKey.Serialize())
	log.Debugf("publicKey: %x", publicKey.SerializeCompressed())
	if err != nil {
		t.Fatal("failed to generate private key")
	}

	in := []byte("Hey there dude. How are you doing? This is a test.")

	out, err := crypto.EncryptAndSign(publicKey, in)
	if err != nil {
		t.Fatal("failed to encrypt:", err)
	}

	dec, err := crypto.DecryptAndCheck(privateKey, out)
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
		log.Infof("publicKey: %x", publicKey.SerializeCompressed())

		nonce := GetPubKeyNonce(publicKey, 10, 200*time.Millisecond, nil)
		log.Infof("nonce: %v", nonce)
		// sometimes nonce difficulty can be little bit higher than expected
		So(nonce.Difficulty, ShouldBeLessThanOrEqualTo, 20)
	})

}

func TestGetThePubKeyNonce(t *testing.T) {
	Convey("translate key error", t, func() {
		publicKey, _ := ec.ParsePubKey([]byte{
			0x02,
			0xc1, 0xdb, 0x96, 0xf2, 0xba, 0x7e, 0x1c, 0xb4,
			0xe9, 0x82, 0x2d, 0x12, 0xde, 0x0f, 0x63, 0xfb,
			0x66, 0x6f, 0xeb, 0x82, 0x8c, 0x7f, 0x50, 0x9e,
			0x81, 0xfa, 0xb9, 0xbd, 0x7a, 0x34, 0x03, 0x9c,
		}, ec.S256())
		nonce := GetPubKeyNonce(publicKey, 256, 1*time.Second, nil)

		log.Infof("nonce: %v", nonce)
		// sometimes nonce difficulty can be little bit higher than expected
		So(nonce.Difficulty, ShouldBeLessThanOrEqualTo, 40)
	})

}
