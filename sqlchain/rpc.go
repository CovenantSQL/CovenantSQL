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
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
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

// AdviseBinLogReq defines a request of the AdviseBinLog RPC method.
type AdviseBinLogReq struct {
}

// AdviseBinLogResp defines a response of the AdviseBinLog RPC method.
type AdviseBinLogResp struct {
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
	Block  *types.Block
}

// SignBillingReq defines a request of the SignBilling RPC method.
type SignBillingReq struct {
	types.BillingRequest
}

// SignBillingResp defines a response of the SignBilling RPC method.
type SignBillingResp struct {
	HeaderHash hash.Hash
	Signee     *asymmetric.PublicKey
	Signature  *asymmetric.Signature
}

// LaunchBillingReq defines a request of LaunchBilling RPC method.
type LaunchBillingReq struct {
	Low, High int32
}

// LaunchBillingResp defines a response of LaunchBilling RPC method.
type LaunchBillingResp struct {
}

// SubscribeTransactionsReq defines a request of SubscribeTransaction RPC method.
type SubscribeTransactionsReq struct {
	SubscriberID proto.NodeID
	Height       int32
}

// SubscribeTransactionsResp defines a response of SubscribeTransaction RPC method.
type SubscribeTransactionsResp struct {
}

// CancelSubscriptionReq defines a request of CancelSubscription RPC method.
type CancelSubscriptionReq struct {
	SubscriberID proto.NodeID
}

// CancelSubscriptionResp defines a response of CancelSubscription RPC method.
type CancelSubscriptionResp struct{}

// AdviseNewBlock is the RPC method to advise a new produced block to the target server.
func (s *ChainRPCService) AdviseNewBlock(req *AdviseNewBlockReq, resp *AdviseNewBlockResp) (
	err error) {
	s.chain.blocks <- req.Block
	return
}

// AdviseBinLog is the RPC method to advise a new binary log to the target server.
func (s *ChainRPCService) AdviseBinLog(req *AdviseBinLogReq, resp *AdviseBinLogResp) error {
	// TODO(leventeliu): need implementation.
	return nil
}

// AdviseAckedQuery is the RPC method to advise a new acknowledged query to the target server.
func (s *ChainRPCService) AdviseAckedQuery(
	req *AdviseAckedQueryReq, resp *AdviseAckedQueryResp) error {
	return s.chain.VerifyAndPushAckedQuery(req.Query)
}

// FetchBlock is the RPC method to fetch a known block from the target server.
func (s *ChainRPCService) FetchBlock(req *FetchBlockReq, resp *FetchBlockResp) (err error) {
	resp.Height = req.Height
	resp.Block, err = s.chain.FetchBlock(req.Height)
	return
}

// SignBilling is the RPC method to get signature for a billing request from the target server.
func (s *ChainRPCService) SignBilling(req *SignBillingReq, resp *SignBillingResp) (err error) {
	resp.HeaderHash = req.BillingRequest.RequestHash
	resp.Signee, resp.Signature, err = s.chain.SignBilling(&req.BillingRequest)
	return
}

// LaunchBilling is the RPC method to launch a new billing process in the target server.
func (s *ChainRPCService) LaunchBilling(req *LaunchBillingReq, _ *LaunchBillingResp) error {
	return s.chain.LaunchBilling(req.Low, req.High)
}

// SubscribeTransactions is the RPC method to fetch subscribe new packed and confirmed transactions from the target server.
func (s *ChainRPCService) SubscribeTransactions(req *SubscribeTransactionsReq, _ *SubscribeTransactionsResp) error {
	return s.chain.addSubscription(req.SubscriberID, req.Height)
}

// CancelSubscription is the RPC method to cancel subscription in the target server.
func (s *ChainRPCService) CancelSubscription(req *CancelSubscriptionReq, _ *CancelSubscriptionResp) error {
	return s.chain.cancelSubscription(req.SubscriberID)
}
