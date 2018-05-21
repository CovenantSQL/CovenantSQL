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

	. "github.com/smartystreets/goconvey/convey"
	"github.com/thunderdb/ThunderDB/crypto"
)

func TestGenSecp256k1Keypair(t *testing.T) {
	privateKey, publicKey, err := GenSecp256k1Keypair()
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
	privateKey1, publicKey1, _ := GenSecp256k1Keypair()
	privateKey2, publicKey2, _ := GenSecp256k1Keypair()
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
		_, publicKey, err := GenSecp256k1Keypair()
		if err != nil {
			t.Fatal("failed to generate private key")
		}

		nonce := GetPubKeyNonce(publicKey, 10, 200*time.Millisecond)

		// sometimes nonce difficulty can be little bit higher than expected
		So(nonce.Difficulty, ShouldBeLessThanOrEqualTo, 20)
	})

}
