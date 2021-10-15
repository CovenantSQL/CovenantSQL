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

package worker

import (
	"bytes"
	"context"
	"expvar"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/sqlchain"
	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

const (
	// DBKayakRPCName defines rpc service name of database internal consensus.
	DBKayakRPCName = "DBC" // aka. database consensus

	// DBMetaFileName defines dbms meta file name.
	DBMetaFileName = "db.meta"

	// DefaultSlowQueryTime defines the default slow query log time
	DefaultSlowQueryTime = time.Second * 5

	mwMinerDBCount = "service:miner:db:count"
)

var (
	dbCount = new(expvar.Int)
)

func init() {
	expvar.Publish(mwMinerDBCount, dbCount)
}

// DBMS defines a database management instance.
type DBMS struct {
	cfg        *DBMSConfig
	dbMap      sync.Map
	kayakMux   *DBKayakMuxService
	chainMux   *sqlchain.MuxService
	rpc        *DBMSRPCService
	busService *BusService
	address    proto.AccountAddress
	privKey    *asymmetric.PrivateKey
}

// NewDBMS returns new database management instance.
func NewDBMS(cfg *DBMSConfig) (dbms *DBMS, err error) {
	dbms = &DBMS{
		cfg: cfg,
	}

	// init kayak rpc mux
	if dbms.kayakMux, err = NewDBKayakMuxService(DBKayakRPCName, cfg.Server); err != nil {
		err = errors.Wrap(err, "register kayak mux service failed")
		return
	}

	// init sql-chain rpc mux
	if dbms.chainMux, err = sqlchain.NewMuxService(route.SQLChainRPCName, cfg.Server); err != nil {
		err = errors.Wrap(err, "register sqlchain mux service failed")
		return
	}

	// cache address of node
	var (
		pk   *asymmetric.PublicKey
		addr proto.AccountAddress
	)
	if pk, err = kms.GetLocalPublicKey(); err != nil {
		err = errors.Wrap(err, "failed to cache public key")
		return
	}
	if addr, err = crypto.PubKeyHash(pk); err != nil {
		err = errors.Wrap(err, "generate address failed")
		return
	}
	dbms.address = addr

	// init chain bus service
	ctx := context.Background()
	bs := NewBusService(ctx, addr, conf.GConf.ChainBusPeriod)
	dbms.busService = bs

	// private key cache
	dbms.privKey, err = kms.GetLocalPrivateKey()
	if err != nil {
		log.WithError(err).Warning("get private key failed")
		return
	}

	// init service
	dbms.rpc = NewDBMSRPCService(route.DBRPCName, cfg.Server, cfg.DirectServer, dbms)
	return
}

func (dbms *DBMS) readMeta() (meta *DBMSMeta, err error) {
	filePath := filepath.Join(dbms.cfg.RootDir, DBMetaFileName)
	meta = NewDBMSMeta()

	var fileContent []byte

	if fileContent, err = ioutil.ReadFile(filePath); err != nil {
		// if not exists
		if os.IsNotExist(err) {
			// new without meta
			err = nil
			return
		}

		return
	}

	err = utils.DecodeMsgPack(fileContent, meta)

	// create empty meta if meta map is nil
	if err == nil && meta.DBS == nil {
		meta = NewDBMSMeta()
	}

	return
}

func (dbms *DBMS) writeMeta() (err error) {
	meta := NewDBMSMeta()

	dbms.dbMap.Range(func(key, value interface{}) bool {
		dbID := key.(proto.DatabaseID)
		meta.DBS[dbID] = true
		return true
	})

	var buf *bytes.Buffer
	if buf, err = utils.EncodeMsgPack(meta); err != nil {
		return
	}

	filePath := filepath.Join(dbms.cfg.RootDir, DBMetaFileName)
	err = ioutil.WriteFile(filePath, buf.Bytes(), 0644)

	return
}

// Init defines dbms init logic.
func (dbms *DBMS) Init() (err error) {
	// read meta
	var localMeta *DBMSMeta
	if localMeta, err = dbms.readMeta(); err != nil {
		err = errors.Wrap(err, "read dbms meta failed")
		return
	}

	// load current peers info from block producer
	var dbMapping = dbms.busService.GetCurrentDBMapping()

	// init database
	if err = dbms.initDatabases(localMeta, dbMapping); err != nil {
		err = errors.Wrap(err, "init databases with meta failed")
		return
	}

	if err = dbms.busService.Subscribe("/CreateDatabase/", dbms.createDatabase); err != nil {
		err = errors.Wrap(err, "init chain bus failed")
		return
	}
	if err = dbms.busService.Subscribe("/UpdateBilling/", dbms.updateBilling); err != nil {
		err = errors.Wrap(err, "init chain bus failed")
		return
	}
	dbms.busService.Start()

	return
}

func (dbms *DBMS) updateBilling(itx interfaces.Transaction, count uint32) {
	var (
		tx *types.UpdateBilling
		ok bool
	)
	if tx, ok = itx.(*types.UpdateBilling); !ok {
		log.WithFields(log.Fields{
			"type": itx.GetTransactionType(),
		}).WithError(ErrInvalidTransactionType).Warn("invalid tx type in update billing")
		return
	}
	// Get profile and database instance
	var (
		id       = tx.Receiver.DatabaseID()
		profile  *types.SQLChainProfile
		database *Database
	)
	le := log.WithFields(log.Fields{
		"id": id,
	})
	if database, ok = dbms.getMeta(id); !ok {
		le.Warn("cannot find database")
		return
	}
	if profile, ok = dbms.busService.RequestSQLProfile(id); !ok {
		le.Warn("cannot find profile")
		return
	}
	database.chain.SetLastBillingHeight(int32(profile.LastUpdatedHeight))
}

func (dbms *DBMS) createDatabase(tx interfaces.Transaction, count uint32) {
	cd, ok := tx.(*types.CreateDatabase)
	if !ok {
		log.WithError(ErrInvalidTransactionType).Warningf("invalid tx type in createDatabase: %s",
			tx.GetTransactionType().String())
		return
	}

	var (
		dbID          = proto.FromAccountAndNonce(cd.Owner, uint32(cd.Nonce))
		isTargetMiner = false
	)
	log.WithFields(log.Fields{
		"databaseid": dbID,
		"owner":      cd.Owner.String(),
		"nonce":      cd.Nonce,
	}).Debug("in createDatabase")
	p, ok := dbms.busService.RequestSQLProfile(dbID)
	if !ok {
		log.WithFields(log.Fields{
			"databaseid": dbID,
		}).Warning("database profile not found")
		return
	}

	nodeIDs := make([]proto.NodeID, len(p.Miners))

	for i, mi := range p.Miners {
		if mi.Address == dbms.address {
			isTargetMiner = true
		}
		nodeIDs[i] = mi.NodeID
	}
	if !isTargetMiner {
		return
	}

	var si, err = dbms.buildSQLChainServiceInstance(p)
	if err != nil {
		log.WithError(err).Warn("failed to build sqlchain service instance from profile")
	}
	err = dbms.Create(si, true)
	if err != nil {
		log.WithError(err).Error("create database error")
	}

	if dbms.cfg.OnCreateDatabase != nil {
		go dbms.cfg.OnCreateDatabase()
	}
}

func (dbms *DBMS) buildSQLChainServiceInstance(
	profile *types.SQLChainProfile) (instance *types.ServiceInstance, err error,
) {
	var (
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
	if dbms.privKey == nil {
		if dbms.privKey, err = kms.GetLocalPrivateKey(); err != nil {
			log.WithError(err).Warning("get private key failed in createDatabase")
			return
		}
	}
	if err = peers.Sign(dbms.privKey); err != nil {
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
	return
}

// UpdatePermission exports the update permission interface for test.
func (dbms *DBMS) UpdatePermission(dbID proto.DatabaseID, user proto.AccountAddress, permStat *types.PermStat) (err error) {
	dbms.busService.lock.Lock()
	defer dbms.busService.lock.Unlock()
	profile, ok := dbms.busService.sqlChainProfiles[dbID]
	if !ok {
		dbms.busService.sqlChainProfiles[dbID] = &types.SQLChainProfile{
			ID: dbID,
			Users: []*types.SQLChainUser{
				&types.SQLChainUser{
					Address:    user,
					Permission: permStat.Permission,
					Status:     permStat.Status,
				},
			},
		}
	} else {
		exist := false
		for _, u := range profile.Users {
			if u.Address == user {
				u.Permission = permStat.Permission
				u.Status = permStat.Status
				exist = true
				break
			}
		}
		if !exist {
			profile.Users = append(profile.Users, &types.SQLChainUser{
				Address:    user,
				Permission: permStat.Permission,
				Status:     permStat.Status,
			})
			dbms.busService.sqlChainProfiles[dbID] = profile
		}
	}

	_, ok = dbms.busService.sqlChainState[dbID]
	if !ok {
		dbms.busService.sqlChainState[dbID] = make(map[proto.AccountAddress]*types.PermStat)
	}
	dbms.busService.sqlChainState[dbID][user] = permStat

	return
}

func (dbms *DBMS) initDatabases(
	meta *DBMSMeta, profiles map[proto.DatabaseID]*types.SQLChainProfile) (err error,
) {
	currentInstance := make(map[proto.DatabaseID]bool)
	wg := &sync.WaitGroup{}

	for id, profile := range profiles {
		currentInstance[id] = true
		var instance *types.ServiceInstance
		if instance, err = dbms.buildSQLChainServiceInstance(profile); err != nil {
			return
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := dbms.Create(instance, false); err != nil {
				log.WithFields(log.Fields{
					"id": instance.DatabaseID,
				}).WithError(err).Error("failed to create database instance")
			}
		}()
	}
	wg.Wait()

	// calculate to drop databases
	toDropInstance := make(map[proto.DatabaseID]bool)

	for dbID := range meta.DBS {
		if _, exists := currentInstance[dbID]; !exists {
			toDropInstance[dbID] = true
		}
	}

	// drop database
	for dbID := range toDropInstance {
		if err = dbms.Drop(dbID); err != nil {
			return
		}
	}

	return
}

// Create add new database to the miner dbms.
func (dbms *DBMS) Create(instance *types.ServiceInstance, cleanup bool) (err error) {
	if _, alreadyExists := dbms.getMeta(instance.DatabaseID); alreadyExists {
		return ErrAlreadyExists
	}

	// set database root dir
	rootDir := filepath.Join(dbms.cfg.RootDir, string(instance.DatabaseID))

	// clear current data
	if cleanup {
		if err = os.RemoveAll(rootDir); err != nil {
			return
		}
	}

	var db *Database

	defer func() {
		// close on failed
		if err != nil {
			if db != nil {
				db.Shutdown()
			}

			dbms.removeMeta(instance.DatabaseID)
		}
	}()

	// new db
	dbCfg := &DBConfig{
		DatabaseID:             instance.DatabaseID,
		RootDir:                dbms.cfg.RootDir,
		DataDir:                rootDir,
		KayakMux:               dbms.kayakMux,
		ChainMux:               dbms.chainMux,
		MaxWriteTimeGap:        dbms.cfg.MaxReqTimeGap,
		EncryptionKey:          instance.ResourceMeta.EncryptionKey,
		SpaceLimit:             instance.ResourceMeta.Space,
		UpdateBlockCount:       conf.GConf.BillingBlockCount,
		UseEventualConsistency: instance.ResourceMeta.UseEventualConsistency,
		ConsistencyLevel:       instance.ResourceMeta.ConsistencyLevel,
		IsolationLevel:         instance.ResourceMeta.IsolationLevel,
		SlowQueryTime:          DefaultSlowQueryTime,
	}

	// set last billing height
	if profile, ok := dbms.busService.RequestSQLProfile(dbCfg.DatabaseID); ok {
		dbCfg.LastBillingHeight = int32(profile.LastUpdatedHeight)
	}

	if db, err = NewDatabase(dbCfg, instance.Peers, instance.GenesisBlock); err != nil {
		return
	}

	// add to meta
	err = dbms.addMeta(instance.DatabaseID, db)

	// update metrics
	dbCount.Add(1)

	return
}

// Drop remove database from the miner dbms.
func (dbms *DBMS) Drop(dbID proto.DatabaseID) (err error) {
	var db *Database
	var exists bool

	if db, exists = dbms.getMeta(dbID); !exists {
		return ErrNotExists
	}

	// shutdown database
	if err = db.Destroy(); err != nil {
		return
	}

	// update metrics
	dbCount.Add(-1)

	// remove meta
	return dbms.removeMeta(dbID)
}

// Query handles query request in dbms.
func (dbms *DBMS) Query(req *types.Request) (res *types.Response, err error) {
	var db *Database
	var exists bool

	// check permission
	addr, err := crypto.PubKeyHash(req.Header.Signee)
	if err != nil {
		return
	}
	err = dbms.checkPermission(addr, req.Header.DatabaseID, req.Header.QueryType, req.Payload.Queries)
	if err != nil {
		return
	}

	// find database
	if db, exists = dbms.getMeta(req.Header.DatabaseID); !exists {
		err = ErrNotExists
		return
	}

	return db.Query(req)
}

// Ack handles ack of previous response.
func (dbms *DBMS) Ack(ack *types.Ack) (err error) {
	var db *Database
	var exists bool

	// check permission
	addr, err := crypto.PubKeyHash(ack.Header.Signee)
	if err != nil {
		return
	}
	err = dbms.checkPermission(addr, ack.Header.Response.Request.DatabaseID, types.ReadQuery, nil)
	if err != nil {
		return
	}
	// find database
	if db, exists = dbms.getMeta(ack.Header.Response.Request.DatabaseID); !exists {
		err = ErrNotExists
		return
	}

	// send query
	return db.Ack(ack)
}

func (dbms *DBMS) getMeta(dbID proto.DatabaseID) (db *Database, exists bool) {
	var rawDB interface{}

	if rawDB, exists = dbms.dbMap.Load(dbID); !exists {
		return
	}

	db = rawDB.(*Database)

	return
}

func (dbms *DBMS) addMeta(dbID proto.DatabaseID, db *Database) (err error) {
	if _, alreadyExists := dbms.dbMap.LoadOrStore(dbID, db); alreadyExists {
		return ErrAlreadyExists
	}

	return dbms.writeMeta()
}

func (dbms *DBMS) removeMeta(dbID proto.DatabaseID) (err error) {
	dbms.dbMap.Delete(dbID)
	return dbms.writeMeta()
}

func (dbms *DBMS) checkPermission(addr proto.AccountAddress,
	dbID proto.DatabaseID, queryType types.QueryType, queries []types.Query) (err error) {
	log.Debugf("in checkPermission, database id: %s, user addr: %s", dbID, addr.String())

	var (
		permStat *types.PermStat
		ok       bool
	)

	// get database perm stat
	permStat, ok = dbms.busService.RequestPermStat(dbID, addr)

	// perm stat not exists
	if !ok {
		err = errors.Wrap(ErrPermissionDeny, "database not exists")
		return
	}

	// check if query is enabled
	if !permStat.Status.EnableQuery() {
		err = errors.Wrapf(ErrPermissionDeny, "cannot query, status: %d", permStat.Status)
		return
	}

	// check query type permission
	switch queryType {
	case types.ReadQuery:
		if !permStat.Permission.HasReadPermission() {
			err = errors.Wrapf(ErrPermissionDeny, "cannot read, permission: %v", permStat.Permission)
			return
		}
	case types.WriteQuery:
		if !permStat.Permission.HasWritePermission() {
			err = errors.Wrapf(ErrPermissionDeny, "cannot write, permission: %v", permStat.Permission)
			return
		}
	default:
		err = errors.Wrapf(ErrInvalidPermission,
			"invalid permission, permission: %v", permStat.Permission)
		return
	}

	// check for query pattern
	var (
		disallowedQuery    string
		hasDisallowedQuery bool
	)

	if disallowedQuery, hasDisallowedQuery = permStat.Permission.HasDisallowedQueryPatterns(queries); hasDisallowedQuery {
		err = errors.Wrapf(ErrPermissionDeny, "disallowed query %s", disallowedQuery)
		log.WithError(err).WithFields(log.Fields{
			"permission": permStat.Permission,
			"query":      disallowedQuery,
		}).Debug("can not query")
		return
	}

	return
}

// Shutdown defines dbms shutdown logic.
func (dbms *DBMS) Shutdown() (err error) {
	dbms.dbMap.Range(func(_, rawDB interface{}) bool {
		db := rawDB.(*Database)

		if err = db.Shutdown(); err != nil {
			log.WithError(err).Error("shutdown database failed")
		}

		return true
	})

	// persist meta
	err = dbms.writeMeta()

	dbms.busService.Stop()

	return
}
