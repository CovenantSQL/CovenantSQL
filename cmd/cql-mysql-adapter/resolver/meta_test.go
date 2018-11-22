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

package resolver

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestMetaHandler(t *testing.T) {
	Convey("test handler", t, func() {
		mh := NewMetaHandler()
		testDBID := randomDBID()
		anotherDBID := randomDBID()
		db, err := fakeConn([]string{
			"CREATE TABLE test1 (test2 string)",
		})
		defer db.Close()
		So(err, ShouldBeNil)
		mh.AddConn(testDBID, db)
		result, err := mh.GetTables(testDBID)
		So(err, ShouldBeNil)
		So(result, ShouldResemble, []string{"test1"})
		result, err = mh.GetTables(testDBID)
		So(err, ShouldBeNil)
		So(result, ShouldResemble, []string{"test1"})
		result, err = mh.GetTable(testDBID, "test1")
		So(err, ShouldBeNil)
		So(result, ShouldResemble, []string{"test2"})
		result, err = mh.GetTable(testDBID, "test1")
		So(err, ShouldBeNil)
		So(result, ShouldResemble, []string{"test2"})
		_, err = mh.GetTable(testDBID, "test2")
		So(err, ShouldNotBeNil)
		mh.ReloadMeta()
		result, err = mh.GetTable(testDBID, "test1")
		So(err, ShouldBeNil)
		So(result, ShouldResemble, []string{"test2"})
		result, err = mh.GetTable(testDBID, "test1")
		So(err, ShouldBeNil)
		So(result, ShouldResemble, []string{"test2"})
		_, err = mh.GetTable(testDBID, "test2")
		So(err, ShouldNotBeNil)
		c, exists := mh.GetConn(testDBID)
		So(exists, ShouldBeTrue)
		So(c, ShouldNotBeNil)
		_, exists = mh.GetConn(anotherDBID)
		So(exists, ShouldBeFalse)
		_, err = mh.GetTables(anotherDBID)
		So(err, ShouldNotBeNil)
		_, err = mh.GetTable(anotherDBID, "test1")
		So(err, ShouldNotBeNil)
	})
}
