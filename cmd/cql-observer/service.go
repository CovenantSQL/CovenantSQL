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
	"errors"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	bolt "github.com/coreos/bbolt"
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
  |  [count2height]-->[`dbID`]
  |    |                 |---> [count] => height
  |    |                  \--> [count] => height
  |    |
  |  [block]-->[`dbID`]
  |    |          |---> [height+hash+count] => block
  |    |           \--> [height+hash+count] => block
  |    |
  |  [ack]-->[`dbID`]
  |    |	    |---> [hash] => height+offset
  |    |         \--> [hash] => height+offset
  |    |
  |  [request]-->[`dbID`]
  |    |            |---> [hash] => height+offset
  |    |             \--> [hash] => height+offset
  |    |
  |  [response]-->[`dbID`]
  |                 |---> [hash] => height+offset
  |                  \--> [hash] => height+offset
  |
   \-> [subscription]
             \---> [`dbID`] => height
*/

var (
	// ErrStopped defines error on observer service has already stopped
	ErrStopped = errors.New("observer service has stopped")
	// ErrNotFound defines error on fail to found specified resource
	ErrNotFound = errors.New("resource not found")
	// ErrInconsistentData represents corrupted observation data.
	ErrInconsistentData = errors.New("inconsistent data")

	// bolt db buckets
	blockBucket             = []byte("block")
	blockCount2HeightBucket = []byte("block-count-to-height")
	ackBucket               = []byte("ack")
	requestBucket           = []byte("request")
	responseBucket          = []byte("response")
	subscriptionBucket      = []byte("subscription")

	blockHeightBucket = []byte("height")
)

// Service defines the observer service structure.
type Service struct {
	subscription    sync.Map // map[proto.DatabaseID]*subscribeWorker
	upstreamServers sync.Map // map[proto.DatabaseID]*types.ServiceInstance

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
		if _, err = tx.CreateBucketIfNotExists(blockCount2HeightBucket); err != nil {
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
		_, err = tx.CreateBucketIfNotExists(responseBucket)
		return
	}); err != nil {
		return
	}

	// init service
	service = &Service{
		db:     db,
		caller: rpc.NewCaller(),
	}

	// load previous subscriptions
	if err = db.View(func(tx *bolt.Tx) error {
		return tx.Bucket(subscriptionBucket).ForEach(func(rawDBID, rawCount []byte) (err error) {
			dbID := proto.DatabaseID(string(rawDBID))
			count := bytesToInt32(rawCount)
			service.subscription.Store(dbID, newSubscribeWorker(dbID, count, service))
			return
		})
	}); err != nil {
		return
	}

	return
}

func int32ToBytes(h int32) (data []byte) {
	data = make([]byte, 4)
	binary.BigEndian.PutUint32(data, uint32(h))
	return
}

func bytesToInt32(data []byte) int32 {
	return int32(binary.BigEndian.Uint32(data))
}

func (s *Service) subscribe(dbID proto.DatabaseID, resetSubscribePosition string) (err error) {
	if atomic.LoadInt32(&s.stopped) == 1 {
		return ErrStopped
	}

	if resetSubscribePosition != "" {
		var fromPos int32

		switch resetSubscribePosition {
		case "newest":
			fromPos = types.ReplicateFromNewest
		case "oldest":
			fromPos = types.ReplicateFromBeginning
		default:
			fromPos = types.ReplicateFromNewest
		}

		unpackWorker(s.subscription.LoadOrStore(dbID,
			newSubscribeWorker(dbID, fromPos, s))).reset(fromPos)
	} else {
		// not resetting
		unpackWorker(s.subscription.LoadOrStore(dbID,
			newSubscribeWorker(dbID, types.ReplicateFromNewest, s))).start()
	}

	return
}

func unpackWorker(actual interface{}, _ ...interface{}) (worker *subscribeWorker) {
	if actual == nil {
		return
	}

	worker, _ = actual.(*subscribeWorker)

	return
}

