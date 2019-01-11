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

package timer

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestTimer(t *testing.T) {
	Convey("test timer", t, func() {
		t := NewTimer()
		time.Sleep(time.Millisecond * 100)
		t.Add("stage1")

		time.Sleep(time.Second * 1)
		t.Add("stage2")

		m := t.ToMap()
		So(m, ShouldHaveLength, 3)
		So(m, ShouldContainKey, "stage1")
		So(m, ShouldContainKey, "stage2")
		So(m["stage1"], ShouldBeGreaterThanOrEqualTo, time.Millisecond*100)
		So(m["stage2"], ShouldBeGreaterThanOrEqualTo, time.Second)
		So(m["total"], ShouldBeGreaterThanOrEqualTo, time.Second+time.Millisecond*100)

		f := t.ToLogFields()
		So(f, ShouldHaveLength, 3)
		So(f, ShouldContainKey, "stage1")
		So(f, ShouldContainKey, "stage2")
		So(m["stage1"], ShouldEqual, f["stage1"])
		So(m["stage2"], ShouldEqual, f["stage2"])
		So(m["total"], ShouldEqual, f["total"])
	})
}
