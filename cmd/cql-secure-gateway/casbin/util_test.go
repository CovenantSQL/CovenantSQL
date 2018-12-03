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

package casbin

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGlobMatch(t *testing.T) {
	Convey("glob match test", t, func() {
		So(globMatch("a*", "a"), ShouldBeTrue)
		So(globMatch("a*", "ab"), ShouldBeTrue)
		So(globMatch("a*", "abc"), ShouldBeTrue)
		So(globMatch("*a", "a"), ShouldBeTrue)
		So(globMatch("*a", "ca"), ShouldBeTrue)
		So(globMatch("*a", "bca"), ShouldBeTrue)
		So(globMatch("*a*", "a"), ShouldBeTrue)
		So(globMatch("*a*", "ab"), ShouldBeTrue)
		So(globMatch("*a*", "ca"), ShouldBeTrue)
		So(globMatch("*a*", "abc"), ShouldBeTrue)
		So(globMatch("*a*", "b"), ShouldBeFalse)
		So(globMatch("**a", "a"), ShouldBeTrue)
		So(globMatch("a*b", "ab"), ShouldBeTrue)
		So(globMatch("a*b", "acb"), ShouldBeTrue)
		So(globMatch("a*b", "accb"), ShouldBeTrue)
		So(globMatch("a*b", "abc"), ShouldBeFalse)
		So(globMatch("ab", "abc"), ShouldBeFalse)
		So(globMatch("abc", "bc"), ShouldBeFalse)
	})
	Convey("try both way glob", t, func() {
		So(tryGlobMatch("a*", "a"), ShouldBeTrue)
		So(tryGlobMatch("a", "a*"), ShouldBeTrue)
		So(tryGlobMatch("a", "a"), ShouldBeTrue)
		So(tryGlobMatch("a*", "a*"), ShouldBeTrue)
		So(tryGlobMatch("a*", "b*"), ShouldBeFalse)
		So(tryGlobMatch("*a", "a*"), ShouldBeFalse)
		So(tryGlobMatch("a*", "A"), ShouldBeTrue)
	})
}
