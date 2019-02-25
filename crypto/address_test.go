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
	"encoding/hex"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
)

func TestPubKeyHashAndAddressing(t *testing.T) {
	testPubkeyAndAddr := []struct {
		pubkey string
		addr   string
	}{
		{
			pubkey: "0367aa51809a7c1dc0f82c02452fec9557b3e1d10ce7c919d8e73d90048df86d20",
			addr:   "ba0ba731c7a76ccef2c1170f42038f7e228dfb474ef0190dfe35d9a37911ed37",
		},
		{
			pubkey: "02914bca0806f040dd842207c44474ab41ecd29deee7f2d355789c5c78d448ca16",
			addr:   "1a7b0959bbd0d0ec529278a61c0056c277bffe75b2646e1699b46b10a90210be",
		},
		{
			pubkey: "02ec784ca599f21ef93fe7abdc68d78817ab6c9b31f2324d15ea174d9da498b4c4",
			addr:   "9e1618775cceeb19f110e04fbc6c5bca6c8e4e9b116e193a42fe69bf602e7bcd",
		},
		{
			pubkey: "03ae859eac5b72ee428c7a85f10b2ce748d9de5e480aefbb70f6597dfa8b2175e5",
			addr:   "9235bc4130a2ed4e6c35ea189dab35198ebb105640bedb97dd5269cc80863b16",
		},
		{
			pubkey: "02c1db96f2ba7e1cb4e9822d12de0f63fb666feb828c7f509e81fab9bd7a34039c",
			addr:   "58aceaf4b730b54bf00c0fb3f7b14886de470767f313c2d108968cd8bf0794b7",
		},
	}

	Convey("Test the public key and address", t, func() {
		for _, c := range testPubkeyAndAddr {
			pubByte, err := hex.DecodeString(c.pubkey)
			So(err, ShouldBeNil)
			pub, err := asymmetric.ParsePubKey(pubByte)
			So(err, ShouldBeNil)
			addr, err := PubKeyHash(pub)
			So(err, ShouldBeNil)
			So(addr.String(), ShouldEqual, c.addr)
		}
	})

	Convey("empty pubkey to address should fail", t, func() {
		pub := &asymmetric.PublicKey{}
		addr, err := PubKeyHash(pub)
		So(err, ShouldBeError)
		So(addr.String(), ShouldEqual, "0000000000000000000000000000000000000000000000000000000000000000")
	})
}
