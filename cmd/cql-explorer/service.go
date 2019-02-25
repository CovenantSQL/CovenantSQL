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

package main

import (
	"bytes"
	"encoding/binary"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"

	pi "github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	pt "github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

const (
	dbFileName = "explorer.db"
)

var (
	// storage keys
	blockKeyPrefix    = []byte("BLOCK_")
	blockHashPrefix   = []byte("HASH_")
	blockHeightPrefix = []byte("HEIGHT_")
	txKeyPrefix       = []byte("TX_")
)

// Service defines the main chain explorer service structure.
type Service struct {
	db *leveldb.DB

	caller    *rpc.Caller
	stopped   int32
	stopCh    chan struct{}
	triggerCh chan struct{}
	wg        sync.WaitGroup

	// new block check interval
	checkInterval time.Duration

	// next block to fetch
	nextBlockToFetch uint32
}

// NewService creates new explorer service handler.
func NewService(checkInterval time.Duration) (service *Service, err error) {
	// open explorer database
	dbFile := filepath.Join(conf.GConf.WorkingRoot, dbFileName)
	db, err := leveldb.OpenFile(dbFile, nil)
	if err != nil {
		return
	}

	defer func() {
		if err != nil {
			db.Close()
		}
	}()

	// init service
	service = &Service{
		db:            db,
		caller:        rpc.NewCaller(),
		stopCh:        make(chan struct{}),
		triggerCh:     make(chan struct{}, 1),
		checkInterval: checkInterval,
	}

	return
}

func (s *Service) start() (err error) {
	if atomic.LoadInt32(&s.stopped) == 1 {
		// stopped
		return ErrStopped
	}

	if err = s.getSubscriptionCheckpoint(); err != nil {
		return
	}

	// start subscription worker
	s.wg.Add(1)
	go s.subscriptionWorker()

	return
}

func (s *Service) getBlockByCount(c uint32) (b *pt.BPBlock, count uint32, height uint32, err error) {
	var bKey []byte
	bKey = append(bKey, blockKeyPrefix...)
	bKey = append(bKey, uint32ToBytes(c)...)

	it := s.db.NewIterator(util.BytesPrefix(bKey), nil)
	if it.First() {
		// decode
		bKeyLen := len(bKey)
		hBytes := it.Key()[bKeyLen : bKeyLen+4]
		height = bytesToUint32(hBytes)
		count = c
		err = utils.DecodeMsgPack(it.Value(), &b)
	} else {
		// not found
		err = ErrNotFound
	}
	it.Release()

	if err != nil {
		// ignore iterator error
		it.Error()
		return
	}

	err = it.Error()

	return
}

func (s *Service) getBlockByHash(h *hash.Hash) (b *pt.BPBlock, count uint32, height uint32, err error) {
	if h == nil {
		err = ErrNotFound
		return
	}

	var bKey []byte
	bKey = append(bKey, blockHashPrefix...)
	bKey = append(bKey, h[:]...)

	var bCountData []byte
	if bCountData, err = s.db.Get(bKey, nil); err != nil {
		if err == leveldb.ErrNotFound {
			err = ErrNotFound
		}
		return
	}

	count = bytesToUint32(bCountData)
	return s.getBlockByCount(count)
}

func (s *Service) getBlockByHeight(h uint32) (b *pt.BPBlock, count uint32, height uint32, err error) {
	var bKey []byte
	bKey = append(bKey, blockHeightPrefix...)
	bKey = append(bKey, uint32ToBytes(h)...)

	var bCountData []byte
	if bCountData, err = s.db.Get(bKey, nil); err != nil {
		if err == leveldb.ErrNotFound {
			err = ErrNotFound
		}
		return
	}

	count = bytesToUint32(bCountData)
	return s.getBlockByCount(count)
}

func (s *Service) getTxByHash(h *hash.Hash) (tx pi.Transaction, c uint32, height uint32, err error) {
	if h == nil {
		err = ErrNotFound
		return
	}

	var txKey []byte
	txKey = append(txKey, txKeyPrefix...)
	txKey = append(txKey, h[:]...)

	var bCountData []byte
	if bCountData, err = s.db.Get(txKeyPrefix, nil); err != nil {
		if err == leveldb.ErrNotFound {
			err = ErrNotFound
		}
		return
	}

	c = bytesToUint32(bCountData)

	var b *pt.BPBlock
	if b, _, height, err = s.getBlockByCount(c); err != nil {
		return
	}

	if b == nil || b.Transactions == nil {
		err = ErrNotFound
		return
	}

	for _, curTx := range b.Transactions {
		if curTx == nil {
			continue
		}

		if curH := curTx.Hash(); h.IsEqual(&curH) {
			tx = curTx
			break
		}
	}

	if tx == nil {
		err = ErrNotFound
		return
	}

	return
}

func (s *Service) getHighestCount() (c uint32, err error) {
	// load previous committed counts
	it := s.db.NewIterator(util.BytesPrefix(blockKeyPrefix), nil)
	if it.Last() {
		// decode block count from key
		blockKey := it.Key()
		prefixLen := len(blockKeyPrefix)
		c = bytesToUint32(blockKey[prefixLen : prefixLen+4])
	} else {
		err = ErrNotFound
	}
	it.Release()

	if err != nil {
		it.Error()
		return
	}

	err = it.Error()

	return
}

func (s *Service) getSubscriptionCheckpoint() (err error) {
	var lastBlockCount uint32
	if lastBlockCount, err = s.getHighestCount(); err != nil {
		log.WithError(err).Warning("get last block failed")

		if err == ErrNotFound {
			// not found, set last block count to 0
			log.Info("set current block count fetch head to 0")
			err = nil
			atomic.StoreUint32(&s.nextBlockToFetch, 0)
		}

		return
	}

	log.WithFields(log.Fields{
		"count": lastBlockCount,
	}).Info("fetched last block count")

	atomic.StoreUint32(&s.nextBlockToFetch, lastBlockCount+1)

	return
}

func (s *Service) subscriptionWorker() {
	defer s.wg.Done()

	log.Info("started subscription worker")
	for {
		select {
		case <-s.stopCh:
			log.Info("exited subscription worker")
			return
		case <-s.triggerCh:
		case <-time.After(s.checkInterval):
		}

		// request block producer for next block
		s.requestBlock()
	}
}

func (s *Service) requestBlock() {
	if atomic.LoadInt32(&s.stopped) == 1 {
		return
	}

	blockCount := atomic.LoadUint32(&s.nextBlockToFetch)
	log.WithFields(log.Fields{"count": blockCount}).Info("try fetch next block")

	req := &pt.FetchBlockByCountReq{Count: blockCount}
	resp := &pt.FetchBlockResp{}

	if err := s.requestBP(route.MCCFetchBlockByCount.String(), req, resp); err != nil {
		// fetch block failed
		log.WithError(err).Warning("fetch block failed，wait for next round")
		return
	}

	// process block
	if err := s.processBlock(blockCount, resp.Height, resp.Block); err != nil {
		log.WithError(err).Warning("process block failed, try fetch/process again")
		return
	}

	atomic.AddUint32(&s.nextBlockToFetch, 1)

	// last fetch success, trigger next fetch for fast sync
	select {
	case s.triggerCh <- struct{}{}:
	default:
	}
}

func (s *Service) processBlock(c uint32, h uint32, b *pt.BPBlock) (err error) {
	if b == nil {
		log.WithField("count", c).Warning("processed nil block")
		return ErrNilBlock
	}

	log.WithFields(log.Fields{
		"hash":   b.BlockHash(),
		"parent": b.ParentHash(),
		"height": h,
		"count":  c,
	}).Info("process new block")

	if err = s.saveTransactions(c, b.Transactions); err != nil {
		return
	}

	err = s.saveBlock(c, h, b)

	return
}

func (s *Service) saveTransactions(c uint32, txs []pi.Transaction) (err error) {
	if txs == nil || len(txs) == 0 {
		return
	}

	for _, t := range txs {
		if err = s.saveTransaction(c, t); err != nil {
			return
		}
	}

	return
}

func (s *Service) saveTransaction(c uint32, tx pi.Transaction) (err error) {
	if tx == nil {
		return ErrNilTransaction
	}

	txHash := tx.Hash()

	var txKey []byte

	txKey = append(txKey, txKeyPrefix...)
	txKey = append(txKey, txHash[:]...)
	txData := uint32ToBytes(c)

	err = s.db.Put(txKey, txData, nil)

	return
}

func (s *Service) saveBlock(c uint32, h uint32, b *pt.BPBlock) (err error) {
	if b == nil {
		return ErrNilBlock
	}

	bHash := b.BlockHash()

	var buf *bytes.Buffer

	if buf, err = utils.EncodeMsgPack(b); err != nil {
		return
	}

	cBytes := uint32ToBytes(c)
	hBytes := uint32ToBytes(h)

	var bKey, bHashKey, bHeightKey []byte

	bKey = append(bKey, blockKeyPrefix...)
	bKey = append(bKey, cBytes...)
	bKey = append(bKey, hBytes...)

	bHashKey = append(bHashKey, blockHashPrefix...)
	bHashKey = append(bHashKey, bHash[:]...)

	bHeightKey = append(bHeightKey, blockHeightPrefix...)
	bHeightKey = append(bHeightKey, hBytes...)

	if err = s.db.Put(bKey, buf.Bytes(), nil); err != nil {
		return
	}

	if err = s.db.Put(bHashKey, cBytes, nil); err != nil {
		return
	}

	err = s.db.Put(bHeightKey, cBytes, nil)

	return
}

func (s *Service) requestBP(method string, request interface{}, response interface{}) (err error) {
	var bpNodeID proto.NodeID
	if bpNodeID, err = rpc.GetCurrentBP(); err != nil {
		return
	}
	return s.caller.CallNode(bpNodeID, method, request, response)
}

func (s *Service) stop() (err error) {
	if !atomic.CompareAndSwapInt32(&s.stopped, 0, 1) {
		// stopped
		return ErrStopped
	}

	log.Info("stop subscription")

	select {
	case <-s.stopCh:
	default:
		close(s.stopCh)
	}

	s.wg.Wait()
	s.db.Close()

	return
}

func uint32ToBytes(h uint32) (data []byte) {
	data = make([]byte, 4)
	binary.BigEndian.PutUint32(data, h)
	return
}

func bytesToUint32(data []byte) uint32 {
	return binary.BigEndian.Uint32(data)
}
