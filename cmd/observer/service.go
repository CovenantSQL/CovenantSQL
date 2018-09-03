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
	"encoding/binary"
	"errors"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"bytes"

	bp "github.com/CovenantSQL/CovenantSQL/blockproducer"
	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	"github.com/CovenantSQL/CovenantSQL/sqlchain"
	ct "github.com/CovenantSQL/CovenantSQL/sqlchain/types"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	wt "github.com/CovenantSQL/CovenantSQL/worker/types"
	"github.com/coreos/bbolt"
)

const (
	dbFileName = "observer.db"
)

// Bucket stores transaction/block information as follows
/*
[root]
  |
  |--[height]-->[`dbID`]
  |    |          |---> [hash] => height
  |    |           \--> [hash] => height
  |    |
  |    |
  |  [block]-->[`dbID`]
  |    |          |---> [height+hash] => block
  |    |           \--> [height+hash] => block
  |    |
  |  [ack]-->[`dbID`]
  |    |	    |---> [hash] => ack
  |    |         \--> [hash] => ack
  |    |
  |  [request]-->[`dbID`]
  |    |            |---> [offset+hash] => request
  |    |             \--> [offset+hash] => request
  |    |
  |  [offset]-->[`dbID`]
  |                 |---> [hash] => offset
  |                  \--> [hash] => offset
  |
   \-> [subscription]
             \---> [`dbID`] => height
*/

var (
	// ErrStopped defines error on observer service has already stopped
	ErrStopped = errors.New("observer service has stopped")
	// ErrNotFound defines error on fail to found specified resource
	ErrNotFound = errors.New("resource not found")

	// bolt db buckets
	blockBucket        = []byte("block")
	ackBucket          = []byte("ack")
	requestBucket      = []byte("request")
	subscriptionBucket = []byte("subscription")

	blockHeightBucket = []byte("height")
	logOffsetBucket   = []byte("offset")

	// blockProducePeriod defines the block producing interval
	blockProducePeriod = 60 * time.Second
)

// Service defines the observer service structure.
type Service struct {
	lock            sync.Mutex
	subscription    map[proto.DatabaseID]int32
	upstreamServers sync.Map

	db      *bolt.DB
	caller  *rpc.Caller
	stopped int32
}

// NewService creates new observer service and load previous subscription from the meta database.
func NewService() (service *Service, err error) {
	// open observer database
	dbFile := filepath.Join(conf.GConf.WorkingRoot, dbFileName)

	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		return
	}

	defer func() {
		if err != nil {
			db.Close()
		}
	}()

	if err = db.Update(func(tx *bolt.Tx) (err error) {
		if _, err = tx.CreateBucketIfNotExists(blockBucket); err != nil {
			return
		}
		if _, err = tx.CreateBucketIfNotExists(ackBucket); err != nil {
			return
		}
		if _, err = tx.CreateBucketIfNotExists(requestBucket); err != nil {
			return
		}
		if _, err = tx.CreateBucketIfNotExists(subscriptionBucket); err != nil {
			return
		}
		if _, err = tx.CreateBucketIfNotExists(blockHeightBucket); err != nil {
			return
		}
		_, err = tx.CreateBucketIfNotExists(logOffsetBucket)
		return
	}); err != nil {
		return
	}

	// init service
	service = &Service{
		subscription: make(map[proto.DatabaseID]int32),
		db:           db,
		caller:       rpc.NewCaller(),
	}

	// load previous subscriptions
	if err = db.View(func(tx *bolt.Tx) error {
		return tx.Bucket(subscriptionBucket).ForEach(func(rawDBID, rawHeight []byte) (err error) {
			dbID := proto.DatabaseID(string(rawDBID))
			h := bytesToHeight(rawHeight)
			service.subscription[dbID] = h
			return
		})
	}); err != nil {
		return
	}

	return
}

func offsetToBytes(offset uint64) (data []byte) {
	data = make([]byte, 8)
	binary.BigEndian.PutUint64(data, offset)
	return
}

func bytesToOffset(data []byte) uint64 {
	return uint64(binary.BigEndian.Uint64(data))
}

func heightToBytes(h int32) (data []byte) {
	data = make([]byte, 4)
	binary.BigEndian.PutUint32(data, uint32(h))
	return
}

