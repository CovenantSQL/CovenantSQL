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

package wal

import (
	"os"
	"testing"

	kt "github.com/CovenantSQL/CovenantSQL/kayak/types"
	"github.com/CovenantSQL/CovenantSQL/proto"
	. "github.com/smartystreets/goconvey/convey"
)

func TestLevelDBWal_Write(t *testing.T) {
	Convey("test leveldb wal write", t, func() {
		var p *LevelDBWal
		var err error
		p, err = NewLevelDBWal("testWrite.ldb")
		So(err, ShouldBeNil)
		defer func() {
			p.Close()
			os.RemoveAll("testWrite.ldb")
		}()

		l1 := &kt.Log{
			LogHeader: kt.LogHeader{
				Index:    0,
				Type:     kt.LogPrepare,
				Producer: proto.NodeID("0000000000000000000000000000000000000000000000000000000000000000"),
			},
			Data: []byte("happy1"),
		}

		err = p.Write(l1)
		So(err, ShouldBeNil)
		err = p.Write(l1)
		So(err, ShouldNotBeNil)

		// test get
		var l *kt.Log
		l, err = p.Get(l1.Index)
		So(err, ShouldBeNil)
		So(l, ShouldResemble, l1)

		// test consecutive writes
		l2 := &kt.Log{
			LogHeader: kt.LogHeader{
				Index: 1,
				Type:  kt.LogPrepare,
			},
			Data: []byte("happy2"),
		}
		err = p.Write(l2)
		So(err, ShouldBeNil)

		// test not consecutive writes
		l4 := &kt.Log{
			LogHeader: kt.LogHeader{
				Index: 3,
				Type:  kt.LogPrepare,
			},
			Data: []byte("happy3"),
		}
		err = p.Write(l4)
		So(err, ShouldBeNil)

		l3 := &kt.Log{
			LogHeader: kt.LogHeader{
				Index: 2,
				Type:  kt.LogPrepare,
			},
			Data: []byte("happy4"),
		}
		err = p.Write(l3)
		So(err, ShouldBeNil)
	})
}
