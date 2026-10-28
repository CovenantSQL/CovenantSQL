/*
 * Copyright 2026 The CovenantSQL Authors.
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

package crypto

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
)

// Ciphertexts recorded from the btcsuite/btcd/btcec v1 Encrypt implementation.
// Data encrypted by earlier releases must stay decryptable.
func TestCompatDecrypt(t *testing.T) {
	const priv = "54c54aaab164f481fb3bb54d9081faba0f5981afc0fbf01da46caa330fc651e2"
	const plain = "436f76656e616e7453514c20454349455320636f6d7061746962696c69747920766563746f72"
	vectors := []struct {
		name, plain, ct string
	}{
		{
			name:  "full length shared secret",
			plain: plain,
			ct:    "ef588c5186c66f62e0aa944151cb7bf402ca002049a82bac752024596da5b5ca24ec5be669805fed0525a4686eee2ff8e08d93cd0020248f0b63c5f8fbd1e4ef13e937542050aae928c10b21d6ab61e22270caace34720f03913a4f0a98ba3656fa9de083fb00b26a9f054e01b5aec13371966e2f63c0b824171ac85d2ecaa5e60faed3f85b8078c15c9900ee8de594d1c834e8371bf60108f039cc48d2ee42846b18bf862e0",
		},
		{
			// The ephemeral shared X coordinate has a leading zero byte.
			name:  "31 byte shared secret",
			plain: plain,
			ct:    "8f43a50baaa2fe3baf11be96b7378cc402ca002062f003cc2d1368a04597284549a45f1ca09bfc84297fc576c3f7f598724f67ea00201ccff29305772c216200f2ad9b494083d7d9031f2920bedd97826214599c10c4e77fc92fbc534cbf106a25ab7d6919056da4dba33d7e6114115aea762d7eb72c19d66664230a1fdae2bfaac15474fd509c305d139962203815f917df6013bb5809b97bde099bb5fb918c95ccccc77974",
		},
		{
			name:  "empty plaintext",
			plain: "",
			ct:    "e4a7ec3af60accd90da4134765ac4c1402ca002061f2252401ac06ca8aaf52f18289f05fd021d0f18f58be8fdda3829a36fd163e00203d4402345894fadeeef24f727b204c28e68e38583b18c02227af1c68b7fea2f2d41d57b8e8dde1a411efcabb82fb9aa1327b2070379ccfb96904db1caf9ff44c787420ec40bfb6429e4584e5eea9b536",
		},
	}

	privBytes, _ := hex.DecodeString(priv)
	privKey, _ := asymmetric.PrivKeyFromBytes(privBytes)
	for _, v := range vectors {
		ct, _ := hex.DecodeString(v.ct)
		want, _ := hex.DecodeString(v.plain)
		got, err := DecryptAndCheck(privKey, ct)
		if err != nil {
			t.Fatalf("%s: decrypt: %v", v.name, err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s: got %x, want %x", v.name, got, want)
		}
	}
}
