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

import ec "github.com/btcsuite/btcd/btcec"

// GenECDHSharedSecret is just a wrapper of ec.GenerateSharedSecret which
// generates a shared secret based on a private key and a
// public key using Diffie-Hellman key exchange (ECDH) (RFC 4753).
// RFC5903 Section 9 states we should only return x.
// Key Feature:
// 		GenECDHSharedSecret(BPub, APriv) == GenECDHSharedSecret(APub, BPriv)
func GenECDHSharedSecret(privateKey *PrivateKey, publicKey *PublicKey) []byte {
	return ec.GenerateSharedSecret((*ec.PrivateKey)(privateKey), (*ec.PublicKey)(publicKey))
}
