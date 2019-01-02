/*
 *  Copyright 2018 The CovenantSQL Authors.
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package worker

import (
	"sync/atomic"

	"github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/types"
)

var (
	testEventProfiles = []*types.SQLChainProfile{
		&types.SQLChainProfile{
			ID: proto.DatabaseID("111"),
		},
		&types.SQLChainProfile{
			ID: proto.DatabaseID("222"),
		},
		&types.SQLChainProfile{
			ID: proto.DatabaseID("333"),
		},
		&types.SQLChainProfile{
			ID: proto.DatabaseID("444"),
		},
		&types.SQLChainProfile{
			ID: proto.DatabaseID("555"),
		},
	}
	testOddProfiles = []*types.SQLChainProfile{
		&types.SQLChainProfile{
			ID: proto.DatabaseID("777"),
		},
		&types.SQLChainProfile{
			ID: proto.DatabaseID("888"),
		},
		&types.SQLChainProfile{
			ID: proto.DatabaseID("999"),
		},
	}
	testEventBlocks = types.BPBlock{
		SignedHeader: types.BPSignedHeader{
			BPHeader: types.BPHeader{
				Version: 1,
			},
		},
		Transactions: []interfaces.Transaction{
			&types.Transfer{},
			&types.Transfer{},
			&types.Transfer{},
		},
	}
	testOddBlocks = types.BPBlock{
		SignedHeader: types.BPSignedHeader{
			BPHeader: types.BPHeader{
				Version: 1,
			},
		},
		Transactions: []interfaces.Transaction{
			&types.Transfer{},
		},
	}
	testEventID = proto.DatabaseID("111")
	testOddID   = proto.DatabaseID("777")
)

type blockInfo struct {
	c, h    uint32
	block   *types.BPBlock
	profile []*types.SQLChainProfile
}

type stubBPService struct {
	blockMap map[uint32]*blockInfo
	count    uint32
}

func (s *stubBPService) FetchLastIrreversibleBlock(
	req *types.FetchLastIrreversibleBlockReq, resp *types.FetchLastIrreversibleBlockResp) (err error) {
	count := atomic.LoadUint32(&s.count)
	if bi, ok := s.blockMap[count%2]; ok {
		resp.Height = bi.h
		resp.Count = bi.c
		resp.Block = bi.block
		resp.SQLChains = bi.profile
	}
	atomic.AddUint32(&s.count, 1)
	return
}

func (s *stubBPService) FetchBlockByCount(req *types.FetchBlockByCountReq, resp *types.FetchBlockResp) (err error) {
	count := atomic.LoadUint32(&s.count)
	if req.Count > count {
		return ErrNotExists
	}
	if bi, ok := s.blockMap[req.Count%2]; ok {
		resp.Count = bi.c
		resp.Height = bi.h
		resp.Block = bi.block
	}
	return
}

func (s *stubBPService) Init() {
	s.blockMap = make(map[uint32]*blockInfo)
	s.blockMap[0] = &blockInfo{
		c:       0,
		h:       0,
		block:   &testEventBlocks,
		profile: testEventProfiles,
	}
	s.blockMap[1] = &blockInfo{
		c:       1,
		h:       1,
		block:   &testOddBlocks,
		profile: testOddProfiles,
	}
	atomic.StoreUint32(&s.count, 0)
}
