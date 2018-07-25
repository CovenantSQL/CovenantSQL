/*
 * Copyright 2018 The ThunderDB Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the “License”);
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an “AS IS” BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package blockproducer

import "gitlab.com/thunderdb/ThunderDB/blockproducer/types"

const (
	// MainChainRPCName defines rpc service name of main chain internal consensus.
	MainChainRPCName = "MCC"
)

type ChainRPCService struct {
	chain *Chain
}

type AdviseNewBlockReq struct {
	Block *types.Block
}

type AdviseNewBlockResp struct {
}

type AdviseTxBillingReq struct {
}

type AdviseTxBillingResp struct {
}

type FetchBlockReq struct {
}

type FetchBlockResp struct {
}

type FetchTxBillingReq struct {
}

type FetchTxBillingResp struct {
}

func (s *ChainRPCService) AdviseNewBlock(req *AdviseNewBlockReq, resp *AdviseNewBlockResp) error {
	return s.chain.pushBlock(req.Block)
}

func (s *ChainRPCService) AdviseTxBilling(req *AdviseTxBillingReq, resp *AdviseTxBillingResp) error {
	return nil
}

func (s *ChainRPCService) FetchBlock(req *FetchBlockReq, resp *FetchBlockResp) error {
	return nil
}

func (s *ChainRPCService) FetchTxBilling(req *FetchTxBillingReq, resp *FetchTxBillingResp) error {
	return nil
}
