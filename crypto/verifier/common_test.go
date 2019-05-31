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

package verifier

import (
	"math/big"
	"testing"

	"github.com/pkg/errors"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
)

var (
	MockHash = []byte{
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
	}
)

type MockHeader struct{}

func (*MockHeader) MarshalHash() ([]byte, error) {
	return MockHash, nil
}

type MockObject struct {
	MockHeader
	HSV DefaultHashSignVerifierImpl
}

func (o *MockObject) Sign(signer *asymmetric.PrivateKey) error {
	return o.HSV.Sign(&o.MockHeader, signer)
}

func (o *MockObject) Verify() error {
	return o.HSV.Verify(&o.MockHeader)
}

func TestDefaultHashSignVerifierImpl(t *testing.T) {
	Convey("Given a dummy object and a pair of keys", t, func() {
		var (
			obj          = &MockObject{}
			priv, _, err = asymmetric.GenSecp256k1KeyPair()
		)
		So(err, ShouldBeNil)
		So(priv, ShouldNotBeNil)
		Convey("When the object is signed by the key pair", func() {
			err = obj.Sign(priv)
			So(err, ShouldBeNil)
			Convey("The object should be verifiable", func() {
				err = obj.Verify()
				So(err, ShouldBeNil)
			})
			Convey("The object should have data hash", func() {
				So(obj.HSV.Hash(), ShouldEqual, hash.THashH(MockHash))
			})
			Convey("When the hash is modified", func() {
				obj.HSV.DataHash = hash.Hash{0x0, 0x0, 0x0, 0x1}
				Convey("The verifier should return hash value not match error", func() {
					err = errors.Cause(obj.Verify())
					So(err, ShouldEqual, ErrHashValueNotMatch)
				})
			})
			Convey("When the signee is not set", func() {
				obj.HSV.Signee = nil
				Convey("The verifier should return signature not match error", func() {
					err = errors.Cause(obj.Verify())
					So(err, ShouldEqual, ErrSignatureNotMatch)
				})
			})
			Convey("When the signature is not set", func() {
				obj.HSV.Signature = nil
				Convey("The verifier should return signature not match error", func() {
					err = errors.Cause(obj.Verify())
					So(err, ShouldEqual, ErrSignatureNotMatch)
				})
			})
			Convey("When the signee is modified", func() {
				var _, pub, err = asymmetric.GenSecp256k1KeyPair()
				So(err, ShouldBeNil)
				obj.HSV.Signee = pub
				Convey("The verifier should return signature not match error", func() {
					err = errors.Cause(obj.Verify())
					So(err, ShouldEqual, ErrSignatureNotMatch)
				})
			})
			Convey("When the signature is modified", func() {
				var val = obj.HSV.Signature.R
				val.Add(val, big.NewInt(1))
				Convey("The verifier should return signature not match error", func() {
					err = errors.Cause(obj.Verify())
					So(err, ShouldEqual, ErrSignatureNotMatch)
				})
			})
		})
	})
}
