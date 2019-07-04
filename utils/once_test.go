/*
 * Copyright 2019 The CovenantSQL Authors.
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

	. "github.com/smartystreets/goconvey/convey"
)

func TestOnce(t *testing.T) {
	Convey("test once", t, func() {
		var a = 0
		o := Once{}
		o.Do(func() {
			a++
		})
		o.Do(func() {
			a++
		})
		So(a, ShouldEqual, 1)
		o.Reset()
		o.Do(func() {
			a++
		})
		So(a, ShouldEqual, 2)
	})
}
