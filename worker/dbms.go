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
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	"github.com/CovenantSQL/CovenantSQL/crypto"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/sqlchain"
	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/pkg/errors"
)

const (
	// DBKayakRPCName defines rpc service name of database internal consensus.
	DBKayakRPCName = "DBC" // aka. database consensus

	// DBMetaFileName defines dbms meta file name.
	DBMetaFileName = "db.meta"

	// CheckInterval defines the bus service period.
	CheckInterval = time.Second

	// UpdatePeriod defines the
)

// DBMS defines a database management instance.
type DBMS struct {
	cfg        *DBMSConfig
	dbMap      sync.Map
	chainMap   sync.Map
	kayakMux   *DBKayakMuxService
	chainMux   *sqlchain.MuxService
	rpc        *DBMSRPCService
	busService *sqlchain.BusService
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
	bs := sqlchain.NewBusService(ctx, addr, CheckInterval)
	dbms.busService = bs

	// private key cache
	dbms.privKey, err = kms.GetLocalPrivateKey()
	if err != nil {
		log.WithError(err).Warning("get private key failed")
		return
	}

	// init service
	dbms.rpc = NewDBMSRPCService(route.DBRPCName, cfg.Server, dbms)
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

	dbms.busService.Start()
	if err = dbms.busService.Subscribe("/CreateDatabase/", dbms.createDatabase); err != nil {
		err = errors.Wrap(err, "init chain bus failed")
		return
	}
	if err = dbms.busService.Subscribe("/UpdatePermission/", dbms.updatePermission); err != nil {
		err = errors.Wrap(err, "init chain bus failed")
		return
	}
	if err = dbms.busService.Subscribe("/UpdateBilling/", dbms.updateBilling); err != nil {
		err = errors.Wrap(err, "init chain bus failed")
		return
	}

	return
}

func (dbms *DBMS) updateBilling(tx interfaces.Transaction, count uint32) {
	ub, ok := tx.(*types.UpdateBilling)
	if !ok {
		log.WithError(ErrInvalidTransactionType).Warningf("invalid tx type in updateBilling: %s",
			tx.GetTransactionType().String())
		return
	}

	var (
		dbid     = ub.Receiver.DatabaseID()
		newState = types.UserState{
			State: make(map[proto.AccountAddress]*types.PermStat),
		}
	)

	p, ok := dbms.busService.RequestSQLProfile(&dbid)
	if !ok {
		log.WithFields(log.Fields{
			"databaseid": dbid,
		}).Warning("database profile not found")
		return
	}

	for _, user := range p.Users {
		newState.State[user.Address] = &types.PermStat{
			Permission: user.Permission,
			Status:     user.Status,
		}
	}
	dbms.chainMap.Store(ub.Receiver.DatabaseID(), newState)
}

