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

// Package sign is a wrapper of btcsuite's signature package, except that it only exports types and
// functions which will be used by ThunderDB.
package signature

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/btcsuite/btcd/btcec"
)

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
		priv, pub := PrivKeyFromBytes(btcec.S256(), test.key)
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

func TestPubKey(t *testing.T) {
	pubKeyTests := []struct {
		name string
		key  []byte
	}{
		{
			name: "Test serialize",
			key: []byte{0x04, 0x11, 0xdb, 0x93, 0xe1, 0xdc, 0xdb, 0x8a,
				0x01, 0x6b, 0x49, 0x84, 0x0f, 0x8c, 0x53, 0xbc, 0x1e,
				0xb6, 0x8a, 0x38, 0x2e, 0x97, 0xb1, 0x48, 0x2e, 0xca,
				0xd7, 0xb1, 0x48, 0xa6, 0x90, 0x9a, 0x5c, 0xb2, 0xe0,
				0xea, 0xdd, 0xfb, 0x84, 0xcc, 0xf9, 0x74, 0x44, 0x64,
				0xf8, 0x2e, 0x16, 0x0b, 0xfa, 0x9b, 0x8b, 0x64, 0xf9,
				0xd4, 0xc0, 0x3f, 0x99, 0x9b, 0x86, 0x43, 0xf6, 0x56,
				0xb4, 0x12, 0xa3,
			},
		},
	}

	for _, test := range pubKeyTests {
		pubKey, err := ParsePubKey(test.key, btcec.S256())

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
