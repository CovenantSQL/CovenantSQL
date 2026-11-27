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

package types

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
)

func TestUpdateBillingVerify(t *testing.T) {
	Convey("UpdateBilling verifies only transactions with a signed range", t, func() {
		priv, _, err := asymmetric.GenSecp256k1KeyPair()
		So(err, ShouldBeNil)
		tx := NewUpdateBilling(&UpdateBillingHeader{
			Nonce:   1,
			Range:   Range{From: 1, To: 10},
			Version: 1,
		})

		Convey("Version 0 is rejected even with a valid signature", func() {
			tx.Version = 0
			So(tx.Sign(priv), ShouldBeNil)
			So(tx.DefaultHashSignVerifierImpl.Verify(&tx.UpdateBillingHeader), ShouldBeNil)
			So(tx.Verify(), ShouldEqual, ErrUnsupportedBillingVersion)
		})

		Convey("Version 1 still verifies", func() {
			So(tx.Sign(priv), ShouldBeNil)
			So(tx.Verify(), ShouldBeNil)
		})
	})
}
