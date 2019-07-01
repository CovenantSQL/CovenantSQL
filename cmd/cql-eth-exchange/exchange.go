/*
 * Copyright 2019 The CovenantSQL Authors.
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

package main

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	gorp "gopkg.in/gorp.v2"

	cinter "github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	"github.com/CovenantSQL/CovenantSQL/client"
	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/rpc/mux"
	ctypes "github.com/CovenantSQL/CovenantSQL/types"
)

// ExchangeRequestMagic defines the magic bytes in ethereum transaction for malformed transaction detection.
var ExchangeRequestMagic = []byte{0xC0, 0x4E, 0xA4, 0x71}

// ExchangeConfig defines the exchange config object for yaml config marshal/unmarshal.
type ExchangeConfig struct {
	Endpoint         string           `json:"endpoint" yaml:"endpoint" validate:"required,url"`
	Database         string           `json:"database" yaml:"database" validate:"required"`
	UseLocalDatabase bool             `json:"use_local_database" yaml:"use_local_database"`
	FromBlock        *big.Int         `json:"from_block" yaml:"from_block"`
	ReceiveAddresses []common.Address `json:"eth_addresses" yaml:"eth_addresses"`
	ExchangeRate     *big.Float       `json:"exchange_rate" yaml:"exchange_rate"`
	TokenType        ctypes.TokenType `json:"token_type" yaml:"token_type"`
	DelayBlockCount  uint64           `json:"delay_block_count"`
}

// Exchange defines the exchange instance object.
type Exchange struct {
	client           *ethclient.Client
	rawClient        *rpc.Client
	subClient        *ethclient.Client // separate subscribe client and rpc client for stability
	subscription     ethereum.Subscription
	newBlockCh       chan *types.Block
	newBlockHeaderCh chan *types.Header
	currentBlock     uint64
	chainID          int64
	chainCfg         *params.ChainConfig
	receivedBlockMap map[uint64]bool
	processCtx       context.Context
	cancelProcess    context.CancelFunc
	receiveAddrMap   map[string]bool
	vaultPrivateKey  *asymmetric.PrivateKey
	vaultAddress     proto.AccountAddress
	metaDatabase     *gorp.DbMap
	cfg              *ExchangeConfig
	wg               *sync.WaitGroup
}

// NewExchange creates new exchange process object.
func NewExchange(cfg *ExchangeConfig) (e *Exchange, err error) {
	e = &Exchange{
		cfg:              cfg,
		receivedBlockMap: make(map[uint64]bool),
		receiveAddrMap:   make(map[string]bool),
	}

	// init keys
	e.vaultPrivateKey, err = kms.GetLocalPrivateKey()
	if err != nil {
		return
	}
	e.vaultAddress, err = crypto.PubKeyHash(e.vaultPrivateKey.PubKey())
	if err != nil {
		return
	}

	// init meta database
	var db *sql.DB
	if cfg.UseLocalDatabase {
		db, err = sql.Open("sqlite3", cfg.Database)
	} else {
		dsnCfg := client.Config{}
		dsnCfg.DatabaseID = cfg.Database
		db, err = sql.Open("covenantsql", dsnCfg.FormatDSN())
	}

	e.metaDatabase = &gorp.DbMap{Db: db, Dialect: gorp.SqliteDialect{}}
	e.metaDatabase.AddTableWithName(TxRecord{}, "record").
		SetKeys(false, "Hash")
	e.metaDatabase.AddTableWithName(AuditRecord{}, "audit").
		SetKeys(true, "ID")
	if err = e.metaDatabase.CreateTablesIfNotExists(); err != nil {
		return
	}

	for _, addr := range cfg.ReceiveAddresses {
		e.receiveAddrMap[addr.String()] = true
	}

	return
}

// Start connects to server and listen to new blocks for token exchanges.
func (e *Exchange) Start(ctx context.Context) (err error) {
	e.newBlockCh = make(chan *types.Block)
	e.newBlockHeaderCh = make(chan *types.Header)
	e.wg = &sync.WaitGroup{}

	err = e.connectServer(ctx)
	if err != nil {
		return
	}

	// process old blocks
	err = e.fetchOldBlocks(ctx)

	e.wg.Add(1)
	go e.process()

	return
}

// Stop stops the exchange service and disconnect from the ethereum network.
func (e *Exchange) Stop() {
	e.stopPreviousSubscribe()

	if e.newBlockHeaderCh != nil {
		close(e.newBlockHeaderCh)
		e.newBlockHeaderCh = nil
	}
	if e.cancelProcess != nil {
		e.cancelProcess()
		e.cancelProcess = nil
		e.processCtx = nil
	}
}

func (e *Exchange) stopPreviousSubscribe() {
	if e.subscription != nil {
		e.subscription.Unsubscribe()
		e.subscription = nil
	}
	if e.client != nil {
		e.client.Close()
		e.client = nil
	}
}

func (e *Exchange) getChainID() (chainID int64, err error) {
	var result hexutil.Big
	err = e.rawClient.Call(&result, "eth_chainId")
	if err != nil {
		return
	}
	chainID = result.ToInt().Int64()
	return
}

func (e *Exchange) connectServer(ctx context.Context) (err error) {
	e.stopPreviousSubscribe()

	e.rawClient, err = rpc.DialContext(ctx, e.cfg.Endpoint)
	if err != nil {
		return
	}

	e.client = ethclient.NewClient(e.rawClient)

	e.subClient, err = ethclient.DialContext(ctx, e.cfg.Endpoint)
	if err != nil {
		return
	}

	e.subscription, err = e.subClient.SubscribeNewHead(ctx, e.newBlockHeaderCh)
	if err != nil {
		return
	}

	e.chainID, err = e.getChainID()
	if err != nil {
		return
	}
	switch e.chainID {
	case params.MainnetChainConfig.ChainID.Int64():
		e.chainCfg = params.MainnetChainConfig
	case params.TestnetChainConfig.ChainID.Int64():
		e.chainCfg = params.TestnetChainConfig
	case params.RinkebyChainConfig.ChainID.Int64():
		e.chainCfg = params.RinkebyChainConfig
	case params.GoerliChainConfig.ChainID.Int64():
		e.chainCfg = params.GoerliChainConfig
	default:
		e.chainCfg = params.AllEthashProtocolChanges
	}

	return
}

func (e *Exchange) fetchOldBlocks(ctx context.Context) (err error) {
	var blk *types.Block

	// fetch old block failed
	if blk, err = e.client.BlockByNumber(ctx, nil); err != nil {
		return
	}

	e.newBlockCh <- blk
	targetBlkNumber := blk.NumberU64()

	if e.currentBlock != 0 {
		for i := e.currentBlock; i < targetBlkNumber; {
			blk, err = e.client.BlockByNumber(ctx, new(big.Int).SetUint64(i))
			if err != nil {
				time.Sleep(time.Second)
				continue
			}

			e.newBlockCh <- blk
			i++
		}
	}

	return
}

func (e *Exchange) process() {
	if e.newBlockHeaderCh == nil || e.wg == nil {
		return
	}

	defer e.wg.Done()
	processTick := time.NewTicker(10 * conf.GConf.BPPeriod)
	defer processTick.Stop()

	for {
		select {
		case b := <-e.newBlockCh:
			_ = e.processBlock(b)
		case blockHead := <-e.newBlockHeaderCh:
			e.wg.Add(1)
			go e.fetchBlock(blockHead)
		case <-e.processCtx.Done():
			return
		case <-processTick.C:
			// load tx and process
			e.processPendingTxs()
		default:
			// update transferring tx state
			e.updateTransferringTxState()
			// update transferring tx state
			e.expireStaleTx()
		}
	}

}

func (e *Exchange) processBlock(b *types.Block) (err error) {
	// set back to queue
	defer func() {
		if err != nil {
			// set back to queue
			go func() {
				time.Sleep(time.Second)
				e.newBlockCh <- b
			}()
		}
	}()

	blkNumber := b.NumberU64()

	if blkNumber < atomic.LoadUint64(&e.currentBlock) {
		// reorganize current block transactions
		err = InvalidateTx(e.metaDatabase, blkNumber)
		if err != nil {
			return
		}
	}

	// process transactions in block
	for _, t := range b.Body().Transactions {
		if !e.receiveAddrMap[t.To().String()] {
			continue
		}

		// save to database
		if _, err = e.saveTx(b.Number(), t); err != nil {
			return
		}
	}

	// process block complete, advance currentBlock
	e.receivedBlockMap[blkNumber] = true

	for i := atomic.LoadUint64(&e.currentBlock) + 1; e.receivedBlockMap[i]; i++ {
		if atomic.CompareAndSwapUint64(&e.currentBlock, i-1, i) {
			delete(e.receivedBlockMap, i)
		}
	}

	return
}

func (e *Exchange) fetchBlock(head *types.Header) {
	var (
		blk *types.Block
		err error
	)

	defer e.wg.Done()

	for {
		blk, err = e.client.BlockByNumber(e.processCtx, head.Number)
		if err == nil {
			e.newBlockCh <- blk
			return
		}

		time.Sleep(time.Second)
	}
}

func (e *Exchange) saveTx(blockNumber *big.Int, tx *types.Transaction) (r *TxRecord, err error) {
	var receipt *types.Receipt
	if receipt, err = e.client.TransactionReceipt(e.processCtx, tx.Hash()); err != nil {
		return
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		return
	}

	var msg types.Message
	msg, err = tx.AsMessage(types.MakeSigner(e.chainCfg, blockNumber))
	if err != nil {
		_ = AddAuditRecord(e.metaDatabase, &AuditRecord{
			Hash: tx.Hash().String(),
			Op:   "get_sender",
			Data: tx,
		})
		err = nil
		return
	}

	// validate exchange target account
	var accountAddr proto.AccountAddress
	if accountAddr, err = e.extractExchangeInfo(tx.Data()); err != nil {
		_ = AddAuditRecord(e.metaDatabase, &AuditRecord{
			Hash: tx.Hash().String(),
			Op:   "extract_exchange",
			Data: tx,
		})
		err = nil
		return
	}

	var cqlAmount uint64
	if cqlAmount, err = e.getExchangedAmount(tx.Value()); err != nil {
		_ = AddAuditRecord(e.metaDatabase, &AuditRecord{
			Hash: tx.Hash().String(),
			Op:   "get_amount",
			Data: tx,
		})
		err = nil
		return
	}

	r, err = UpsertTx(e.metaDatabase, &TxRecord{
		Hash:           tx.Hash().String(),
		Tx:             tx,
		ETHBlockNumber: blockNumber.Uint64(),
		ETHFromAddr:    msg.From().String(),
		ETHToAddr:      tx.To().String(),
		ETHAmount:      tx.Value().Uint64(),

		// CQL
		CQLAccount: accountAddr.String(),
		CQLAmount:  cqlAmount,

		// State
		IsReverted: 0,
	})

	return
}

func (e *Exchange) getExchangedAmount(ethAmount *big.Int) (cqlAmount uint64, err error) {
	ethAmountFloat := new(big.Float).SetInt(ethAmount)
	cqlAmount, accuracy := new(big.Float).Mul(ethAmountFloat, e.cfg.ExchangeRate).Uint64()
	if accuracy != big.Exact {
		err = errors.New("invalid number of tokens to exchanges")
		return
	}

	return
}

func (e *Exchange) extractExchangeInfo(data []byte) (addr proto.AccountAddress, err error) {
	if !bytes.HasPrefix(data, ExchangeRequestMagic) {
		// invalid request, maybe need refund
		err = errors.New("invalid tx data, head magic not match")
		return
	}

	rawAddr := data[len(ExchangeRequestMagic):]

	if len(rawAddr) != len(addr) {
		err = errors.New("invalid address in transaction")
		return
	}
	copy(addr[:], rawAddr)

	return
}

func (e *Exchange) processPendingTxs() {
	var (
		records []*TxRecord
		err     error
	)

	lastAvailBlock := atomic.LoadUint64(&e.currentBlock) - e.cfg.DelayBlockCount
	records, err = GetToProcessTx(e.metaDatabase, lastAvailBlock)
	if err != nil {
		return
	}

	// get current balance
	var balance uint64
	balance, err = e.getAccountBalance()
	if err != nil {
		return
	}

	// get current nonce
	var nonce cinter.AccountNonce
	nonce, err = e.getAccountNonce()
	if err != nil {
		return
	}

	for _, r := range records {
		if r.CQLAmount > balance {
			// skip this tx, account
			_ = AddAuditRecord(e.metaDatabase, &AuditRecord{
				Hash: r.Hash,
				Op:   "check_balance",
				Data: map[string]interface{}{
					"required": r.CQLAmount,
					"balance":  balance,
				},
			})
			continue
		}

		// send request and update state
		// parse cql account
		var target proto.AccountAddress
		err = hash.Decode((*hash.Hash)(&target), r.CQLAccount)
		if err != nil {
			_ = AddAuditRecord(e.metaDatabase, &AuditRecord{
				Hash: r.Hash,
				Op:   "invalid_target_account",
				Data: map[string]interface{}{
					"account": r.CQLAccount,
				},
			})
		}

		var txHash hash.Hash
		txHash, err = e.transferToken(target, nonce, r.CQLAmount)
		_ = AddAuditRecord(e.metaDatabase, &AuditRecord{
			Hash: r.Hash,
			Op:   "transfer_token",
			Data: map[string]interface{}{
				"target": target,
				"nonce":  nonce,
				"amount": r.CQLAmount,
				"type":   e.cfg.TokenType,
			},
		})

		if err == nil {
			// set record
			if err = SetTxToTransferring(e.metaDatabase, r, txHash); err != nil {
				_ = AddAuditRecord(e.metaDatabase, &AuditRecord{
					Hash: r.Hash,
					Op:   "begin_transfer",
					Data: map[string]interface{}{
						"target": target,
						"nonce":  nonce,
						"amount": r.CQLAmount,
						"type":   e.cfg.TokenType,
					},
				})
			}
		}

	}

	return
}

func (e *Exchange) updateTransferringTxState() {
	var (
		records []*TxRecord
		err     error
	)
	records, err = GetTransferringTx(e.metaDatabase)

	for _, r := range records {
		// get state
		var (
			req  = new(ctypes.QueryTxStateReq)
			resp = new(ctypes.QueryTxStateResp)
		)

		if err = hash.Decode(&req.Hash, r.CQLTxHash); err != nil {
			continue
		}

		err = mux.RequestBP(route.MCCQueryTxState.String(), req, resp)
		if err != nil {
			continue
		}

		if resp.State == cinter.TransactionStateConfirmed {
			// update state
			err = SetTxConfirmed(e.metaDatabase, r)
			_ = AddAuditRecord(e.metaDatabase, &AuditRecord{
				Hash: r.Hash,
				Op:   "transfer_confirmed",
				Data: map[string]interface{}{
					"tx": r.CQLTxHash,
				},
			})
		}
	}
}

func (e *Exchange) expireStaleTx() {

}

func (e *Exchange) getAccountBalance() (balance uint64, err error) {
	var (
		req  = new(ctypes.QueryAccountTokenBalanceReq)
		resp = new(ctypes.QueryAccountTokenBalanceResp)
	)

	req.TokenType = e.cfg.TokenType
	req.Addr = e.vaultAddress

	err = mux.RequestBP(route.MCCQueryAccountTokenBalance.String(), req, resp)
	if err == nil {
		balance = resp.Balance
	}

	return
}

func (e *Exchange) getAccountNonce() (nonce cinter.AccountNonce, err error) {
	var (
		req  = new(ctypes.NextAccountNonceReq)
		resp = new(ctypes.NextAccountNonceResp)
	)

	req.Addr = e.vaultAddress

	err = mux.RequestBP(route.MCCNextAccountNonce.String(), req, resp)
	if err == nil {
		nonce = resp.Nonce
	}

	return
}

func (e *Exchange) transferToken(target proto.AccountAddress, nonce cinter.AccountNonce, amount uint64) (txHash hash.Hash, err error) {
	var (
		req  = new(ctypes.AddTxReq)
		resp = new(ctypes.AddTxResp)
		tx   = ctypes.NewTransfer(&ctypes.TransferHeader{
			Sender:    e.vaultAddress,
			Receiver:  target,
			Nonce:     nonce,
			TokenType: e.cfg.TokenType,
			Amount:    amount,
		})
	)

	if err = tx.Sign(e.vaultPrivateKey); err != nil {
		return
	}

	req.Tx = tx
	txHash = tx.Hash()
	err = mux.RequestBP(route.MCCAddTx.String(), req, resp)

	return
}
