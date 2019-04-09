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

package sqlchain

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestBlockCacheTTL(t *testing.T) {
	Convey("Test block cache TTL setting", t, func() {
		var cases = []struct {
			config *Config
			expect int32
		}{
			{
				config: &Config{
					BlockCacheTTL: -1,
					UpdatePeriod:  0,
				},
				expect: 0,
			},
			{
				config: &Config{
					BlockCacheTTL: 100,
					UpdatePeriod:  0,
				},
				expect: 100,
			},
			{
				config: &Config{
					BlockCacheTTL: 0,
					UpdatePeriod:  100,
				},
				expect: 0,
			},
		}
		for _, v := range cases {
			So(blockCacheTTLRequired(v.config), ShouldEqual, v.expect)
		}
	})
}
