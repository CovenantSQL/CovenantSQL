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
	"math/rand"
	"reflect"
	"testing"
	"time"

	"github.com/btcsuite/btcd/btcec"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	priv *PrivateKey
	pub  *PublicKey
)

func init() {
	rand.Seed(time.Now().UnixNano())

	var err error
	priv, pub, err = GenSecp256k1KeyPair()

	if err != nil {
		panic(err)
	}
}

func TestSign(t *testing.T) {
	tests := []struct {
		name string
		key  []byte
	}{
		{
			name: "Test curve",
			key: []byte{
				0xea, 0xf0, 0x2c, 0xa3, 0x48, 0xc5, 0x24, 0xe6,
				0x39, 0x26, 0x55, 0xba, 0x4d, 0x29, 0x60, 0x3c,
				0xd1, 0xa7, 0x34, 0x7d, 0x9d, 0x65, 0xcf, 0xe9,
				0x3c, 0xe1, 0xeb, 0xff, 0xdc, 0xa2, 0x26, 0x94,
			},
		},
	}

	for _, test := range tests {
		priv, pub := PrivKeyFromBytes(test.key)
		hash := []byte{0x0, 0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8, 0x9}
		sig, err := priv.Sign(hash)

		if err != nil {
			t.Errorf("%s could not sign: %v", test.name, err)
			continue
		}

		if !sig.Verify(hash, pub) {
			t.Errorf("%s could not verify: %v", test.name, err)
			continue
		}

		serializedKey := priv.Serialize()
		if !bytes.Equal(serializedKey, test.key) {
			t.Errorf("%s unexpected serialized bytes - got: %x, want: %x", test.name,
				serializedKey, test.key)
		}

		serializedSig := sig.Serialize()
		targetSig, err := ParseDERSignature(serializedSig, btcec.S256())
		if err != nil {
			t.Errorf("%s could not serialized: %v", test.name, err)
			continue
		}
		if !reflect.DeepEqual(sig, targetSig) {
			t.Errorf("%s unexpected serialized bytes - got: %x, want: %x", test.name,
				targetSig, sig)
		}
	}
}

func TestSignature_MarshalBinary(t *testing.T) {
	Convey("marshal unmarshal signature", t, func() {
		// generate
		privateKey, publicKey, _ := GenSecp256k1KeyPair()

		// random data
		buf := make([]byte, 16)
		rand.Read(buf)

		// sign
		var sign *Signature
		sign, _ = privateKey.Sign(buf)

		var err error
		keyBytes, err := sign.MarshalBinary()
		So(err, ShouldBeNil)

		sign2 := new(Signature)
		err = sign2.UnmarshalBinary(keyBytes)
		So(err, ShouldBeNil)

		So(sign2.Verify(buf, publicKey), ShouldBeTrue)
	})

	Convey("test marshal unmarshal nil value", t, func() {
		var sig *Signature
		var err error

		// nil value marshal
		_, err = sig.MarshalBinary()
		So(err, ShouldNotBeNil)

		// nil value unmarshal
		err = sig.UnmarshalBinary([]byte{})
		So(err, ShouldNotBeNil)
	})
}

func BenchmarkGenKey(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, _, err := GenSecp256k1KeyPair(); err != nil {
			b.Fatalf("Error occurred: %v", err)
		}
	}
}

func BenchmarkSign(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		var hash [32]byte
		rand.Read(hash[:])

		b.StartTimer()
		_, err := priv.Sign(hash[:])
		b.StopTimer()

		if err != nil {
			b.Fatalf("Error occurred: %v", err)
		}
	}
}

func BenchmarkVerify(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		var hash [32]byte
		rand.Read(hash[:])
		sig, err := priv.Sign(hash[:])

		if err != nil {
			b.Fatalf("Error occurred: %v", err)
		}

		b.StartTimer()

		if !sig.Verify(hash[:], pub) {
			b.Fatalf("Failed to verify signature")
		}
	}
}

func BenchmarkPublicKeySerialization(b *testing.B) {
	for i := 0; i < b.N; i++ {
		pub.Serialize()
	}
}

func BenchmarkParsePublicKey(b *testing.B) {
	buffer := pub.Serialize()

	for i := 0; i < b.N; i++ {
		_, err := ParsePubKey(buffer)

		if err != nil {
			b.Fatalf("Error occurred: %v", err)
		}
	}
}

func BenchmarkSignatureSerialization(b *testing.B) {
	var hash [32]byte
	rand.Read(hash[:])
	sig, err := priv.Sign(hash[:])

	if err != nil {
		b.Fatalf("Error occurred: %v", err)
	}

	for i := 0; i < b.N; i++ {
		sig.Serialize()
	}
}

func BenchmarkParseSignature(b *testing.B) {
	var hash [32]byte
	rand.Read(hash[:])
	sig, err := priv.Sign(hash[:])
	buffer := sig.Serialize()

	if err != nil {
		b.Fatalf("Error occurred: %v", err)
	}

	for i := 0; i < b.N; i++ {
		_, err := ParseSignature(buffer)

		if err != nil {
			b.Fatalf("Error occurred: %v", err)
		}
	}
}
