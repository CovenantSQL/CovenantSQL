/*
 * Copyright 2018 The CovenantSQL Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package shard

import (
	"database/sql/driver"
	"testing"

	"github.com/CovenantSQL/sqlparser"
	. "github.com/smartystreets/goconvey/convey"
)

func TestGetShardTs(t *testing.T) {
	Convey("", t, func() {

		its, err := getShardTS(&sqlparser.SQLVal{
			Type: sqlparser.StrVal,
			Val:  []byte{'x'},
		},
			[]driver.NamedValue{
				driver.NamedValue{
					Name:    "",
					Ordinal: 0,
					Value:   nil,
				},
			})
		So(err, ShouldNotBeNil)
		So(its, ShouldEqual, -1)

		its, err = getShardTS(&sqlparser.SQLVal{
			Type: sqlparser.StrVal,
			Val:  nil,
		},
			[]driver.NamedValue{
				driver.NamedValue{
					Name:    "",
					Ordinal: 0,
					Value:   nil,
				},
			})
		So(err, ShouldNotBeNil)
		So(its, ShouldEqual, -1)

		its, err = getShardTS(&sqlparser.SQLVal{
			Type: sqlparser.IntVal,
			Val:  nil,
		},
			[]driver.NamedValue{
				driver.NamedValue{
					Name:    "",
					Ordinal: 0,
					Value:   nil,
				},
			})
		So(err, ShouldNotBeNil)
		So(its, ShouldEqual, -1)

		its, err = getShardTS(&sqlparser.SQLVal{
			Type: sqlparser.ValArg,
			Val:  nil,
		},
			[]driver.NamedValue{
				driver.NamedValue{
					Name:    "",
					Ordinal: 0,
					Value:   nil,
				},
			})
		So(err, ShouldNotBeNil)
		So(its, ShouldEqual, -1)


		its, err = getShardTS(&sqlparser.SQLVal{
			Type: sqlparser.ValArg,
			Val:  []byte{':', 'v', '1'},
		},
			[]driver.NamedValue{
				driver.NamedValue{
					Name:    "",
					Ordinal: 1,
					Value:   int64(11111111111),
				},
			})
		So(err, ShouldBeNil)
		So(its, ShouldEqual, 11111111111)


		its, err = getShardTS(&sqlparser.SQLVal{
			Type: sqlparser.ValArg,
			Val:  []byte{':', 'v', '1'},
		},
			[]driver.NamedValue{
				{
					Name:    "",
					Ordinal: 1,
					Value:   "2019-01-02T15:04:05.999999999",
				},
			})
		So(err, ShouldBeNil)
		So(its, ShouldEqual, 1546441445)

		its, err = getShardTS(&sqlparser.SQLVal{
			Type: sqlparser.ValArg,
			Val:  []byte{':', 'v', '1'},
		},
			[]driver.NamedValue{
				{
					Name:    "v1",
					Ordinal: 1,
					Value:   "2019-01-02T15:04:05.999999999",
				},
			})
		So(err, ShouldBeNil)
		So(its, ShouldEqual, 1546441445)

		its, err = getShardTS(&sqlparser.SQLVal{
			Type: sqlparser.ValArg,
			Val:  []byte{':', 'v', '1'},
		},
			[]driver.NamedValue{
				{
					Name:    "v1",
					Ordinal: 1,
					Value:   "xxxx",
				},
			})
		So(err, ShouldNotBeNil)
		So(its, ShouldEqual, -1)

		its, err = getShardTS(&sqlparser.SQLVal{
			Type: sqlparser.ValArg,
			Val:  []byte{':', 'v', '1'},
		},
			[]driver.NamedValue{
				{
					Name:    "v1",
					Ordinal: 1,
					Value:   true,
				},
			})
		So(err, ShouldNotBeNil)
		So(its, ShouldEqual, -1)

		its, err = getShardTS(&sqlparser.SQLVal{
			Type: sqlparser.FloatVal,
			Val:  nil,
		},
			nil)
		So(err, ShouldNotBeNil)
		So(its, ShouldEqual, -1)

		its, err = getShardTS(&sqlparser.SQLVal{
			Type: sqlparser.ValArg,
			Val:  []byte{':', 'v', '1'},
		},
			[]driver.NamedValue{
				{
					Name:    "",
					Ordinal: 1,
					Value:   float64(1111111111.1),
				},
			})
		So(err, ShouldBeNil)
		So(its, ShouldEqual, 1111111111)

		t, _ := ParseTime("2019-01-02T15:04:05.999999999")
		its, err = getShardTS(&sqlparser.SQLVal{
			Type: sqlparser.ValArg,
			Val:  []byte{':', 'v', '1'},
		},
			[]driver.NamedValue{
				{
					Name:    "",
					Ordinal: 1,
					Value:   t,
				},
			})
		So(err, ShouldBeNil)
		So(its, ShouldEqual, 1546441445)

		its, err = getShardTS(&sqlparser.SQLVal{
			Type: sqlparser.ValArg,
			Val:  nil,
		},
			nil)
		So(err, ShouldNotBeNil)
		So(its, ShouldEqual, -1)
	})
}