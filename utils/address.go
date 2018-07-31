/*
 * Copyright 2018 The ThunderDB Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the “License”);
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an “AS IS” BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package utils

import (
	"github.com/btcsuite/btcutil/base58"
	"gitlab.com/thunderdb/ThunderDB/crypto/asymmetric"
	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
)

const (
	MainNet byte = 0x0
	TestNet byte = 0x6f
)

// PubKey2Addr converts the pubKey to a address
// and the format refers to https://bitcoin.org/en/developer-guide#standard-transactions
func PubKey2Addr(pubKey *asymmetric.PublicKey, version byte) (string, error) {
	enc, err := pubKey.MarshalBinary()
	if err != nil {
		return "", err
	}
	h := hash.THashB(enc[:])
	return base58.CheckEncode(h[:], version), nil
}
