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

package observer

import (
	"database/sql"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/pkg/errors"

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	"github.com/CovenantSQL/CovenantSQL/rpc/mux"
	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	xs "github.com/CovenantSQL/CovenantSQL/xenomint/sqlite"
)

const (
	dbFileName = "observer.db3"
)

var (
	// ErrStopped defines error on observer service has already stopped
	ErrStopped = errors.New("observer service has stopped")
	// ErrNotFound defines error on fail to found specified resource
	ErrNotFound = errors.New("resource not found")
	// ErrInconsistentData represents corrupted observation data.
	ErrInconsistentData = errors.New("inconsistent data")

	// observer init table sql
	initTableSQL = []string{
		`CREATE TABLE IF NOT EXISTS "block" (
			"db"		TEXT,
			"height" 	INTEGER,
			"count" 	INTEGER,
			"hash"		TEXT,
			"block"		BLOB,
			UNIQUE("db", "hash")
		)`,
		`CREATE INDEX IF NOT EXISTS "idx_block_db_height" ON "block" ("db", "height")`,
		`CREATE INDEX IF NOT EXISTS "idx_block_db_count" ON "block" ("db", "count")`,
		`CREATE TABLE IF NOT EXISTS "ack" (
			"db"		TEXT,
			"hash"		TEXT,
			"height"	INTEGER,
			"offset"	INTEGER,
			UNIQUE("db", "hash")
		)`,
		`CREATE TABLE IF NOT EXISTS "request" (
			"db"		TEXT,
			"hash"		TEXT,
			"height"	INTEGER,
			"offset"	INTEGER,
			UNIQUE("db", "hash")
		)`,
		`CREATE TABLE IF NOT EXISTS "response" (
			"db"		TEXT,
			"hash"		TEXT,
			"height"	INTEGER,
			"offset"	INTEGER,
			UNIQUE("db", "hash")
		)`,
		`CREATE TABLE IF NOT EXISTS "subscription" (
			"db"		TEXT,
			"count"	INTEGER
		)`,
	}
	getAllSubscriptionsSQL = `SELECT "db", "count" FROM "subscription"`
	saveSubscriptionSQL    = `INSERT OR REPLACE INTO "subscription" ("db", "count") VALUES(?, ?)`
	saveAckSQL             = `INSERT OR REPLACE INTO "ack" ("db", "hash", "height", "offset") VALUES(?, ?, ?, ?)`
	saveRequestSQL         = `INSERT OR REPLACE INTO "request" ("db", "hash", "height", "offset") VALUES(?, ?, ?, ?)`
	saveResponseSQL        = `INSERT OR REPLACE INTO "response" ("db", "hash", "height", "offset") VALUES(?, ?, ?, ?)`
	saveBlockSQL           = `INSERT OR REPLACE INTO "block" ("db", "height", "count", "hash", "block") VALUES(?, ?, ?, ?, ?)`
	getAckByHashSQL        = `SELECT "height", "offset" FROM "ack" WHERE "db" = ? AND "hash" = ? LIMIT 1`
	getRequestByHashSQL    = `SELECT "height", "offset" FROM "request" WHERE "db" = ? AND "hash" = ? LIMIT 1`
	getResponseByHashSQL   = `SELECT "height", "offset" FROM "response" WHERE "db" = ? AND "hash" = ? LIMIT 1`
	getHighestBlockSQL     = `SELECT "height", "count", "block" FROM "block" WHERE "db" = ? ORDER BY "count" DESC LIMIT 1`
	getBlockByHeightSQL    = `SELECT "count", "block" FROM "block" WHERE "db" = ? AND "height" = ? LIMIT 1`
	getBlockByCountSQL     = `SELECT "height", "block" FROM "block" WHERE "db" = ? AND "count" = ? LIMIT 1`
	getBlockByHashSQL      = `SELECT "height", "count", "block" FROM "block" WHERE "db" = ? AND "hash" = ? LIMIT 1`
)

