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

package blockproducer

import "gitlab.com/thunderdb/ThunderDB/blockproducer/types"

const (
	// MainChainRPCName defines rpc service name of main chain internal consensus.
	MainChainRPCName = "MCC"
)

// ChainRPCService defines a main chain RPC server.
type ChainRPCService struct {
	chain *Chain
}

// AdviseNewBlockReq defines a request of the AdviseNewBlock RPC method.
type AdviseNewBlockReq struct {
	Block *types.Block
}

// AdviseNewBlockResp defines a response of the AdviseNewBlock RPC method.
type AdviseNewBlockResp struct {
}

// AdviseTxBillingReq defines a request of the AdviseTxBilling RPC method.
type AdviseTxBillingReq struct {
	TxBilling *types.TxBilling
}

// AdviseTxBillingResp defines a response of the AdviseTxBilling RPC method.
type AdviseTxBillingResp struct {
}

// AdviseBillingReq defines a request of the AdviseBillingRequest RPC method.
type AdviseBillingReq struct {
	Req *types.BillingRequest
}

// AdviseBillingResp defines a request of the AdviseBillingRequest RPC method.
type AdviseBillingResp struct {
	Resp *types.BillingResponse
}

// FetchBlockReq defines a request of the FetchBlock RPC method
type FetchBlockReq struct {
	Height uint64
}

// FetchBlockResp defines a response of the FetchBlock RPC method
type FetchBlockResp struct {
	Height uint64
	Block  *types.Block
}

// FetchTxBillingReq defines a request of the FetchTxBilling RPC method
type FetchTxBillingReq struct {
}

// FetchTxBillingResp defines a response of the FetchTxBilling RPC method
type FetchTxBillingResp struct {
}

// AdviseNewBlock is the RPC method to advise a new block to target server
func (s *ChainRPCService) AdviseNewBlock(req *AdviseNewBlockReq, resp *AdviseNewBlockResp) error {
	return s.chain.pushBlock(req.Block)
}

// AdviseTxBilling is the RPC method to advise a new billing tx to target server
func (s *ChainRPCService) AdviseTxBilling(req *AdviseTxBillingReq, resp *AdviseTxBillingResp) error {
	return s.chain.pushTxBilling(req.TxBilling)
}

// AdviseBillingRequest is the RPC method to advise a new billing request to main chain
func (s *ChainRPCService) AdviseBillingRequest(req *AdviseBillingReq, resp *AdviseBillingResp) error {
	response, err := s.chain.produceTxBilling(req.Req)
	if err != nil {
		return err
	}
	resp.Resp = response
	return nil
}

// FetchBlock is the RPC method to fetch a known block form the target server.
func (s *ChainRPCService) FetchBlock(req *FetchBlockReq, resp *FetchBlockResp) error {
	resp.Height = req.Height
	block, err := s.chain.fetchBlockByHeight(req.Height)
	resp.Block = block
	return err
}

// FetchTxBilling is the RPC method to fetch a known billing tx form the target server.
func (s *ChainRPCService) FetchTxBilling(req *FetchTxBillingReq, resp *FetchTxBillingResp) error {
	return nil
}
