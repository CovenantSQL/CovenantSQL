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

package utils

import (
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/btcsuite/btcutil/base58"
)

const (
	// MainNet is the version byte for main net.
	MainNet byte = 0x0
	// TestNet is the version byte for test net.
	TestNet byte = 0x6f
)

// PubKey2Addr converts the pubKey to a address
// and the format refers to https://bitcoin.org/en/developer-guide#standard-transactions
func PubKey2Addr(pubKey *asymmetric.PublicKey, version byte) (addr string, err error) {
	var internalAddr proto.AccountAddress
	if internalAddr, err = PubKeyHash(pubKey); err != nil {
		return
	}

	addr = Hash2Addr(internalAddr, version)
	return
}

// PubKeyHash generates the account hash address for specified public key.
func PubKeyHash(pubKey *asymmetric.PublicKey) (addr proto.AccountAddress, err error) {
	var enc []byte

	if enc, err = pubKey.MarshalHash(); err != nil {
		return
	}

	addr = proto.AccountAddress(hash.THashH(enc))
	return
}

// Addr2Hash converts base58 address to internal account address hash.
func Addr2Hash(addr string) (version byte, internalAddr proto.AccountAddress, err error) {
	var hashBytes []byte
	if hashBytes, version, err = base58.CheckDecode(addr); err != nil {
		return
	}
	var h *hash.Hash
	if h, err = hash.NewHash(hashBytes); err != nil {
		return
	}
	internalAddr = proto.AccountAddress(*h)
	return
}

// Hash2Addr converts interal account address hash to base58 format.
func Hash2Addr(addr proto.AccountAddress, version byte) string {
	return base58.CheckEncode(addr[:], version)
}
