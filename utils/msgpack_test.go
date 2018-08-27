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

	"github.com/CovenantSQL/CovenantSQL/utils/log"
	. "github.com/smartystreets/goconvey/convey"
)

type msgpackNestedStruct struct {
	C int64
}

type msgpackTestStruct struct {
	A string
	B msgpackNestedStruct
}

func TestMsgPack_EncodeDecode(t *testing.T) {
	Convey("primitive value encode decode test", t, func() {
		log.SetLevel(log.DebugLevel)
		i := uint64(1)
		buf, err := EncodeMsgPack(i)
		log.Debugf("uint64 1 encoded len %d to %x", len(buf.Bytes()), buf.Bytes())
		So(err, ShouldBeNil)
		var value uint64
		err = DecodeMsgPack(buf.Bytes(), &value)
		So(err, ShouldBeNil)
		So(value, ShouldEqual, i)
	})

	Convey("complex structure encode decode test", t, func() {
		preValue := &msgpackTestStruct{
			A: "happy",
			B: msgpackNestedStruct{
				C: 1,
			},
		}
		buf, err := EncodeMsgPack(preValue)
		So(err, ShouldBeNil)
		var postValue msgpackTestStruct
		err = DecodeMsgPack(buf.Bytes(), &postValue)
		So(err, ShouldBeNil)
		So(*preValue, ShouldResemble, postValue)
	})
}