func (s *Service) start() (err error) {
	if atomic.LoadInt32(&s.stopped) == 1 {
		// stopped
		return ErrStopped
	}

	s.subscription.Range(func(_, rawWorker interface{}) bool {
		unpackWorker(rawWorker).start()
		return true
	})

	return nil
}

func (s *Service) addAck(dbID proto.DatabaseID, height int32, offset int32, ack *types.SignedAckHeader) (err error) {
	log.WithFields(log.Fields{
		"height": height,
		"ack":    ack.Hash().String(),
		"db":     dbID,
	}).Debug("add ack")

	if atomic.LoadInt32(&s.stopped) == 1 {
		// stopped
		return ErrStopped
	}

	if err = ack.Verify(); err != nil {
		return
	}

	// store ack
	return s.db.Update(func(tx *bolt.Tx) (err error) {
		ab, err := tx.Bucket(ackBucket).CreateBucketIfNotExists([]byte(dbID))
		if err != nil {
			return
		}
		err = ab.Put(ack.Hash().AsBytes(), utils.ConcatAll(int32ToBytes(height), int32ToBytes(offset)))
		return
	})
}

func (s *Service) addQueryTracker(dbID proto.DatabaseID, height int32, offset int32, qt *types.QueryAsTx) (err error) {
	log.WithFields(log.Fields{
		"req":  qt.Request.Header.Hash(),
		"resp": qt.Response.Hash(),
	}).Debug("add query tracker")

	if atomic.LoadInt32(&s.stopped) == 1 {
		// stopped
		return ErrStopped
	}

	if err = qt.Request.Verify(); err != nil {
		return
	}
	if err = qt.Response.VerifyHash(); err != nil {
		return
	}

	dataBytes := utils.ConcatAll(int32ToBytes(height), int32ToBytes(offset))

	// store request and response
	return s.db.Update(func(tx *bolt.Tx) (err error) {
		reqb, err := tx.Bucket(requestBucket).CreateBucketIfNotExists([]byte(dbID))
		if err != nil {
			return
		}
		resb, err := tx.Bucket(responseBucket).CreateBucketIfNotExists([]byte(dbID))
		if err != nil {
			return
		}
		if err = reqb.Put(qt.Request.Header.Hash().AsBytes(), dataBytes); err != nil {
			return
		}
		err = resb.Put(qt.Response.Hash().AsBytes(), dataBytes)
		return
	})
}

func (s *Service) addBlock(dbID proto.DatabaseID, count int32, b *types.Block) (err error) {
	instance, err := s.getUpstream(dbID)
	h := int32(b.Timestamp().Sub(instance.GenesisBlock.Timestamp()) / conf.GConf.SQLChainPeriod)
	key := utils.ConcatAll(int32ToBytes(h), b.BlockHash().AsBytes(), int32ToBytes(count))
	// It's actually `countToBytes`
	ckey := int32ToBytes(count)
	blockBytes, err := utils.EncodeMsgPack(b)
	if err != nil {
		return
	}
	log.WithFields(log.Fields{
		"database": dbID,
		"count":    count,
		"height":   h,
		"producer": b.Producer(),
		"block":    b,
	}).Debugf("add new block %v -> %v", b.BlockHash(), b.ParentHash())

	if err = s.db.Update(func(tx *bolt.Tx) (err error) {
		bb, err := tx.Bucket(blockBucket).CreateBucketIfNotExists([]byte(dbID))
		if err != nil {
			return
		}
		if err = bb.Put(key, blockBytes.Bytes()); err != nil {
			return
		}
		cb, err := tx.Bucket(blockCount2HeightBucket).CreateBucketIfNotExists([]byte(dbID))
		if err != nil {
			return
		}
		if count >= 0 {
			if err = cb.Put(ckey, int32ToBytes(h)); err != nil {
				return
			}
		}
		hb, err := tx.Bucket(blockHeightBucket).CreateBucketIfNotExists([]byte(dbID))
		if err != nil {
			return
		}
		err = hb.Put(b.BlockHash()[:], int32ToBytes(h))
		return
	}); err != nil {
		return
	}

	// save ack
	for i, q := range b.Acks {
		if err = s.addAck(dbID, h, int32(i), q); err != nil {
			return
		}
	}

	// save queries
	for i, q := range b.QueryTxs {
		if err = s.addQueryTracker(dbID, h, int32(i), q); err != nil {
			return
		}
	}

	return
}

