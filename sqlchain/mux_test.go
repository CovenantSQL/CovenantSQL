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

	. "github.com/smartystreets/goconvey/convey"

	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/types"
)

func TestMuxAdviseNewBlock(t *testing.T) {
	Convey("A nil block should not reach the block processing of the sql-chain", t, func() {
		var (
			blocks = make(chan *types.Block, 1)
			mux    = &MuxService{}
		)
		mux.register(testDatabaseID, &ChainRPCService{chain: &Chain{blocks: blocks}})
		err := mux.AdviseNewBlock(
			&MuxAdviseNewBlockReq{DatabaseID: testDatabaseID}, &MuxAdviseNewBlockResp{})
		So(err, ShouldEqual, ErrNilBlock)
		So(blocks, ShouldBeEmpty)
	})
}

func TestMuxFetchBlock(t *testing.T) {
	Convey("Given a sql-chain with a single peer", t, func() {
		var (
			peer     = proto.NodeID(hash.THashH([]byte("peer")).String())
			stranger = proto.NodeID(hash.THashH([]byte("stranger")).String())
			mux      = &MuxService{}
		)
		mux.register(testDatabaseID, &ChainRPCService{chain: &Chain{rt: &runtime{
			peers: &proto.Peers{PeersHeader: proto.PeersHeader{Servers: []proto.NodeID{peer}}},
			head:  &state{node: &blockNode{}},
		}}})
		fetch := func(caller *proto.RawNodeID) error {
			return mux.FetchBlock(&MuxFetchBlockReq{
				Envelope:      proto.Envelope{NodeID: caller},
				DatabaseID:    testDatabaseID,
				FetchBlockReq: FetchBlockReq{Height: 1},
			}, &MuxFetchBlockResp{})
		}
		Convey("The peer should be able to fetch blocks", func() {
			So(fetch(peer.ToRawNodeID()), ShouldBeNil)
		})
		Convey("Other nodes should be rejected", func() {
			So(fetch(stranger.ToRawNodeID()), ShouldEqual, ErrNotPeer)
			So(fetch(nil), ShouldEqual, ErrNotPeer)
		})
	})
}
