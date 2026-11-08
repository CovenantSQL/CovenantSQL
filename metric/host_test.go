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

package metric

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestReadHost(t *testing.T) {
	Convey("read host resources", t, func() {
		h := ReadHost(t.TempDir())
		So(h.MemAvailable, ShouldBeGreaterThan, 0)
		So(h.DiskAvailable, ShouldBeGreaterThan, 0)
		So(h.Load15, ShouldBeGreaterThanOrEqualTo, 0)
	})
}
