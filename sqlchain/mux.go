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
	"sync"

	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/rpc"
)

// MuxService defines multiplexing service of sql-chain.
type MuxService struct {
	ServiceName string
	serviceMap  sync.Map
}

// NewMuxService creates a new multiplexing service and registers it to rpc server.
func NewMuxService(serviceName string, server *rpc.Server) (service *MuxService, err error) {
	service = &MuxService{
		ServiceName: serviceName,
	}

	err = server.RegisterService(serviceName, service)
	return
}

func (s *MuxService) register(id proto.DatabaseID, service *ChainRPCService) {
	s.serviceMap.Store(id, service)
}

func (s *MuxService) unregister(id proto.DatabaseID) {
	s.serviceMap.Delete(id)
}

// MuxAdviseNewBlockReq defines a request of the AdviseNewBlock RPC method.
type MuxAdviseNewBlockReq struct {
	proto.Envelope
	proto.DatabaseID
	AdviseNewBlockReq
}

// MuxAdviseNewBlockResp defines a response of the AdviseNewBlock RPC method.
type MuxAdviseNewBlockResp struct {
	proto.Envelope
	proto.DatabaseID
	AdviseNewBlockResp
}

// MuxAdviseBinLogReq defines a request of the AdviseBinLog RPC method.
type MuxAdviseBinLogReq struct {
	proto.Envelope
	proto.DatabaseID
	AdviseBinLogReq
}

// MuxAdviseBinLogResp defines a response of the AdviseBinLog RPC method.
type MuxAdviseBinLogResp struct {
	proto.Envelope
	proto.DatabaseID
	AdviseBinLogResp
}

// MuxAdviseAckedQueryReq defines a request of the AdviseAckedQuery RPC method.
type MuxAdviseAckedQueryReq struct {
	proto.Envelope
	proto.DatabaseID
	AdviseAckedQueryReq
}

// MuxAdviseAckedQueryResp defines a response of the AdviseAckedQuery RPC method.
type MuxAdviseAckedQueryResp struct {
	proto.Envelope
	proto.DatabaseID
	AdviseAckedQueryResp
}

// MuxFetchBlockReq defines a request of the FetchBlock RPC method.
type MuxFetchBlockReq struct {
	proto.Envelope
	proto.DatabaseID
	FetchBlockReq
}

// MuxFetchBlockResp defines a response of the FetchBlock RPC method.
type MuxFetchBlockResp struct {
	proto.Envelope
	proto.DatabaseID
	FetchBlockResp
}

// MuxSignBillingReq defines a request of the SignBilling RPC method.
type MuxSignBillingReq struct {
	proto.Envelope
	proto.DatabaseID
	SignBillingReq
}

// MuxSignBillingResp defines a response of the SignBilling RPC method.
type MuxSignBillingResp struct {
	proto.Envelope
	proto.DatabaseID
	SignBillingResp
}

// MuxLaunchBillingReq defines a request of the LaunchBilling RPC method.
type MuxLaunchBillingReq struct {
	proto.Envelope
	proto.DatabaseID
	LaunchBillingReq
}

// MuxLaunchBillingResp defines a response of the LaunchBilling RPC method.
type MuxLaunchBillingResp struct {
	proto.Envelope
	proto.DatabaseID
	LaunchBillingResp
}

// MuxSubscribeTransactionsReq defines a request of the SubscribeTransactions RPC method.
type MuxSubscribeTransactionsReq struct {
	proto.Envelope
	proto.DatabaseID
	SubscribeTransactionsReq
}

// MuxSubscribeTransactionsResp defines a response of the SubscribeTransactions RPC method.
type MuxSubscribeTransactionsResp struct {
	proto.Envelope
	proto.DatabaseID
	SubscribeTransactionsResp
}

// MuxCancelSubscriptionReq defines a request of the CancelSubscription RPC method.
type MuxCancelSubscriptionReq struct {
	proto.Envelope
	proto.DatabaseID
	CancelSubscriptionReq
}

// MuxCancelSubscriptionResp defines a response of the CancelSubscription RPC method.
type MuxCancelSubscriptionResp struct {
	proto.Envelope
	proto.DatabaseID
	CancelSubscriptionResp
}

