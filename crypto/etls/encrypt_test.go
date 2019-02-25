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

package etls

import (
	"bytes"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
)

func TestKeyDerivation(t *testing.T) {
	hSuite := &hash.HashSuite{
		HashLen:  hash.HashBSize,
		HashFunc: hash.DoubleHashB,
	}

	Convey("get addr", t, func() {
		rawKey := bytes.Repeat([]byte("a"), 1000)
		dKey := KeyDerivation(rawKey, 100, hSuite)
		So(dKey, ShouldHaveLength, 100)
	})
}
