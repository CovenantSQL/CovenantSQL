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

package blockproducer

import (
	pi "github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	"github.com/CovenantSQL/CovenantSQL/blockproducer/types"
	ci "github.com/CovenantSQL/CovenantSQL/chain/interfaces"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/proto"
)

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
	proto.Envelope
	Block *types.Block
}

// AdviseNewBlockResp defines a response of the AdviseNewBlock RPC method.
type AdviseNewBlockResp struct {
	proto.Envelope
}

// AdviseTxBillingReq defines a request of the AdviseTxBilling RPC method.
type AdviseTxBillingReq struct {
	proto.Envelope
	TxBilling *types.TxBilling
}

// AdviseTxBillingResp defines a response of the AdviseTxBilling RPC method.
type AdviseTxBillingResp struct {
	proto.Envelope
}

// AdviseBillingReq defines a request of the AdviseBillingRequest RPC method.
type AdviseBillingReq struct {
	proto.Envelope
	Req *types.BillingRequest
}

// AdviseBillingResp defines a request of the AdviseBillingRequest RPC method.
type AdviseBillingResp struct {
	proto.Envelope
	Resp *types.BillingResponse
}

// FetchBlockReq defines a request of the FetchBlock RPC method.
type FetchBlockReq struct {
	proto.Envelope
	Height uint32
}

// FetchBlockResp defines a response of the FetchBlock RPC method.
type FetchBlockResp struct {
	proto.Envelope
	Height uint32
	Block  *types.Block
}

// FetchTxBillingReq defines a request of the FetchTxBilling RPC method.
type FetchTxBillingReq struct {
	proto.Envelope
}

// FetchTxBillingResp defines a response of the FetchTxBilling RPC method.
type FetchTxBillingResp struct {
	proto.Envelope
}

// FetchTxReq defines a request of the FetchTx RPC method.
type FetchTxReq struct {
	proto.Envelope
	Hash hash.Hash
}

// FetchTxResp defines a response of the FetchTx RPC method.
type FetchTxResp struct {
	proto.Envelope
	Ok bool
	Tx ci.Transaction
}

// NextAccountNonceReq defines a request of the NextAccountNonce RPC method.
type NextAccountNonceReq struct {
	proto.Envelope
	Addr proto.AccountAddress
}

// NextAccountNonceResp defines a response of the NextAccountNonce RPC method.
type NextAccountNonceResp struct {
	proto.Envelope
	Addr  proto.AccountAddress
	Nonce pi.AccountNonce
}

// AddTxReq defines a request of the AddTx RPC method.
type AddTxReq struct {
	proto.Envelope
	Tx pi.Transaction
}

// AddTxResp defines a response of the AddTx RPC method.
type AddTxResp struct {
	proto.Envelope
}

// QueryAccountStableBalanceReq defines a request of the QueryAccountStableBalance RPC method.
type QueryAccountStableBalanceReq struct {
	proto.Envelope
	Addr proto.AccountAddress
}

// QueryAccountStableBalanceResp defines a request of the QueryAccountStableBalance RPC method.
type QueryAccountStableBalanceResp struct {
	proto.Envelope
	Addr    proto.AccountAddress
	OK      bool
	Balance uint64
}

// QueryAccountCovenantBalanceReq defines a request of the QueryAccountCovenantBalance RPC method.
type QueryAccountCovenantBalanceReq struct {
	proto.Envelope
	Addr proto.AccountAddress
}

// QueryAccountCovenantBalanceResp defines a request of the QueryAccountCovenantBalance RPC method.
type QueryAccountCovenantBalanceResp struct {
	proto.Envelope
	Addr    proto.AccountAddress
	OK      bool
	Balance uint64
}

// AdviseNewBlock is the RPC method to advise a new block to target server.
func (s *ChainRPCService) AdviseNewBlock(req *AdviseNewBlockReq, resp *AdviseNewBlockResp) error {
	s.chain.blocksFromRPC <- req.Block
	return s.chain.pushBlock(req.Block)
}

// AdviseTxBilling is the RPC method to advise a new billing tx to target server.
func (s *ChainRPCService) AdviseTxBilling(req *AdviseTxBillingReq, resp *AdviseTxBillingResp) error {
	return s.chain.pushTxBilling(req.TxBilling)
}

// AdviseBillingRequest is the RPC method to advise a new billing request to main chain.
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

// FetchTx is the RPC method to fetch a transaction from the target server.
func (s *ChainRPCService) FetchTx(req *FetchTxReq, resp *FetchTxResp) (err error) {
	resp.Ok, err = s.chain.fetchTx(req.Hash, resp.Tx)
	return
}

// NextAccountNonce is the RPC method to query the next nonce of an account.
func (s *ChainRPCService) NextAccountNonce(
	req *NextAccountNonceReq, resp *NextAccountNonceResp) (err error,
) {
	if resp.Nonce, err = s.chain.ms.nextNonce(req.Addr); err != nil {
		return
	}
	resp.Addr = req.Addr
	return
}

// AddTx is the RPC method to add a transaction.
func (s *ChainRPCService) AddTx(req *AddTxReq, resp *AddTxResp) (err error) {
	s.chain.pendingTxs <- req.Tx
	return
}

// QueryAccountStableBalance is the RPC method to query acccount stable coin balance.
func (s *ChainRPCService) QueryAccountStableBalance(
	req *QueryAccountStableBalanceReq, resp *QueryAccountStableBalanceResp) (err error,
) {
	resp.Addr = req.Addr
	resp.Balance, resp.OK = s.chain.ms.loadAccountStableBalance(req.Addr)
	return
}

// QueryAccountCovenantBalance is the RPC method to query acccount covenant coin balance.
func (s *ChainRPCService) QueryAccountCovenantBalance(
	req *QueryAccountCovenantBalanceReq, resp *QueryAccountCovenantBalanceResp) (err error,
) {
	resp.Addr = req.Addr
	resp.Balance, resp.OK = s.chain.ms.loadAccountCovenantBalance(req.Addr)
	return
}
