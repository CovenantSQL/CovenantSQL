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
	"database/sql"
	"fmt"
	"strings"
	"time"

	pi "github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	"github.com/CovenantSQL/CovenantSQL/crypto"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	"github.com/CovenantSQL/CovenantSQL/types"
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
func (s *ChainRPCService) AddTx(req *types.AddTxReq, _ *types.AddTxResp) (err error) {
	s.chain.addTx(req)
	return
}

// QueryAccountTokenBalance is the RPC method to query account token balance.
func (s *ChainRPCService) QueryAccountTokenBalance(
	req *types.QueryAccountTokenBalanceReq, resp *types.QueryAccountTokenBalanceResp) (err error,
) {
	resp.Addr = req.Addr
	resp.Balance, resp.OK = s.chain.loadAccountTokenBalance(req.Addr, req.TokenType)
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

// QueryTxState is the RPC method to query a transaction state.
func (s *ChainRPCService) QueryTxState(
	req *types.QueryTxStateReq, resp *types.QueryTxStateResp) (err error,
) {
	var state pi.TransactionState
	if state, err = s.chain.queryTxState(req.Hash); err != nil {
		return
	}
	resp.Hash = req.Hash
	resp.State = state
	return
}

// Sub is the RPC method to subscribe some event.
func (s *ChainRPCService) Sub(req *types.SubReq, resp *types.SubResp) (err error) {
	return s.chain.chainBus.Subscribe(req.Topic, func(request interface{}, response interface{}) {
		s.chain.caller.CallNode(req.NodeID.ToNodeID(), req.Callback, request, response)
	})
}

// WaitDatabaseCreation waits for database creation complete.
func WaitDatabaseCreation(
	ctx context.Context,
	dbID proto.DatabaseID,
	db *sql.DB,
	period time.Duration,
) (err error) {
	var (
		ticker = time.NewTicker(period)
		req    = &types.QuerySQLChainProfileReq{
			DBID: dbID,
		}
	)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err = rpc.RequestBP(
				route.MCCQuerySQLChainProfile.String(), req, nil,
			); err != nil {
				if !strings.Contains(err.Error(), ErrDatabaseNotFound.Error()) {
					// err != nil && err != ErrDatabaseNotFound (unexpected error)
					return
				}
			} else {
				// err == nil (creation done on BP): try to use database connection
				if db == nil {
					return
				}
				if _, err = db.ExecContext(ctx, "SHOW TABLES"); err == nil {
					// err == nil (connect to Miner OK)
					return
				}
			}
		case <-ctx.Done():
			err = ctx.Err()
			return
		}
	}
}

// WaitBPChainService waits until BP chain service is ready.
func WaitBPChainService(ctx context.Context, period time.Duration) (err error) {
	var (
		ticker = time.NewTicker(period)
		req    = &types.FetchBlockReq{
			Height: 0, // Genesis block
		}
	)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err = rpc.RequestBP(
				route.MCCFetchBlock.String(), req, nil,
			); err == nil || !strings.Contains(err.Error(), "can't find service") {
				return
			}
		case <-ctx.Done():
			err = ctx.Err()
			return
		}
	}
}

// Create allocates new database.
func Create(
	meta types.ResourceMeta,
	gasPrice uint64,
	advancePayment uint64,
	privateKey *asymmetric.PrivateKey,
) (
	dbID proto.DatabaseID, dsn string, err error,
) {
	var (
		nonceReq   = new(types.NextAccountNonceReq)
		nonceResp  = new(types.NextAccountNonceResp)
		req        = new(types.AddTxReq)
		resp       = new(types.AddTxResp)
		clientAddr proto.AccountAddress
	)
	if clientAddr, err = crypto.PubKeyHash(privateKey.PubKey()); err != nil {
		err = errors.Wrap(err, "get local account address failed")
		return
	}
	// allocate nonce
	nonceReq.Addr = clientAddr

	if err = rpc.RequestBP(route.MCCNextAccountNonce.String(), nonceReq, nonceResp); err != nil {
		err = errors.Wrap(err, "allocate create database transaction nonce failed")
		return
	}

	req.Tx = types.NewCreateDatabase(&types.CreateDatabaseHeader{
		Owner:          clientAddr,
		ResourceMeta:   meta,
		GasPrice:       gasPrice,
		AdvancePayment: advancePayment,
		TokenType:      types.Particle,
		Nonce:          nonceResp.Nonce,
	})

	if err = req.Tx.Sign(privateKey); err != nil {
		err = errors.Wrap(err, "sign request failed")
		return
	}

	if err = rpc.RequestBP(route.MCCAddTx.String(), req, resp); err != nil {
		err = errors.Wrap(err, "call create database transaction failed")
		return
	}

	dbID = proto.FromAccountAndNonce(clientAddr, uint32(nonceResp.Nonce))
	dsn = fmt.Sprintf("cql://%s", string(dbID))
	return
}
