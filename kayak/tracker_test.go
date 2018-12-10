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

package kayak

import (
	"context"
	"testing"
	"time"

	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/pkg/errors"
	. "github.com/smartystreets/goconvey/convey"
)

type fakeTrackerCaller struct {
	c C
}

func (c *fakeTrackerCaller) Call(method string, req interface{}, resp interface{}) (err error) {
	time.Sleep(time.Millisecond * 500)
	c.c.So(method, ShouldEqual, "test")
	if req != 1 {
		err = errors.New("invalid result")
	}
	return
}

func TestTracker(t *testing.T) {
	Convey("test tracker", t, func(c C) {
		nodeID1 := proto.NodeID("000005f4f22c06f76c43c4f48d5a7ec1309cc94030cbf9ebae814172884ac8b5")
		nodeID2 := proto.NodeID("000005aa62048f85da4ae9698ed59c14ec0d48a88a07c15a32265634e7e64ade")
		r := &Runtime{
			applyRPCMethod: "test",
			followers: []proto.NodeID{
				nodeID1,
				nodeID2,
			},
		}
		r.SetCaller(nodeID1, &fakeTrackerCaller{c: c})
		r.SetCaller(nodeID2, &fakeTrackerCaller{c: c})
		t1 := newApplyTracker(r, 1, 0)
		t1.send()
		_, meets, _ := t1.get(context.Background())
		So(meets, ShouldBeTrue)

		t2 := newApplyTracker(r, 1, 1)
		t2.send()
		r2, meets, _ := t2.get(context.Background())
		So(r2, ShouldNotBeEmpty)
		So(meets, ShouldBeTrue)

		t3 := newApplyTracker(r, 1, 1)
		t3.send()
		ctx1, cancelCtx1 := context.WithTimeout(context.Background(), time.Millisecond*1)
		defer cancelCtx1()
		r3, meets, finished := t3.get(ctx1)
		So(r3, ShouldBeEmpty)
		So(meets, ShouldBeFalse)
		So(finished, ShouldBeFalse)

		r3, meets, finished = t3.get(context.Background())
		So(r3, ShouldNotBeEmpty)
		So(meets, ShouldBeTrue)

		t4 := newApplyTracker(r, 1, 2)
		t4.send()
		r4, meets, finished := t4.get(context.Background())
		So(r4, ShouldHaveLength, 2)
		So(meets, ShouldBeTrue)
		So(finished, ShouldBeTrue)

		t5 := newApplyTracker(r, 2, 2)
		t5.send()
		ctx2, cancelCtx2 := context.WithTimeout(context.Background(), time.Millisecond*1)
		defer cancelCtx2()
		r5, meets, finished := t5.get(ctx2)
		So(r5, ShouldBeEmpty)
		So(meets, ShouldBeFalse)
		So(finished, ShouldBeFalse)

		r5, meets, finished = t5.get(context.Background())
		So(r5, ShouldHaveLength, 2)
		So(meets, ShouldBeTrue)
		So(finished, ShouldBeTrue)

		t5.close()
		So(t5.closed, ShouldEqual, 1)
	})
}
