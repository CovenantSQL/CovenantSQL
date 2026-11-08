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
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestHostResources(t *testing.T) {
	Convey("read host resources", t, func() {
		mem, err := MemAvailable()
		So(err, ShouldBeNil)
		So(mem, ShouldBeGreaterThan, 0)

		load, err := LoadAvg()
		So(err, ShouldBeNil)
		So(load[2], ShouldBeGreaterThanOrEqualTo, 0)

		disk, err := DiskAvailable(t.TempDir())
		So(err, ShouldBeNil)
		So(disk, ShouldBeGreaterThan, 0)

		_, err = DiskAvailable(filepath.Join(t.TempDir(), "missing"))
		So(err, ShouldNotBeNil)
	})
}
