/*
 * Copyright 2018 The ThunderDB Authors.
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

package client

import (
	"testing"
	"time"

	"github.com/CovenantSQL/CovenantSQL/proto"
	. "github.com/smartystreets/goconvey/convey"
)

func TestConfig(t *testing.T) {
	Convey("test config", t, func() {
		var cfg *Config
		var err error

		cfg, err = ParseDSN("thunderdb://db")
		So(err, ShouldBeNil)
		So(cfg.DatabaseID, ShouldEqual, proto.DatabaseID("db"))
		So(cfg.FormatDSN(), ShouldEqual, "thunderdb://db")

		// test with parameters
		cfg, err = ParseDSN("thunderdb://db?debug=true&update_interval=1s")
		So(err, ShouldBeNil)
		So(cfg.DatabaseID, ShouldEqual, proto.DatabaseID("db"))
		So(cfg.Debug, ShouldBeTrue)
		So(cfg.PeersUpdateInterval, ShouldEqual, time.Second)

		cfg.Debug = false
		So(cfg.FormatDSN(), ShouldEqual, "thunderdb://db?update_interval=1s")

		cfg.Debug = true
		cfg.PeersUpdateInterval = DefaultPeersUpdateInterval
		So(cfg.FormatDSN(), ShouldEqual, "thunderdb://db?debug=true")
	})
}
