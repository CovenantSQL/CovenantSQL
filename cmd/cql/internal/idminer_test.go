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

package internal

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	mine "github.com/CovenantSQL/CovenantSQL/pow/cpuminer"
)

func TestNonceGen(t *testing.T) {
	Convey("nonceGen returns after a nonce is found", t, func() {
		_, publicKey, err := asymmetric.GenSecp256k1KeyPair()
		So(err, ShouldBeNil)
		difficulty = 16
		for i := 0; i < 50; i++ {
			nonceCh := make(chan *mine.NonceInfo, 1)
			go func() { nonceCh <- nonceGen(publicKey) }()
			var nonce *mine.NonceInfo
			select {
			case nonce = <-nonceCh:
			case <-time.After(10 * time.Second):
			}
			So(nonce, ShouldNotBeNil)
			So(nonce.Difficulty, ShouldBeGreaterThanOrEqualTo, difficulty)
		}
	})
}
