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

package utils

import (
	"testing"

	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/btcsuite/btcutil/base58"
	. "github.com/smartystreets/goconvey/convey"
)

func TestPubKeyHashAndAddressing(t *testing.T) {
	Convey("Randomly generate some key pairs and calculate public key hash values", t, func() {
		for i := 0; i < 20; i++ {
			_, pub, err := asymmetric.GenSecp256k1KeyPair()
			So(err, ShouldBeNil)
			h, err := PubKeyHash(pub)
			So(err, ShouldBeNil)
			addr, err := PubKey2Addr(pub, MainNet)
			So(err, ShouldBeNil)
			targetAddr := base58.CheckEncode(h[:], MainNet)
			So(addr, ShouldEqual, targetAddr)
			t.Logf("main net address: %s", targetAddr)

			addr, err = PubKey2Addr(pub, TestNet)
			So(err, ShouldBeNil)
			targetAddr = base58.CheckEncode(h[:], TestNet)
			So(err, ShouldBeNil)
			t.Logf("test net address: %s", targetAddr)
		}
	})
}