func (s *Service) stop() (err error) {
	if !atomic.CompareAndSwapInt32(&s.stopped, 0, 1) {
		// stopped
		return ErrStopped
	}

	// send cancel subscription to all databases
	log.Info("stop subscribing all databases")

	s.subscription.Range(func(_, rawWorker interface{}) bool {
		unpackWorker(rawWorker).stop()
		return true
	})

	// close the subscription database
	s.db.Close()

	return
}

func (s *Service) minerRequest(dbID proto.DatabaseID, method string, request interface{}, response interface{}) (err error) {
	instance, err := s.getUpstream(dbID)
	if err != nil {
		return
	}

	return s.caller.CallNode(instance.Peers.Leader, method, request, response)
}

func (s *Service) getUpstream(dbID proto.DatabaseID) (instance *types.ServiceInstance, err error) {
	log.WithField("db", dbID).Debug("get peers info for database")

	if iInstance, exists := s.upstreamServers.Load(dbID); exists {
		instance = iInstance.(*types.ServiceInstance)
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

	var (
		req = &types.QuerySQLChainProfileReq{
			DBID: dbID,
		}
		resp = &types.QuerySQLChainProfileResp{}
	)
	// get peers list from block producer
	if err = s.caller.CallNode(
		curBP, route.MCCQuerySQLChainProfile.String(), req, resp,
	); err != nil {
		return
	}

	// Build server instance from sqlchain profile
	var (
		profile = resp.Profile
		nodeids = make([]proto.NodeID, len(profile.Miners))
		peers   *proto.Peers
		genesis = &types.Block{}
	)
	for i, v := range profile.Miners {
		nodeids[i] = v.NodeID
	}
	peers = &proto.Peers{
		PeersHeader: proto.PeersHeader{
			Leader:  nodeids[0],
			Servers: nodeids[:],
		},
	}
	if err = peers.Sign(privateKey); err != nil {
		return
	}
	if err = utils.DecodeMsgPack(profile.EncodedGenesis, genesis); err != nil {
		return
	}
	instance = &types.ServiceInstance{
		DatabaseID:   profile.ID,
		Peers:        peers,
		ResourceMeta: profile.Meta,
		GenesisBlock: genesis,
	}
	s.upstreamServers.Store(dbID, instance)

	return
}

func (s *Service) getAck(dbID proto.DatabaseID, h *hash.Hash) (ack *types.SignedAckHeader, err error) {
	var (
		blockHeight int32
		dataOffset  int32
	)

	if err = s.db.View(func(tx *bolt.Tx) (err error) {
		bucket := tx.Bucket(ackBucket).Bucket([]byte(dbID))

		if bucket == nil {
			return ErrNotFound
		}

		ackBytes := bucket.Get(h.AsBytes())
		if ackBytes == nil {
			return ErrNotFound
		}

		// get block height and object offset in block
		if len(ackBytes) != 8 {
			// invalid data payload
			return ErrInconsistentData
		}

		blockHeight = bytesToInt32(ackBytes[:4])
		dataOffset = bytesToInt32(ackBytes[4:])

		return
	}); err != nil {
		return
	}

	// get data from block
	var b *types.Block
	if _, b, err = s.getBlockByHeight(dbID, blockHeight); err != nil {
		return
	}

	if dataOffset < 0 || int32(len(b.Acks)) <= dataOffset {
		err = ErrInconsistentData
		return
	}

	ack = b.Acks[int(dataOffset)]

	// verify hash
	ackHash := ack.Hash()
	if !ackHash.IsEqual(h) {
		err = ErrInconsistentData
	}

	return
}

func (s *Service) getRequest(dbID proto.DatabaseID, h *hash.Hash) (request *types.Request, err error) {
	var (
		blockHeight int32
		dataOffset  int32
	)

	if err = s.db.View(func(tx *bolt.Tx) (err error) {
		bucket := tx.Bucket(requestBucket).Bucket([]byte(dbID))
		if bucket == nil {
			return ErrNotFound
		}

		reqBytes := bucket.Get(h.AsBytes())
		if reqBytes == nil {
			return ErrNotFound
		}

		// get block height and object offset in block
		if len(reqBytes) != 8 {
			// invalid data payload
			return ErrInconsistentData
		}

		blockHeight = bytesToInt32(reqBytes[:4])
		dataOffset = bytesToInt32(reqBytes[4:])

		return
	}); err != nil {
		return
	}

	// get data from block
	var b *types.Block
	if _, b, err = s.getBlockByHeight(dbID, blockHeight); err != nil {
		return
	}

	if dataOffset < 0 || int32(len(b.QueryTxs)) <= dataOffset {
		err = ErrInconsistentData
		return
	}

	request = b.QueryTxs[int(dataOffset)].Request

	// verify hash
	reqHash := request.Header.Hash()
	if !reqHash.IsEqual(h) {
		err = ErrInconsistentData
	}

	return
}

func (s *Service) getResponseHeader(dbID proto.DatabaseID, h *hash.Hash) (response *types.SignedResponseHeader, err error) {
	var (
		blockHeight int32
		dataOffset  int32
	)

	if err = s.db.View(func(tx *bolt.Tx) (err error) {
		bucket := tx.Bucket(requestBucket).Bucket([]byte(dbID))
		if bucket == nil {
			return ErrNotFound
		}

		respBytes := bucket.Get(h.AsBytes())
		if respBytes == nil {
			return ErrNotFound
		}

		// get block height and object offset in block
		if len(respBytes) != 8 {
			// invalid data payload
			return ErrInconsistentData
		}

		blockHeight = bytesToInt32(respBytes[:4])
		dataOffset = bytesToInt32(respBytes[4:])

		return
	}); err != nil {
		return
	}

	// get data from block
	var b *types.Block
	if _, b, err = s.getBlockByHeight(dbID, blockHeight); err != nil {
		return
	}

	if dataOffset < 0 || int32(len(b.QueryTxs)) <= dataOffset {
		err = ErrInconsistentData
		return
	}

	response = b.QueryTxs[int(dataOffset)].Response

	// verify hash
	respHash := response.Hash()
	if !respHash.IsEqual(h) {
		err = ErrInconsistentData
	}

	return
}

func (s *Service) getHighestBlock(dbID proto.DatabaseID) (height int32, b *types.Block, err error) {
	err = s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(blockBucket).Bucket([]byte(dbID))

		if bucket == nil {
			return ErrNotFound
		}

		cur := bucket.Cursor()
		if last, blockData := cur.Last(); last != nil {
			// decode bytes
			height = bytesToInt32(last[:4])
			return utils.DecodeMsgPack(blockData, &b)
		}

		return ErrNotFound
	})

	return
}

