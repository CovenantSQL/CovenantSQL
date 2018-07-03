/*
 * Copyright 2018 The ThunderDB Authors.
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
	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
	"gitlab.com/thunderdb/ThunderDB/worker/types"
)

// ChainRPCServer defines a sql-chain RPC server.
type ChainRPCServer struct {
	chain *Chain
}

// AdviseNewBlockReq defines a request of the AdviseNewBlock RPC method.
type AdviseNewBlockReq struct {
	Block *Block
}

// AdviseNewBlockResp defines a response of the AdviseNewBlock RPC method.
type AdviseNewBlockResp struct {
}

// AdviseBinLogReq defines a request of the AdviseBinLog RPC method.
type AdviseBinLogReq struct {
}

// AdviseBinLogResp defines a response of the AdviseBinLog RPC method.
type AdviseBinLogResp struct {
}

// AdviseResponsedQueryReq defines a request of the AdviseAckedQuery RPC method.
type AdviseResponsedQueryReq struct {
	Query *types.SignedResponseHeader
}

// AdviseResponsedQueryResp defines a response of the AdviseAckedQuery RPC method.
type AdviseResponsedQueryResp struct {
}

// AdviseAckedQueryReq defines a request of the AdviseAckedQuery RPC method.
type AdviseAckedQueryReq struct {
	Query *types.SignedAckHeader
}

// AdviseAckedQueryResp defines a response of the AdviseAckedQuery RPC method.
type AdviseAckedQueryResp struct {
}

// FetchBlockReq defines a request of the FetchBlock RPC method.
type FetchBlockReq struct {
	Height int32
}

// FetchBlockResp defines a response of the FetchBlock RPC method.
type FetchBlockResp struct {
	Height int32
	Block  *Block
}

// FetchAckedQueryReq defines a request of the FetchAckedQuery RPC method.
type FetchAckedQueryReq struct {
	height                   int32
	SignedResponseHeaderHash *hash.Hash
}

// FetchAckedQueryResp defines a request of the FetchAckedQuery RPC method.
type FetchAckedQueryResp struct {
	Ack *types.SignedAckHeader
}

// AdviseNewBlock is the RPC method to advise a new produced block to the target server.
func (s *ChainRPCServer) AdviseNewBlock(req *AdviseNewBlockReq, resp *AdviseNewBlockResp) error {
	return s.chain.CheckAndPushNewBlock(req.Block)
}

// AdviseBinLog is the RPC method to advise a new binary log to the target server.
func (s *ChainRPCServer) AdviseBinLog(req *AdviseBinLogReq, resp *AdviseBinLogResp) error {
	// TOOD(leventeliu): need implementation.
	return nil
}

// AdviseResponsedQuery is the RPC method to advise a new responsed query to the target server.
func (s *ChainRPCServer) AdviseResponsedQuery(
	req *AdviseResponsedQueryReq, resp *AdviseResponsedQueryResp) error {
	return s.chain.VerifyAndPushResponsedQuery(req.Query)
}

// AdviseAckedQuery is the RPC method to advise a new acknowledged query to the target server.
func (s *ChainRPCServer) AdviseAckedQuery(
	req *AdviseAckedQueryReq, resp *AdviseAckedQueryResp) error {
	return s.chain.VerifyAndPushAckedQuery(req.Query)
}

// FetchBlock is the RPC method to fetch a known block form the target server.
func (s *ChainRPCServer) FetchBlock(req *FetchBlockReq, resp *FetchBlockResp) (err error) {
	resp.Height = req.Height
	resp.Block, err = s.chain.FetchBlock(req.Height)
	return
}

// FetchAckedQuery is the RPC method to fetch a known block form the target server.
func (s *ChainRPCServer) FetchAckedQuery(req *FetchAckedQueryReq, resp *FetchAckedQueryResp,
) (err error) {
	resp.Ack, err = s.chain.FetchAckedQuery(req.height, req.SignedResponseHeaderHash)
	return
}
