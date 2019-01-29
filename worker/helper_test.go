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
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/types"
)

var (
	testEventProfiles = []*types.SQLChainProfile{
		&types.SQLChainProfile{
			ID: proto.DatabaseID("111"),
			Users: []*types.SQLChainUser{
				testUser1,
			},
		},
		&types.SQLChainProfile{
			ID: proto.DatabaseID("222"),
			Users: []*types.SQLChainUser{
				testUser2,
			},
		},
		&types.SQLChainProfile{
			ID: proto.DatabaseID("333"),
			Users: []*types.SQLChainUser{
				testUser3,
			},
		},
		&types.SQLChainProfile{
			ID: proto.DatabaseID("444"),
			Users: []*types.SQLChainUser{
				testUser4,
			},
		},
	}
	testOddProfiles = []*types.SQLChainProfile{
		&types.SQLChainProfile{
			ID: proto.DatabaseID("111"),
			Users: []*types.SQLChainUser{
				testUser4,
			},
		},
		&types.SQLChainProfile{
			ID: proto.DatabaseID("222"),
			Users: []*types.SQLChainUser{
				testUser3,
			},
		},
		&types.SQLChainProfile{
			ID: proto.DatabaseID("333"),
			Users: []*types.SQLChainUser{
				testUser2,
			},
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
	testID           = proto.DatabaseID("111")
	testNotExistID   = proto.DatabaseID("not exist")
	testAddr         = proto.AccountAddress(hash.THashH([]byte{'a', 'd', 'd', 'r', '1'}))
	testNotExistAddr = proto.AccountAddress(hash.THashH([]byte{'a', 'a'}))
	testUser1        = &types.SQLChainUser{
		Address:    testAddr,
		Permission: types.UserPermissionFromRole(types.Write),
		Status:     types.Normal,
	}
	testUser2 = &types.SQLChainUser{
		Address:    testAddr,
		Permission: types.UserPermissionFromRole(types.Read),
		Status:     types.Arrears,
	}
	testUser3 = &types.SQLChainUser{
		Address:    testAddr,
		Permission: types.UserPermissionFromRole(types.Write),
		Status:     types.Reminder,
	}
	testUser4 = &types.SQLChainUser{
		Address:    testAddr,
		Permission: types.UserPermissionFromRole(types.Read),
		Status:     types.Arbitration,
	}
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