func (s *Service) getHighestBlockV2(
	dbID proto.DatabaseID) (count, height int32, b *types.Block, err error,
) {
	err = s.db.View(func(tx *bolt.Tx) (err error) {
		var (
			bk         *bolt.Bucket
			cur        *bolt.Cursor
			c, h, k, v []byte
		)
		// Get last count and height
		if bk = tx.Bucket(blockCount2HeightBucket).Bucket([]byte(dbID)); bk == nil {
			return ErrNotFound
		}
		if c, h = bk.Cursor().Last(); c == nil || h == nil {
			return ErrNotFound
		}
		// Get block by height prefix
		if bk = tx.Bucket(blockBucket).Bucket([]byte(dbID)); bk == nil {
			return ErrNotFound
		}
		cur = bk.Cursor()
		for k, v = cur.Seek(h); k != nil && v != nil && bytes.HasPrefix(k, h); k, v = cur.Next() {
			if v != nil {
				if err = utils.DecodeMsgPack(v, &b); err == nil {
					count = bytesToInt32(c[:4])
					height = bytesToInt32(h[:4])
				}
				return
			}
		}
		return
	})
	return
}

func (s *Service) getBlockByHeight(dbID proto.DatabaseID, height int32) (count int32, b *types.Block, err error) {
	err = s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(blockBucket).Bucket([]byte(dbID))

		if bucket == nil {
			return ErrNotFound
		}

		keyPrefix := int32ToBytes(height)

		cur := bucket.Cursor()
		for k, v := cur.Seek(keyPrefix); k != nil && bytes.HasPrefix(k, keyPrefix); k, v = cur.Next() {
			if v != nil {
				if len(k) < 4+hash.HashSize+4 {
					return ErrInconsistentData
				}
				count = bytesToInt32(k[4+hash.HashSize:])
				return utils.DecodeMsgPack(v, &b)
			}
		}

		return ErrNotFound
	})

	return
}

