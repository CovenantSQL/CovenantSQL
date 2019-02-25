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

package utils

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

func TestNewLevelDBKey(t *testing.T) {
	Convey("new bytes", t, func() {
		log.SetLevel(log.DebugLevel)
		So(ConcatAll(nil), ShouldResemble, []byte{})
		So(ConcatAll([]byte{}), ShouldResemble, []byte{})
		So(ConcatAll([]byte{'0'}, []byte{'1'}), ShouldResemble, []byte{'0', '1'})
		So(ConcatAll([]byte{'0'}, nil), ShouldResemble, []byte{'0'})
		So(ConcatAll(nil, []byte{'0'}), ShouldResemble, []byte{'0'})
		So(ConcatAll([]byte{'0', '1', '2', '3'}, []byte{'a', 'b', 'c', 'd', 'e'}, []byte{'x', 'y', 'z'}),
			ShouldResemble, []byte{'0', '1', '2', '3', 'a', 'b', 'c', 'd', 'e', 'x', 'y', 'z'})
		So(ConcatAll([]byte{'0', '1', '2', '3'}, nil, []byte{'x', 'y', 'z'}),
			ShouldResemble, []byte{'0', '1', '2', '3', 'x', 'y', 'z'})
		So(ConcatAll([]byte{'0', '1', '2', '3'}, []byte{}, []byte{'x', 'y', 'z'}),
			ShouldResemble, []byte{'0', '1', '2', '3', 'x', 'y', 'z'})
		So(ConcatAll(nil, []byte{'0', '1', '2', '3'}, nil, []byte{'x', 'y', 'z'}),
			ShouldResemble, []byte{'0', '1', '2', '3', 'x', 'y', 'z'})
		So(ConcatAll([]byte{}, []byte{'0', '1', '2', '3'}, nil, []byte{'x', 'y', 'z'}, nil),
			ShouldResemble, []byte{'0', '1', '2', '3', 'x', 'y', 'z'})
	})
}
