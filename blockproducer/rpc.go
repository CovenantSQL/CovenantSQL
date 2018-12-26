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
	"context"
	"time"

	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/pkg/errors"
)

// ChainRPCService defines a main chain RPC server.
type ChainRPCService struct {
	chain *Chain
}

// AdviseNewBlock is the RPC method to advise a new block to target server.
func (s *ChainRPCService) AdviseNewBlock(req *types.AdviseNewBlockReq, resp *types.AdviseNewBlockResp) error {
	s.chain.pendingBlocks <- req.Block
	return nil
}

// FetchBlock is the RPC method to fetch a known block from the target server.
func (s *ChainRPCService) FetchBlock(req *types.FetchBlockReq, resp *types.FetchBlockResp) error {
	resp.Height = req.Height
	block, count, err := s.chain.fetchBlockByHeight(req.Height)
	if err != nil {
		return err
	}
	resp.Block = block
	resp.Count = count
	return err
}

// FetchLastIrreversibleBlock fetches the last block irreversible block from block producer.
func (s *ChainRPCService) FetchLastIrreversibleBlock(
	req *types.FetchLastIrreversibleBlockReq, resp *types.FetchLastIrreversibleBlockResp) error {
	b, c, h, err := s.chain.fetchLastIrreversibleBlock()
	if err != nil {
		return err
	}
	resp.Block = b
	resp.Count = c
	resp.Height = h
	resp.SQLChains = s.chain.loadSQLChainProfiles(req.Address)
	return nil
}

// FetchBlockByCount is the RPC method to fetch a known block from the target server.
func (s *ChainRPCService) FetchBlockByCount(req *types.FetchBlockByCountReq, resp *types.FetchBlockResp) error {
	resp.Count = req.Count
	block, height, err := s.chain.fetchBlockByCount(req.Count)
	if err != nil {
		return err
	}
	resp.Block = block
	resp.Height = height
	return err
}

// FetchTxBilling is the RPC method to fetch a known billing tx from the target server.
func (s *ChainRPCService) FetchTxBilling(req *types.FetchTxBillingReq, resp *types.FetchTxBillingResp) error {
	return nil
}

// NextAccountNonce is the RPC method to query the next nonce of an account.
func (s *ChainRPCService) NextAccountNonce(
	req *types.NextAccountNonceReq, resp *types.NextAccountNonceResp) (err error,
) {
	if resp.Nonce, err = s.chain.nextNonce(req.Addr); err != nil {
		return
	}
	resp.Addr = req.Addr
	return
}

// AddTx is the RPC method to add a transaction.
func (s *ChainRPCService) AddTx(req *types.AddTxReq, resp *types.AddTxResp) (err error) {
	if req.Tx == nil {
		return ErrUnknownTransactionType
	}
	log.Infof("transaction type: %s, hash: %s, address: %s",
		req.Tx.GetTransactionType().String(), req.Tx.Hash(), req.Tx.GetAccountAddress())
	s.chain.addTx(req.Tx)
	return
}

// QueryAccountStableBalance is the RPC method to query account stable coin balance.
func (s *ChainRPCService) QueryAccountStableBalance(
	req *types.QueryAccountStableBalanceReq, resp *types.QueryAccountStableBalanceResp) (err error,
) {
	resp.Addr = req.Addr
	resp.Balance, resp.OK = s.chain.loadAccountStableBalance(req.Addr)
	return
}

// QueryAccountCovenantBalance is the RPC method to query account covenant coin balance.
func (s *ChainRPCService) QueryAccountCovenantBalance(
	req *types.QueryAccountCovenantBalanceReq, resp *types.QueryAccountCovenantBalanceResp) (err error,
) {
	resp.Addr = req.Addr
	resp.Balance, resp.OK = s.chain.loadAccountCovenantBalance(req.Addr)
	return
}

// QuerySQLChainProfile is the RPC method to query SQLChainProfile.
func (s *ChainRPCService) QuerySQLChainProfile(req *types.QuerySQLChainProfileReq,
	resp *types.QuerySQLChainProfileResp) (err error) {
	p, ok := s.chain.loadSQLChainProfile(req.DBID)
	if ok {
		resp.Profile = *p
		return
	}
	err = errors.Wrap(ErrDatabaseNotFound, "rpc query sqlchain profile failed")
	return
}

// Sub is the RPC method to subscribe some event.
func (s *ChainRPCService) Sub(req *types.SubReq, resp *types.SubResp) (err error) {
	return s.chain.bs.Subscribe(req.Topic, func(request interface{}, response interface{}) {
		s.chain.cl.CallNode(req.NodeID.ToNodeID(), req.Callback, request, response)
	})
}

func WaitDatabaseCreation(
	ctx context.Context, dbid proto.DatabaseID, period time.Duration) (err error,
) {
	var (
		timer = time.NewTimer(0)
		req   = &types.QuerySQLChainProfileReq{
			DBID: dbid,
		}
		resp = &types.QuerySQLChainProfileResp{}
	)
	defer func() {
		if !timer.Stop() {
			<-timer.C
		}
	}()
	for {
		select {
		case <-timer.C:
			if err = rpc.RequestBP(
				route.MCCQuerySQLChainProfile.String(), req, resp,
			); err != ErrDatabaseNotFound {
				// err == nil (creation done), or
				// err != nil && err != ErrDatabaseNotFound (unexpected error)
				return
			}
			timer.Reset(period)
		case <-ctx.Done():
			err = ctx.Err()
			return
		}
	}
}