func (s *Service) getBlockByCount(
	dbID proto.DatabaseID, count int32) (height int32, b *types.Block, err error,
) {
	err = s.db.View(func(tx *bolt.Tx) (err error) {
		var (
			bk      *bolt.Bucket
			cur     *bolt.Cursor
			c       = int32ToBytes(count)
			h, k, v []byte
		)
		// Get height by count
		if bk = tx.Bucket(blockCount2HeightBucket).Bucket([]byte(dbID)); bk == nil {
			return ErrNotFound
		}
		if h = bk.Get(c); h == nil {
			return ErrNotFound
		}
		// Get block by height prefix
		if bk = tx.Bucket(blockBucket).Bucket([]byte(dbID)); bk == nil {
			return ErrNotFound
		}
		cur = bk.Cursor()
		for k, v = cur.Seek(h); k != nil && v != nil && bytes.HasPrefix(k, h); k, v = cur.Next() {
			if v != nil {
				if err = utils.DecodeMsgPack(v, &b); err == nil {
					height = bytesToInt32(h[:4])
				}
				return
			}
		}
		return
	})
	return
}

func (s *Service) getBlock(dbID proto.DatabaseID, h *hash.Hash) (count int32, height int32, b *types.Block, err error) {
	err = s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(blockHeightBucket).Bucket([]byte(dbID))

		if bucket == nil {
			return ErrNotFound
		}

		blockKeyPrefix := bucket.Get(h.AsBytes())
		if blockKeyPrefix == nil {
			return ErrNotFound
		}

		blockKeyPrefix = append([]byte{}, blockKeyPrefix...)
		blockKeyPrefix = append(blockKeyPrefix, h.AsBytes()...)

		bucket = tx.Bucket(blockBucket).Bucket([]byte(dbID))
		if bucket == nil {
			return ErrNotFound
		}

		var (
			blockKey   []byte
			blockBytes []byte
		)

		cur := bucket.Cursor()
		for blockKey, blockBytes = cur.Seek(blockKeyPrefix); blockKey != nil && bytes.HasPrefix(blockKey, blockKeyPrefix); blockKey, blockBytes = cur.Next() {
			if blockBytes != nil {
				break
			}
		}

		if blockBytes == nil {
			return ErrNotFound
		}

		// decode count from block key
		if len(blockKey) < 4+hash.HashSize+4 {
			return ErrInconsistentData
		}

		height = bytesToInt32(blockKey[:4])
		count = bytesToInt32(blockKey[4+hash.HashSize:])

		return utils.DecodeMsgPack(blockBytes, &b)
	})

	return
}
