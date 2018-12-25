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

	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/mohae/deepcopy"
	. "github.com/smartystreets/goconvey/convey"
)

func TestBaseAccount(t *testing.T) {
	Convey("base account", t, func() {
		h, err := hash.NewHashFromStr("000005aa62048f85da4ae9698ed59c14ec0d48a88a07c15a32265634e7e64ade")
		So(err, ShouldBeNil)
		addr := proto.AccountAddress(*h)
		ba := NewBaseAccount(&Account{
			Address: addr,
		})
		So(ba.GetAccountAddress(), ShouldEqual, addr)
		So(ba.GetAccountNonce(), ShouldEqual, 0)
		So(ba.Hash(), ShouldEqual, hash.Hash{})
		priv, _, err := asymmetric.GenSecp256k1KeyPair()
		So(err, ShouldBeNil)
		So(ba.Sign(priv), ShouldBeNil)
		So(ba.Verify(), ShouldBeNil)
	})
}

func TestDeepcopier(t *testing.T) {
	Convey("base account", t, func() {
		var p1 = &SQLChainProfile{
			Miners: []*MinerInfo{
				&MinerInfo{},
				&MinerInfo{},
				&MinerInfo{},
			},
		}
		var p2 = deepcopy.Copy(p1).(*SQLChainProfile)
		t.Logf("%p %p", p1.Miners[0], p2.Miners[0])
		So(p1.Miners[0], ShouldNotEqual, p2.Miners[0])
	})
}
