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
	"fmt"

	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/types"
)

// ChainRPCService defines a sql-chain RPC server.
type ChainRPCService struct {
	chain *Chain
}

// AdviseNewBlockReq defines a request of the AdviseNewBlock RPC method.
type AdviseNewBlockReq struct {
	Block *types.Block
	Count int32
}

// AdviseNewBlockResp defines a response of the AdviseNewBlock RPC method.
type AdviseNewBlockResp struct {
}

// FetchBlockReq defines a request of the FetchBlock RPC method.
type FetchBlockReq struct {
	Height int32
	node   proto.NodeID
}

// FetchBlockResp defines a response of the FetchBlock RPC method.
type FetchBlockResp struct {
	Height int32
	Block  *types.Block
}

// AdviseNewBlock is the RPC method to advise a new produced block to the target server.
func (s *ChainRPCService) AdviseNewBlock(req *AdviseNewBlockReq, resp *AdviseNewBlockResp) (
	err error) {
	s.chain.blocks <- req.Block
	return
}

// FetchBlock is the RPC method to fetch a known block from the target server.
func (s *ChainRPCService) FetchBlock(req *FetchBlockReq, resp *FetchBlockResp) (err error) {
	resp.Height = req.Height
	if _, found := s.chain.rt.getPeers().Find(req.node); !found {
		err = fmt.Errorf("node %s not in miner list", req.node)
		return
	}
	resp.Block, err = s.chain.FetchBlock(req.Height)
	if err == nil && resp.Block == nil {
		resp.Height = s.chain.getCurrentHeight()
	}
	return
}
