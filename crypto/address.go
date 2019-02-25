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

package crypto

import (
	"github.com/pkg/errors"

	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/proto"
)

// PublicKeyToAddress is an alias to function crypto.PubKeyHash
var PublicKeyToAddress = PubKeyHash

// PubKeyHash generates the account hash address for specified public key.
func PubKeyHash(pubKey *asymmetric.PublicKey) (addr proto.AccountAddress, err error) {
	if !pubKey.IsValid() {
		err = errors.New("invalid public key")
		return
	}
	var enc []byte

	if enc, err = pubKey.MarshalHash(); err != nil {
		return
	}

	addr = proto.AccountAddress(hash.THashH(enc))
	return
}
