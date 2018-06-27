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
	"gitlab.com/thunderdb/ThunderDB/worker"
)

type ChainRPCServer struct {
	chain *Chain
}

type AdviseNewBlockReq struct {
	block *Block
}

type AdviseNewBlockResp struct {
}

type AdviseBinLogReq struct {
}

type AdviseBinLogResp struct {
}

type AdviseResponsedQueryReq struct {
	query *worker.SignedResponseHeader
}

type AdviseResponsedQueryResp struct {
}

type AdviseAckedQueryReq struct {
	query *worker.SignedAckHeader
}

type AdviseAckedQueryResp struct {
}

func (s *ChainRPCServer) AdviseNewBlock(req *AdviseNewBlockReq, resp *AdviseNewBlockResp) error {
	return s.chain.CheckAndPushNewBlock(req.block)
}

func (s *ChainRPCServer) AdviseBinLog(req *AdviseBinLogReq, resp *AdviseBinLogResp) error {
	return nil
}

func (s *ChainRPCServer) AdviseResponsedQuery(
	req *AdviseResponsedQueryReq, resp *AdviseResponsedQueryResp) error {
	return s.chain.CheckAndPushResponsedQuery(req.query)
}

func (s *ChainRPCServer) AdviseAckedQuery(
	req *AdviseAckedQueryReq, resp *AdviseAckedQueryResp) error {
	return s.chain.CheckAndPushAckedQuery(req.query)
}