// Service defines the observer service structure.
type Service struct {
	subscription    sync.Map // map[proto.DatabaseID]*subscribeWorker
	upstreamServers sync.Map // map[proto.DatabaseID]*types.ServiceInstance

	db      *xs.SQLite3
	caller  *rpc.Caller
	stopped int32
}

// NewService creates new observer service and load previous subscription from the meta database.
func NewService() (service *Service, err error) {
	// open observer database
	dbFile := filepath.Join(conf.GConf.WorkingRoot, dbFileName)

	db, err := xs.NewSqlite(dbFile)
	if err != nil {
		return
	}

	defer func() {
		if err != nil {
			_ = db.Close()
		}
	}()

	for _, q := range initTableSQL {
		if _, err = db.Writer().Exec(q); err != nil {
			err = errors.Wrap(err, "init table failed")
			return
		}
	}

	// init service
	service = &Service{
		db:     db,
		caller: rpc.NewCallerWithPool(mux.GetSessionPoolInstance()),
	}

	// load previous subscriptions
	rows, err := db.Writer().Query(getAllSubscriptionsSQL)
	if err != nil {
		err = errors.Wrap(err, "query previous subscriptions failed")
		return
	}

	defer func() {
		_ = rows.Close()
	}()

	for rows.Next() {
		var (
			rawDBID string
			count   int32
		)

		if err = rows.Scan(&rawDBID, &count); err != nil {
			err = errors.Wrap(err, "scan subscriptions failed")
			return
		}

		dbID := proto.DatabaseID(rawDBID)

		service.subscription.Store(dbID, newSubscribeWorker(dbID, count, service))
	}

	return
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

func (s *Service) saveSubscriptionStatus(dbID proto.DatabaseID, count int32) (err error) {
	log.WithFields(log.Fields{}).Debug("save subscription status")

	if atomic.LoadInt32(&s.stopped) == 1 {
		// stopped
		return ErrStopped
	}

	_, err = s.db.Writer().Exec(saveSubscriptionSQL, string(dbID), count)
	if err != nil {
		err = errors.Wrapf(err, "save subscription failed: %s, %d", dbID, count)
	}
	return
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
	_, err = s.db.Writer().Exec(saveAckSQL, string(dbID), ack.Hash().String(), height, offset)
	if err != nil {
		err = errors.Wrapf(err, "save ack failed: %s, %s, %d", dbID, ack.Hash().String(), height)
	}
	return
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

	_, err = s.db.Writer().Exec(saveRequestSQL, string(dbID), qt.Request.Header.Hash().String(), height, offset)
	if err != nil {
		err = errors.Wrapf(err, "save request failed: %s, %s, %d", dbID, qt.Request.Header.Hash().String(), height)
		return
	}
	_, err = s.db.Writer().Exec(saveResponseSQL, string(dbID), qt.Response.Hash().String(), height, offset)
	if err != nil {
		err = errors.Wrapf(err, "save response failed: %s, %s, %d", dbID, qt.Response.Hash().String(), height)
	}
	return
}

func (s *Service) addBlock(dbID proto.DatabaseID, count int32, b *types.Block) (err error) {
	instance, err := s.getUpstream(dbID)
	if err != nil {
		return
	}
	if b == nil {
		return
	}

	h := int32(b.Timestamp().Sub(instance.GenesisBlock.Timestamp()) / conf.GConf.SQLChainPeriod)
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

	_, err = s.db.Writer().Exec(saveBlockSQL, string(dbID), h, count, b.BlockHash().String(), blockBytes.Bytes())
	if err != nil {
		err = errors.Wrapf(err, "save block failed: %s, %s, %d", dbID, b.BlockHash().String(), count)
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
	_ = s.db.Close()

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

	curBP, err := mux.GetCurrentBP()
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

	err = s.db.Writer().QueryRow(getAckByHashSQL, string(dbID), h.String()).Scan(&blockHeight, &dataOffset)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			err = ErrNotFound
			return
		}

		err = errors.Wrapf(err, "query ack failed: %s, %s", dbID, h.String())
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

	err = s.db.Writer().QueryRow(getRequestByHashSQL, string(dbID), h.String()).Scan(&blockHeight, &dataOffset)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			err = ErrNotFound
			return
		}

		err = errors.Wrapf(err, "query request failed: %s, %s", dbID, h.String())
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

	err = s.db.Writer().QueryRow(getResponseByHashSQL, string(dbID), h.String()).Scan(&blockHeight, &dataOffset)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			err = ErrNotFound
			return
		}

		err = errors.Wrapf(err, "query response failed: %s, %s", dbID, h.String())
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
	var (
		blockData []byte
		count     int32
	)
	err = s.db.Writer().QueryRow(getHighestBlockSQL, string(dbID)).Scan(&height, &count, &blockData)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			err = ErrNotFound
			return
		}

		err = errors.Wrapf(err, "query highest block failed: %s", dbID)
		return
	}

	err = utils.DecodeMsgPack(blockData, &b)
	if err != nil {
		err = errors.Wrapf(err, "decode block failed: %s", dbID)
	}

	return
}

