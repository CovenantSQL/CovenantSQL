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
	"gitlab.com/thunderdb/ThunderDB/worker"
)

type ChainRPCServer struct {
	chain *Chain
}

type AdviseNewBlockReq struct {
	Block *Block
}

type AdviseNewBlockResp struct {
}

type AdviseBinLogReq struct {
}

type AdviseBinLogResp struct {
}

type AdviseResponsedQueryReq struct {
	Query *worker.SignedResponseHeader
}

type AdviseResponsedQueryResp struct {
}

type AdviseAckedQueryReq struct {
	Query *worker.SignedAckHeader
}

type AdviseAckedQueryResp struct {
}

type FetchBlockReq struct {
	Height int32
}

type FetchBlockResp struct {
	Height int32
	Block  *Block
}

type FetchAckedQueryReq struct {
	SignedResponseHeaderHash *hash.Hash
}

type FetchAckedQueryResp struct {
	Ack *worker.Ack
}

func (s *ChainRPCServer) AdviseNewBlock(req *AdviseNewBlockReq, resp *AdviseNewBlockResp) error {
	return s.chain.CheckAndPushNewBlock(req.Block)
}

func (s *ChainRPCServer) AdviseBinLog(req *AdviseBinLogReq, resp *AdviseBinLogResp) error {
	return nil
}

func (s *ChainRPCServer) AdviseResponsedQuery(
	req *AdviseResponsedQueryReq, resp *AdviseResponsedQueryResp) error {
	return s.chain.CheckAndPushResponsedQuery(req.Query)
}

func (s *ChainRPCServer) AdviseAckedQuery(
	req *AdviseAckedQueryReq, resp *AdviseAckedQueryResp) error {
	return s.chain.CheckAndPushAckedQuery(req.Query)
}

func (s *ChainRPCServer) FetchBlock(req *FetchBlockReq, resp *FetchBlockResp) (err error) {
	resp.Height = req.Height
	resp.Block, err = s.chain.FetchBlock(req.Height)
	return
}

func (s *ChainRPCServer) FetchAckedQuery(req *FetchAckedQueryReq, resp *FetchAckedQueryResp,
) (err error) {
	resp.Ack, err = s.chain.FetchAckedQuery(req.SignedResponseHeaderHash)
	return
}
