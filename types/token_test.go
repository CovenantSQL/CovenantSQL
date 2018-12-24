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

	. "github.com/smartystreets/goconvey/convey"
)

func TestTokenType(t *testing.T) {
	Convey("test token util function", t, func() {
		eos := "EOS"
		unknown := "Unknown"
		token := FromString(eos)
		So(eos, ShouldEqual, token.String())
		So(token.Listed(), ShouldBeTrue)

		token = FromString("shitcoin")
		So(token.String(), ShouldEqual, unknown)
		So(token.Listed(), ShouldBeFalse)

		token = SupportTokenNumber
		So(token.String(), ShouldEqual, unknown)
		So(token.Listed(), ShouldBeFalse)
	})

	Convey("test token list", t, func() {
		So(SupportTokenNumber, ShouldEqual, len(TokenList))

		var i TokenType
		token := make(map[string]int)
		for i = 0; i < SupportTokenNumber; i++ {
			t, ok := TokenList[i]
			So(ok, ShouldBeTrue)
			token[t] = 1
		}
		So(len(token), ShouldEqual, SupportTokenNumber)
	})
}
