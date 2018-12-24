/*
 *  Copyright 2018 The CovenantSQL Authors.
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
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
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	"github.com/CovenantSQL/CovenantSQL/chainbus"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

type BusService struct {
	chainbus.Bus

	caller *rpc.Caller

	lock   sync.Mutex // a lock for the map
	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc

	checkInterval time.Duration
	blockCount    uint32
}

func NewBusService(ctx context.Context, checkInterval time.Duration) *BusService {
	ctd, ccl := context.WithCancel(ctx)
	bs := &BusService{
		Bus:           chainbus.New(),
		lock:          sync.Mutex{},
		wg:            sync.WaitGroup{},
		caller:        rpc.NewCaller(),
		ctx:           ctd,
		cancel:        ccl,
		checkInterval: checkInterval,
	}
	return bs
}

func (bs *BusService) subscribeBlock(ctx context.Context) {
	defer bs.wg.Done()

	log.Info("start to subscribe blocks")
	for {
		select {
		case <-ctx.Done():
			log.Info("exit subscription service")
			return
		case <-time.After(bs.checkInterval):
			// fetch block from remote block producer
			c := atomic.LoadUint32(&bs.blockCount)
			log.Debugf("fetch block in count: %d", c)
			b, newCount := bs.requestLastBlock()
			if b == nil {
				continue
			}
			if newCount == c {
				continue
			}
			log.Debugf("success fetch block in count: %d, block hash: %s, number of block tx: %d",
				c, b.BlockHash().String(), len(b.GetTxHashes()))
			bs.extractTxs(b, c)
			atomic.StoreUint32(&bs.blockCount, newCount)
		}
	}

}

func (bs *BusService) requestLastBlock() (block *types.BPBlock, count uint32) {
	req := &types.FetchLastBlockReq{}
	resp := &types.FetchBlockResp{}

	if err := bs.requestBP(route.MCCFetchLastIrreversibleBlock.String(), req, resp); err != nil {
		log.WithError(err).Warning("fetch last block failed")
		return
	}

	block = resp.Block
	count = resp.Count
	return
}

func (bs *BusService) RequestSQLProfile(dbid *proto.DatabaseID) (p *types.SQLChainProfile, err error) {
	req := &types.QuerySQLChainProfileReq{DBID: *dbid}
	resp := &types.QuerySQLChainProfileResp{}
	if err = bs.requestBP(route.MCCQuerySQLChainProfile.String(), req, resp); err != nil {
		log.WithError(err).Warning("fetch sqlchain profile failed")
		return
	}

	p = &resp.Profile
	return
}

func (bs *BusService) requestBP(method string, request interface{}, response interface{}) (err error) {
	var bpNodeID proto.NodeID
	if bpNodeID, err = rpc.GetCurrentBP(); err != nil {
		return
	}
	return bs.caller.CallNode(bpNodeID, method, request, response)
}

func (bs *BusService) extractTxs(blocks *types.BPBlock, count uint32) {
	for _, tx := range blocks.Transactions {
		t := bs.unwrapTx(tx)
		eventName := fmt.Sprintf("/%s/", t.GetTransactionType().String())
		bs.Publish(eventName, t, count)
	}
}

func (bs *BusService) unwrapTx(tx interfaces.Transaction) interfaces.Transaction {
	switch t := tx.(type) {
	case *interfaces.TransactionWrapper:
		return bs.unwrapTx(t.Unwrap())
	default:
		return tx
	}
}

// Start starts a chain bus service.
func (bs *BusService) Start() {
	bs.wg.Add(1)
	go bs.subscribeBlock(bs.ctx)
}

// Stop stops the chain bus service.
func (bs *BusService) Stop() {
	bs.cancel()
	bs.wg.Wait()
}
