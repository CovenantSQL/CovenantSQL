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
	. "github.com/smartystreets/goconvey/convey"
)

func TestTypes(t *testing.T) {
	Convey("test nils", t, func() {
		priv, _, err := asymmetric.GenSecp256k1KeyPair()
		So(err, ShouldBeNil)

		h1 := &SignedCreateDatabaseRequestHeader{}
		err = h1.Sign(priv)
		So(err, ShouldBeNil)
		h1.Signee = nil
		err = h1.Verify()
		So(err, ShouldNotBeNil)

		h2 := &SignedCreateDatabaseResponseHeader{}
		err = h2.Sign(priv)
		So(err, ShouldBeNil)
		h2.Signee = nil
		err = h2.Verify()
		So(err, ShouldNotBeNil)

		h3 := &SignedDropDatabaseRequestHeader{}
		err = h3.Sign(priv)
		So(err, ShouldBeNil)
		h3.Signee = nil
		err = h3.Verify()
		So(err, ShouldNotBeNil)

		h4 := &SignedGetDatabaseRequestHeader{}
		err = h4.Sign(priv)
		So(err, ShouldBeNil)
		h4.Signee = nil
		err = h4.Verify()
		So(err, ShouldNotBeNil)

		h5 := &SignedGetDatabaseResponseHeader{}
		err = h5.Sign(priv)
		So(err, ShouldBeNil)
		h5.Signee = nil
		err = h5.Verify()
		So(err, ShouldNotBeNil)
	})
	Convey("test nested sign/verify", t, func() {
		priv, _, err := asymmetric.GenSecp256k1KeyPair()
		So(err, ShouldBeNil)

		r1 := &CreateDatabaseRequest{}
		err = r1.Sign(priv)
		So(err, ShouldBeNil)
		err = r1.Verify()
		So(err, ShouldBeNil)

		r2 := &CreateDatabaseResponse{}
		err = r2.Sign(priv)
		So(err, ShouldBeNil)
		err = r2.Verify()
		So(err, ShouldBeNil)

		r3 := &DropDatabaseRequest{}
		err = r3.Sign(priv)
		So(err, ShouldBeNil)
		err = r3.Verify()
		So(err, ShouldBeNil)

		r4 := &GetDatabaseRequest{}
		err = r4.Sign(priv)
		So(err, ShouldBeNil)
		err = r4.Verify()
		So(err, ShouldBeNil)

		r5 := &GetDatabaseResponse{}
		err = r5.Sign(priv)
		So(err, ShouldBeNil)
		err = r5.Verify()
		So(err, ShouldBeNil)
	})
}