func bytesToHeight(data []byte) int32 {
	return int32(binary.BigEndian.Uint32(data))
}

func (s *Service) subscribe(dbID proto.DatabaseID, resetSubscribePosition string) (err error) {
	if atomic.LoadInt32(&s.stopped) == 1 {
		return ErrStopped
	}

	s.lock.Lock()

	shouldStartSubscribe := false

	if resetSubscribePosition != "" {
		var fromPos int32

		switch resetSubscribePosition {
		case "newest":
			fromPos = ct.ReplicateFromNewest
		case "oldest":
			fromPos = ct.ReplicateFromBeginning
		default:
			fromPos = ct.ReplicateFromNewest
		}

		s.subscription[dbID] = fromPos

		// send start subscription request
		shouldStartSubscribe = true
	} else {
		// not resetting
		if _, exists := s.subscription[dbID]; !exists {
			s.subscription[dbID] = ct.ReplicateFromNewest
			shouldStartSubscribe = true
		}
	}

	s.lock.Unlock()

	if shouldStartSubscribe {
		return s.startSubscribe(dbID)
	}

	return
}

// AdviseNewBlock handles block replication request from the remote database chain service.
func (s *Service) AdviseNewBlock(req *sqlchain.MuxAdviseNewBlockReq, resp *sqlchain.MuxAdviseNewBlockResp) (err error) {
	if atomic.LoadInt32(&s.stopped) == 1 {
		// stopped
		return ErrStopped
	}

	if req.Block == nil {
		log.Infof("received empty block from node %v", req.GetNodeID().String())
		return
	}

	return s.addBlock(req.DatabaseID, req.Block)
}

// AdviseAckedQuery handles acked query replication request from the remote database chain service.
func (s *Service) AdviseAckedQuery(req *sqlchain.MuxAdviseAckedQueryReq, resp *sqlchain.MuxAdviseAckedQueryResp) (err error) {
	if atomic.LoadInt32(&s.stopped) == 1 {
		// stopped
		return ErrStopped
	}

	if req.Query == nil {
		log.Infof("received empty acked query from node %v", req.GetNodeID().String())
		return
	}

	return s.addAckedQuery(req.DatabaseID, req.Query)
}

func (s *Service) start() (err error) {
	if atomic.LoadInt32(&s.stopped) == 1 {
		// stopped
		return ErrStopped
	}

	s.lock.Lock()
	dbs := make([]proto.DatabaseID, len(s.subscription))
	for dbID := range s.subscription {
		dbs = append(dbs, dbID)
	}
	s.lock.Unlock()

	for _, dbID := range dbs {
		if err = s.startSubscribe(dbID); err != nil {
			log.Warningf("start subscription failed on database %v: %v", dbID, err)
		}
	}

	return nil
}