func (s *Service) getHighestBlockV2(
	dbID proto.DatabaseID) (count, height int32, b *types.Block, err error,
) {
	var blockData []byte
	err = s.db.Writer().QueryRow(getHighestBlockSQL, string(dbID)).Scan(&height, &count, &blockData)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			err = ErrNotFound
			return
		}

		err = errors.Wrapf(err, "query highest block failed: %s", dbID)
		return
	}

	err = utils.DecodeMsgPack(blockData, &b)
	if err != nil {
		err = errors.Wrapf(err, "decode block failed: %s", dbID)
	}

	return
}

func (s *Service) getBlockByHeight(dbID proto.DatabaseID, height int32) (count int32, b *types.Block, err error) {
	var blockData []byte
	err = s.db.Writer().QueryRow(getBlockByHeightSQL, string(dbID), height).Scan(&count, &blockData)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			err = ErrNotFound
			return
		}

		err = errors.Wrapf(err, "query block by height failed: %s, %d", dbID, height)
		return
	}

	err = utils.DecodeMsgPack(blockData, &b)
	if err != nil {
		err = errors.Wrapf(err, "decode block failed: %s", dbID)
	}

	return
}

func (s *Service) getBlockByCount(
	dbID proto.DatabaseID, count int32) (height int32, b *types.Block, err error,
) {
	var blockData []byte
	err = s.db.Writer().QueryRow(getBlockByCountSQL, string(dbID), count).Scan(&height, &blockData)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			err = ErrNotFound
			return
		}

		err = errors.Wrapf(err, "query block by count failed: %s, %d", dbID, count)
		return
	}

	err = utils.DecodeMsgPack(blockData, &b)
	if err != nil {
		err = errors.Wrapf(err, "decode block failed: %s", dbID)
	}

	return
}

func (s *Service) getBlock(dbID proto.DatabaseID, h *hash.Hash) (count int32, height int32, b *types.Block, err error) {
	var blockData []byte
	err = s.db.Writer().QueryRow(getBlockByHashSQL, string(dbID), h.String()).Scan(&height, &count, &blockData)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			err = ErrNotFound
			return
		}

		err = errors.Wrapf(err, "query block by hash failed: %s, %s", dbID, h.String())
		return
	}

	err = utils.DecodeMsgPack(blockData, &b)
	if err != nil {
		err = errors.Wrapf(err, "decode block failed: %s", dbID)
	}

	return
}

func (s *Service) getAllSubscriptions() (subscriptions map[proto.DatabaseID]int32, err error) {
	subscriptions = map[proto.DatabaseID]int32{}
	s.subscription.Range(func(_, rawWorker interface{}) bool {
		worker := unpackWorker(rawWorker)
		subscriptions[worker.dbID] = worker.getHead()
		return true
	})

	return
}
