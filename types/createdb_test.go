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

package types

import (
	"testing"

	"github.com/CovenantSQL/CovenantSQL/crypto"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/proto"
	. "github.com/smartystreets/goconvey/convey"
)

func TestTxCreateDatabase(t *testing.T) {
	Convey("test tx create database", t, func() {
		h, err := hash.NewHashFromStr("000005aa62048f85da4ae9698ed59c14ec0d48a88a07c15a32265634e7e64ade")
		So(err, ShouldBeNil)

		cd := NewCreateDatabase(&CreateDatabaseHeader{
			Owner: proto.AccountAddress(*h),
			Nonce: 1,
		})

		So(cd.GetAccountNonce(), ShouldEqual, 1)

		priv, _, err := asymmetric.GenSecp256k1KeyPair()
		So(err, ShouldBeNil)

		err = cd.Sign(priv)
		So(err, ShouldBeNil)

		err = cd.Verify()
		So(err, ShouldBeNil)

		addr, err := crypto.PubKeyHash(priv.PubKey())
		So(err, ShouldBeNil)
		So(cd.GetAccountAddress(), ShouldEqual, addr)
	})
}
