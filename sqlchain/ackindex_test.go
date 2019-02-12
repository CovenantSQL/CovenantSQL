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

	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/types"
	. "github.com/smartystreets/goconvey/convey"
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