func (dbms *DBMS) createDatabase(tx interfaces.Transaction, count uint32) {
	cd, ok := tx.(*types.CreateDatabase)
	if !ok {
		log.WithError(ErrInvalidTransactionType).Warningf("invalid tx type in createDatabase: %s",
			tx.GetTransactionType().String())
		return
	}

	log.Debugf("create database with owner: %s, nonce: %d", cd.Owner.String(), cd.Nonce)
	var (
		dbid          = proto.FromAccountAndNonce(cd.Owner, uint32(cd.Nonce))
		isTargetMiner = false
	)
	p, ok := dbms.busService.RequestSQLProfile(dbid)
	if !ok {
		log.WithFields(log.Fields{
			"databaseid": &dbid,
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

	state := types.NewUserState()
	for _, user := range p.Users {
		log.Debugf("user address: %s, permission: %d, status: %d",
			user.Address.String(), user.Permission, user.Status)
		state.State[user.Address] = &types.PermStat{
			Permission: user.Permission,
			Status:     user.Status,
		}
	}

	var si, err = dbms.buildSQLChainServiceInstance(p)
	if err != nil {
		log.WithError(err).Warn("failed to build sqlchain service instance from profile")
	}
	err = dbms.Create(si, true)
	if err != nil {
		log.WithError(err).Error("create database error")
	}
	dbms.chainMap.Store(*dbid, *state)
}

func (dbms *DBMS) buildSQLChainServiceInstance(
	profile *types.SQLChainProfile) (instance *types.ServiceInstance, err error,
) {
	var (
		nodeids = make([]proto.NodeID, len(profile.Miners))
		peers   *proto.Peers
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
	instance = &types.ServiceInstance{
		DatabaseID:   profile.ID,
		Peers:        peers,
		ResourceMeta: profile.Meta,
		GenesisBlock: profile.Genesis,
	}
	return
}

func (dbms *DBMS) updatePermission(tx interfaces.Transaction, count uint32) {
	up, ok := tx.(*types.UpdatePermission)
	if !ok {
		log.WithError(ErrInvalidTransactionType).Warning("unexpected error in updatePermission")
		return
	}

	state, loaded := dbms.chainMap.Load(up.TargetSQLChain.DatabaseID())
	if !loaded {
		return
	}

	newState := state.(types.UserState)
	newState.AddPermission(up.TargetUser, up.Permission)
	dbms.chainMap.Store(up.TargetSQLChain.DatabaseID(), newState)
}

// UpdatePermission exports the update permission interface for test.
func (dbms *DBMS) UpdatePermission(dbid proto.DatabaseID, user proto.AccountAddress, permStat *types.PermStat) (err error) {
	s, loaded := dbms.chainMap.Load(dbid)
	if !loaded {
		err = errors.Wrap(ErrNotExists, "update permission failed")
		return
	}
	state := s.(types.UserState)
	state.State[user] = permStat
	return
}

func (dbms *DBMS) initDatabases(
	meta *DBMSMeta, profiles map[proto.DatabaseID]*types.SQLChainProfile) (err error,
) {
	currentInstance := make(map[proto.DatabaseID]bool)

	for id, profile := range profiles {
		currentInstance[id] = true
		var instance *types.ServiceInstance
		if instance, err = dbms.buildSQLChainServiceInstance(profile); err != nil {
			return
		}
		if err = dbms.Create(instance, false); err != nil {
			return
		}
	}

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
		DatabaseID:      instance.DatabaseID,
		DataDir:         rootDir,
		KayakMux:        dbms.kayakMux,
		ChainMux:        dbms.chainMux,
		MaxWriteTimeGap: dbms.cfg.MaxReqTimeGap,
		EncryptionKey:   instance.ResourceMeta.EncryptionKey,
		SpaceLimit:      instance.ResourceMeta.Space,
		// TODO(lambda): make UpdatePeriod Configurable
		UpdatePeriod:           2,
		UseEventualConsistency: instance.ResourceMeta.UseEventualConsistency,
		ConsistencyLevel:       instance.ResourceMeta.ConsistencyLevel,
	}

	if db, err = NewDatabase(dbCfg, instance.Peers, instance.GenesisBlock); err != nil {
		return
	}

	// add to meta
	err = dbms.addMeta(instance.DatabaseID, db)

	// init chainMap
	dbms.chainMap.Store(instance.DatabaseID, types.UserState{
		State: make(map[proto.AccountAddress]*types.PermStat),
	})

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

	// remove meta
	return dbms.removeMeta(dbID)
}

// Update apply the new peers config to dbms.
func (dbms *DBMS) Update(instance *types.ServiceInstance) (err error) {
	var db *Database
	var exists bool

	if db, exists = dbms.getMeta(instance.DatabaseID); !exists {
		return ErrNotExists
	}

	// update peers
	return db.UpdatePeers(instance.Peers)
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
	err = dbms.checkPermission(addr, req)
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

func (dbms *DBMS) checkPermission(addr proto.AccountAddress, req *types.Request) (err error) {
	log.Debugf("in checkPermission, database id: %s, user addr: %s", req.Header.DatabaseID, addr.String())
	state, loaded := dbms.chainMap.Load(req.Header.DatabaseID)
	if !loaded {
		err = errors.Wrap(ErrNotExists, "check permission failed")
		return
	}

	var (
		s = state.(types.UserState)
	)

	if permStat, ok := s.State[addr]; ok {
		if !permStat.Status.EnableQuery() {
			err = ErrPermissionDeny
			log.WithError(err).Debugf("cannot query, status: %d", permStat.Status)
			return
		}
		if req.Header.QueryType == types.ReadQuery {
			if !permStat.Permission.CheckRead() {
				err = ErrPermissionDeny
				log.WithError(err).Debugf("cannot read, permission: %d", permStat.Permission)
				return
			}
		} else if req.Header.QueryType == types.WriteQuery {
			if !permStat.Permission.CheckWrite() {
				err = ErrPermissionDeny
				log.WithError(err).Debugf("cannot write, permission: %d", permStat.Permission)
				return
			}
		} else {
			err = ErrInvalidPermission
			log.WithError(err).Debugf("invalid permission, permission: %d", permStat.Permission)
			return

		}
	} else {
		err = ErrPermissionDeny
		log.WithError(err).Debug("cannot find permission")
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