// AdviseNewBlock is the RPC method to advise a new produced block to the target server.
func (s *MuxService) AdviseNewBlock(req *MuxAdviseNewBlockReq, resp *MuxAdviseNewBlockResp) error {
	if v, ok := s.serviceMap.Load(req.DatabaseID); ok {
		resp.Envelope = req.Envelope
		resp.DatabaseID = req.DatabaseID
		return v.(*ChainRPCService).AdviseNewBlock(&req.AdviseNewBlockReq, &resp.AdviseNewBlockResp)
	}

	return ErrUnknownMuxRequest
}

// AdviseBinLog is the RPC method to advise a new binary log to the target server.
func (s *MuxService) AdviseBinLog(req *MuxAdviseBinLogReq, resp *MuxAdviseBinLogResp) error {
	if v, ok := s.serviceMap.Load(req.DatabaseID); ok {
		resp.Envelope = req.Envelope
		resp.DatabaseID = req.DatabaseID
		return v.(*ChainRPCService).AdviseBinLog(&req.AdviseBinLogReq, &resp.AdviseBinLogResp)
	}

	return ErrUnknownMuxRequest
}

// AdviseAckedQuery is the RPC method to advise a new acknowledged query to the target server.
func (s *MuxService) AdviseAckedQuery(
	req *MuxAdviseAckedQueryReq, resp *MuxAdviseAckedQueryResp) error {
	if v, ok := s.serviceMap.Load(req.DatabaseID); ok {
		resp.Envelope = req.Envelope
		resp.DatabaseID = req.DatabaseID
		return v.(*ChainRPCService).AdviseAckedQuery(
			&req.AdviseAckedQueryReq, &resp.AdviseAckedQueryResp)
	}

	return ErrUnknownMuxRequest
}

// FetchBlock is the RPC method to fetch a known block from the target server.
func (s *MuxService) FetchBlock(req *MuxFetchBlockReq, resp *MuxFetchBlockResp) (err error) {
	if v, ok := s.serviceMap.Load(req.DatabaseID); ok {
		resp.Envelope = req.Envelope
		resp.DatabaseID = req.DatabaseID
		return v.(*ChainRPCService).FetchBlock(&req.FetchBlockReq, &resp.FetchBlockResp)
	}

	return ErrUnknownMuxRequest
}

// SignBilling is the RPC method to get signature for a billing request from the target server.
func (s *MuxService) SignBilling(req *MuxSignBillingReq, resp *MuxSignBillingResp) (err error) {
	if v, ok := s.serviceMap.Load(req.DatabaseID); ok {
		resp.Envelope = req.Envelope
		resp.DatabaseID = req.DatabaseID
		return v.(*ChainRPCService).SignBilling(&req.SignBillingReq, &resp.SignBillingResp)
	}

	return ErrUnknownMuxRequest
}

// LaunchBilling is the RPC method to launch a new billing process in the target server.
func (s *MuxService) LaunchBilling(req *MuxLaunchBillingReq, resp *MuxLaunchBillingResp) (
	err error,
) {
	if v, ok := s.serviceMap.Load(req.DatabaseID); ok {
		resp.Envelope = req.Envelope
		resp.DatabaseID = req.DatabaseID
		return v.(*ChainRPCService).LaunchBilling(&req.LaunchBillingReq, &resp.LaunchBillingResp)
	}

	return ErrUnknownMuxRequest
}

// SubscribeTransactions is the RPC method to subscribe transactions from the target server.
func (s *MuxService) SubscribeTransactions(req *MuxSubscribeTransactionsReq, resp *MuxSubscribeTransactionsResp) (err error) {
	if v, ok := s.serviceMap.Load(req.DatabaseID); ok {
		resp.Envelope = req.Envelope
		resp.DatabaseID = req.DatabaseID
		req.SubscribeTransactionsReq.SubscriberID = req.GetNodeID().ToNodeID()
		return v.(*ChainRPCService).SubscribeTransactions(&req.SubscribeTransactionsReq, &resp.SubscribeTransactionsResp)
	}

	return ErrUnknownMuxRequest
}

// CancelSubscription is the RPC method to cancel subscription from the target server.
func (s *MuxService) CancelSubscription(req *MuxCancelSubscriptionReq, resp *MuxCancelSubscriptionResp) (err error) {
	if v, ok := s.serviceMap.Load(req.DatabaseID); ok {
		resp.Envelope = req.Envelope
		resp.DatabaseID = req.DatabaseID
		req.CancelSubscriptionReq.SubscriberID = req.GetNodeID().ToNodeID()
		return v.(*ChainRPCService).CancelSubscription(&req.CancelSubscriptionReq, &resp.CancelSubscriptionResp)
	}

	return ErrUnknownMuxRequest
}