func (s *Service) startSubscribe(dbID proto.DatabaseID) (err error) {
	if atomic.LoadInt32(&s.stopped) == 1 {
		// stopped
		return ErrStopped
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	// start subscribe on first node of each sqlchain server peers
	log.Infof("start subscribing transactions from database %v", dbID)

	instance, err := s.getUpstream(dbID)
	if err != nil {
		return
	}

	// store the genesis block
	if err = s.addBlock(dbID, instance.GenesisBlock); err != nil {
		return
	}

	req := &sqlchain.MuxSubscribeTransactionsReq{}
	resp := &sqlchain.MuxSubscribeTransactionsResp{}
	req.Height = s.subscription[dbID]
	req.DatabaseID = dbID

	err = s.minerRequest(dbID, route.SQLCSubscribeTransactions.String(), req, resp)

	return
}

func (s *Service) addAckedQuery(dbID proto.DatabaseID, ack *wt.SignedAckHeader) (err error) {
	log.Debugf("add ack query %v: %v", dbID, ack.HeaderHash.String())

	if atomic.LoadInt32(&s.stopped) == 1 {
		// stopped
		return ErrStopped
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	if err = ack.Verify(); err != nil {
		return
	}

	// fetch original query
	if ack.Response.Request.QueryType == wt.WriteQuery {
		req := &wt.GetRequestReq{}
		resp := &wt.GetRequestResp{}

		req.DatabaseID = dbID
		req.LogOffset = ack.Response.LogOffset

		if err = s.minerRequest(dbID, route.DBSGetRequest.String(), req, resp); err != nil {
			return
		}

		key := offsetToBytes(req.LogOffset)
		key = append(key, resp.Request.Header.HeaderHash.CloneBytes()...)

		log.Debugf("add write request, offset: %v, %v, %v",
			req.LogOffset, resp.Request.Header.HeaderHash.String(), resp.Request.Payload.Queries)

		var reqBytes *bytes.Buffer
		if reqBytes, err = utils.EncodeMsgPack(resp.Request); err != nil {
			return
		}

		if err = s.db.Update(func(tx *bolt.Tx) (err error) {
			qb, err := tx.Bucket(requestBucket).CreateBucketIfNotExists([]byte(dbID))
			if err != nil {
				return
			}
			if err = qb.Put(key, reqBytes.Bytes()); err != nil {
				return
			}
			ob, err := tx.Bucket(logOffsetBucket).CreateBucketIfNotExists([]byte(dbID))
			if err != nil {
				return
			}
			err = ob.Put(resp.Request.Header.HeaderHash.CloneBytes(), offsetToBytes(req.LogOffset))
			return
		}); err != nil {
			return
		}
	}

	// store ack
	return s.db.Update(func(tx *bolt.Tx) (err error) {
		ab, err := tx.Bucket(ackBucket).CreateBucketIfNotExists([]byte(dbID))
		if err != nil {
			return
		}
		ackBytes, err := utils.EncodeMsgPack(ack)
		if err != nil {
			return
		}
		err = ab.Put(ack.HeaderHash.CloneBytes(), ackBytes.Bytes())
		return
	})
}

func (s *Service) addBlock(dbID proto.DatabaseID, b *ct.Block) (err error) {
	log.Debugf("add block %v, %v -> %v, %v", dbID, b.BlockHash(), b.ParentHash(), b.Producer())

	instance, err := s.getUpstream(dbID)
	h := int32(b.Timestamp().Sub(instance.GenesisBlock.Timestamp()) / blockProducePeriod)
	key := heightToBytes(h)
	key = append(key, b.BlockHash().CloneBytes()...)

	return s.db.Update(func(tx *bolt.Tx) (err error) {
		bb, err := tx.Bucket(blockBucket).CreateBucketIfNotExists([]byte(dbID))
		if err != nil {
			return
		}
		blockBytes, err := utils.EncodeMsgPack(b)
		if err != nil {
			return
		}
		if err = bb.Put(key, blockBytes.Bytes()); err != nil {
			return
		}
		hb, err := tx.Bucket(blockHeightBucket).CreateBucketIfNotExists([]byte(dbID))
		if err != nil {
			return
		}
		err = hb.Put(b.BlockHash()[:], heightToBytes(h))
		return
	})
}

func (s *Service) stop() (err error) {
	if atomic.LoadInt32(&s.stopped) == 1 {
		// stopped
		return ErrStopped
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	atomic.StoreInt32(&s.stopped, 1)

	// send cancel subscription to all databases
	log.Infof("stop subscribing all databases")

	for dbID := range s.subscription {
		// send cancel subscription rpc
		req := &sqlchain.MuxCancelSubscriptionReq{}
		resp := &sqlchain.MuxCancelSubscriptionResp{}
		req.DatabaseID = dbID

		if err = s.minerRequest(dbID, route.SQLCCancelSubscription.String(), req, resp); err != nil {
			// cancel subscription failed
			log.Warningf("cancel subscription for database %v failed: %v", dbID, err)
		}
	}

	// close the subscription database
	s.db.Close()

	return
}

func (s *Service) minerRequest(dbID proto.DatabaseID, method string, request interface{}, response interface{}) (err error) {
	instance, err := s.getUpstream(dbID)
	if err != nil {
		return
	}

	return s.caller.CallNode(instance.Peers.Leader.ID, method, request, response)
}

func (s *Service) getUpstream(dbID proto.DatabaseID) (instance *wt.ServiceInstance, err error) {
	log.Infof("get peers info for database: %v", dbID)

	if iInstance, exists := s.upstreamServers.Load(dbID); exists {
		instance = iInstance.(*wt.ServiceInstance)
		return
	}

	curBP, err := rpc.GetCurrentBP()
	if err != nil {
		return
	}

	privateKey, err := kms.GetLocalPrivateKey()
	if err != nil {
		return
	}

	pubKey, err := kms.GetLocalPublicKey()
	if err != nil {
		return
	}

	req := &bp.GetDatabaseRequest{}
	req.Header.DatabaseID = dbID
	req.Header.Signee = pubKey
	if err = req.Sign(privateKey); err != nil {
		return
	}
	resp := &bp.GetDatabaseResponse{}
	// get peers list from block producer
	if err = s.caller.CallNode(curBP, route.BPDBGetDatabase.String(), req, resp); err != nil {
		return
	}
	if err = resp.Verify(); err != nil {
		return
	}

	instance = &resp.Header.InstanceMeta
	s.upstreamServers.Store(dbID, instance)

	return
}

func (s *Service) getAck(dbID proto.DatabaseID, h *hash.Hash) (ack *wt.SignedAckHeader, err error) {
	err = s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(ackBucket).Bucket([]byte(dbID))

		if bucket == nil {
			return ErrNotFound
		}

		ackBytes := bucket.Get(h.CloneBytes())
		if ackBytes == nil {
			return ErrNotFound
		}

		return utils.DecodeMsgPack(ackBytes, &ack)
	})

	return
}

func (s *Service) getRequest(dbID proto.DatabaseID, h *hash.Hash) (request *wt.Request, err error) {
	err = s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(logOffsetBucket).Bucket([]byte(dbID))

		if bucket == nil {
			return ErrNotFound
		}

		reqKey := bucket.Get(h.CloneBytes())
		if reqKey == nil {
			return ErrNotFound
		}

		reqKey = append([]byte{}, reqKey...)
		reqKey = append(reqKey, h.CloneBytes()...)

		bucket = tx.Bucket(requestBucket).Bucket([]byte(dbID))
		if bucket == nil {
			return ErrNotFound
		}

		reqBytes := bucket.Get(reqKey)
		if reqBytes == nil {
			return ErrNotFound
		}

		return utils.DecodeMsgPack(reqBytes, &request)
	})

	return
}

