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

package client

import (
	"testing"

	"github.com/CovenantSQL/CovenantSQL/proto"
	. "github.com/smartystreets/goconvey/convey"
)

func TestConfig(t *testing.T) {
	Convey("test config", t, func() {
		var cfg *Config
		var err error

		cfg, err = ParseDSN("covenantsql://db")
		So(err, ShouldBeNil)
		So(cfg.DatabaseID, ShouldEqual, proto.DatabaseID("db"))
		So(cfg.FormatDSN(), ShouldEqual, "covenantsql://db")
	})
	Convey("test invalid config", t, func() {
		_, err := ParseDSN("invalid dsn")
		So(err, ShouldNotBeNil)
	})
	Convey("test dsn with only database id", t, func() {
		dbIDStr := "00000bef611d346c0cbe1beaa76e7f0ed705a194fdf9ac3a248ec70e9c198bf9"
		cfg, err := ParseDSN(dbIDStr)
		So(err, ShouldBeNil)
		So(cfg.DatabaseID, ShouldEqual, dbIDStr)
	})
}
