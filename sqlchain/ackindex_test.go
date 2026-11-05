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
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/types"
)

func TestAckIndex(t *testing.T) {
	Convey("Given a ackIndex instance", t, func() {
		var (
			err error

			ai   = newAckIndex()
			resp = &types.SignedResponseHeader{
				ResponseHeader: types.ResponseHeader{
					Request: types.RequestHeader{
						NodeID: proto.NodeID(
							"0000000000000000000000000000000000000000000000000000000000000000"),
						ConnectionID: 0,
						SeqNo:        0,
					},
				},
			}
			ack = &types.SignedAckHeader{
				AckHeader: types.AckHeader{
					Response: resp.ResponseHeader,
				},
			}
		)
		Convey("Add response and register ack should return no error", func() {
			err = ai.addResponse(0, resp)
			So(err, ShouldBeNil)
			err = ai.register(0, ack)
			So(err, ShouldBeNil)
			err = ai.remove(0, ack)
			So(err, ShouldBeNil)
		})
	})
}

func testResponse(seqNo uint64) *types.SignedResponseHeader {
	return &types.SignedResponseHeader{
		ResponseHeader: types.ResponseHeader{
			Request: types.RequestHeader{
				NodeID: proto.NodeID(
					"0000000000000000000000000000000000000000000000000000000000000000"),
				SeqNo: seqNo,
			},
		},
	}
}

// TestAckIndexAddAfterAdvance adds a response to an index that advance has expired, as a caller
// does that loaded the index before advance removed it. advance used to read the index after
// releasing its lock, which the race detector reports.
func TestAckIndexAddAfterAdvance(t *testing.T) {
	ai := newAckIndex()
	mi, err := ai.load(0)
	if err != nil {
		t.Fatal(err)
	}
	advanced := make(chan struct{})
	go func() {
		ai.advance(1)
		close(advanced)
	}()
	// Wait for advance without synchronizing with it, so that nothing but the lock of mi orders
	// what advance reads before the write below.
	time.Sleep(100 * time.Millisecond)
	if err = mi.addResponse(testResponse(0)); err != nil {
		t.Fatal(err)
	}
	<-advanced
}

// TestAckIndexStats checks that stats counts what the indexes of the heights in the ackIndex
// hold, and not a response added late to an index that advance has expired.
func TestAckIndexStats(t *testing.T) {
	ai := newAckIndex()
	check := func(heights, responses, acks int) {
		t.Helper()
		if h, r, a := ai.stats(); h != heights || r != responses || a != acks {
			t.Errorf("stats() = %d, %d, %d, want %d, %d, %d", h, r, a, heights, responses, acks)
		}
	}
	check(0, 0, 0)
	acked := testResponse(1)
	for _, err := range []error{
		ai.addResponse(0, testResponse(0)),
		ai.addResponse(1, acked),
		ai.addResponse(1, testResponse(2)),
		ai.register(1, &types.SignedAckHeader{AckHeader: types.AckHeader{Response: acked.ResponseHeader}}),
	} {
		if err != nil {
			t.Fatal(err)
		}
	}
	check(2, 2, 1)
	expired, err := ai.load(0)
	if err != nil {
		t.Fatal(err)
	}
	ai.advance(1)
	check(1, 1, 1)
	if err = expired.addResponse(testResponse(3)); err != nil {
		t.Fatal(err)
	}
	check(1, 1, 1)
}