func (s *Service) getRequestByOffset(dbID proto.DatabaseID, offset uint64) (request *wt.Request, err error) {
	err = s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(requestBucket).Bucket([]byte(dbID))

		if bucket == nil {
			return ErrNotFound
		}

		keyPrefix := offsetToBytes(offset)
		cur := bucket.Cursor()

		for k, v := cur.Seek(keyPrefix); k != nil && bytes.HasPrefix(k, keyPrefix); k, v = cur.Next() {
			if v != nil {
				return utils.DecodeMsgPack(v, &request)
			}
		}

		return ErrNotFound
	})

	return
}

func (s *Service) getBlockByHeight(dbID proto.DatabaseID, height int32) (b *ct.Block, err error) {
	err = s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(blockBucket).Bucket([]byte(dbID))

		if bucket == nil {
			return ErrNotFound
		}

		keyPrefix := heightToBytes(height)

		cur := bucket.Cursor()
		for k, v := cur.Seek(keyPrefix); k != nil && bytes.HasPrefix(k, keyPrefix); k, v = cur.Next() {
			if v != nil {
				return utils.DecodeMsgPack(v, &b)
			}
		}

		return ErrNotFound
	})

	return
}

func (s *Service) getBlock(dbID proto.DatabaseID, h *hash.Hash) (height int32, b *ct.Block, err error) {
	err = s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(blockHeightBucket).Bucket([]byte(dbID))

		if bucket == nil {
			return ErrNotFound
		}

		blockKey := bucket.Get(h.CloneBytes())
		if blockKey == nil {
			return ErrNotFound
		}

		blockKey = append([]byte{}, blockKey...)
		blockKey = append(blockKey, h.CloneBytes()...)

		bucket = tx.Bucket(blockBucket).Bucket([]byte(dbID))
		if bucket == nil {
			return ErrNotFound
		}

		blockBytes := bucket.Get(blockKey)
		if blockBytes == nil {
			return ErrNotFound
		}

		return utils.DecodeMsgPack(blockBytes, &b)
	})

	if err == nil {
		// compute height
		var instance *wt.ServiceInstance
		instance, err = s.getUpstream(dbID)
		if err != nil {
			return
		}

		height = int32(b.Timestamp().Sub(instance.GenesisBlock.Timestamp()) / blockProducePeriod)
	}

	return
}
