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
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	ka "github.com/CovenantSQL/CovenantSQL/kayak/api"
	kt "github.com/CovenantSQL/CovenantSQL/kayak/transport"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	"github.com/CovenantSQL/CovenantSQL/sqlchain"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	wt "github.com/CovenantSQL/CovenantSQL/worker/types"
)

const (
	// DBKayakRPCName defines rpc service name of database internal consensus.
	DBKayakRPCName = "DBC" // aka. database consensus

	// SQLChainRPCName defines rpc service name of sql-chain internal consensus.
	SQLChainRPCName = "SQLC"

	// DBServiceRPCName defines rpc service name of database external query api.
	DBServiceRPCName = "DBS" // aka. database service

	// DBMetaFileName defines dbms meta file name.
	DBMetaFileName = "db.meta"
)

// DBMS defines a database management instance.
type DBMS struct {
	cfg      *DBMSConfig
	dbMap    sync.Map
	kayakMux *kt.ETLSTransportService
	chainMux *sqlchain.MuxService
	rpc      *DBMSRPCService
}

// NewDBMS returns new database management instance.
func NewDBMS(cfg *DBMSConfig) (dbms *DBMS, err error) {
	dbms = &DBMS{
		cfg: cfg,
	}

	// init kayak rpc mux
	dbms.kayakMux = ka.NewMuxService(DBKayakRPCName, cfg.Server)

	// init sql-chain rpc mux
	dbms.chainMux = sqlchain.NewMuxService(SQLChainRPCName, cfg.Server)

	// init service
	dbms.rpc = NewDBMSRPCService(DBServiceRPCName, cfg.Server, dbms)

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
		return
	}

	// load current peers info from block producer
	var dbMapping []wt.ServiceInstance
	if dbMapping, err = dbms.getMappedInstances(); err != nil {
		return
	}

	// init database
	if err = dbms.initDatabases(localMeta, dbMapping); err != nil {
		return
	}

	return
}

func (dbms *DBMS) initDatabases(meta *DBMSMeta, conf []wt.ServiceInstance) (err error) {
	currentInstance := make(map[proto.DatabaseID]bool)

	for _, instanceConf := range conf {
		currentInstance[instanceConf.DatabaseID] = true
		if err = dbms.Create(&instanceConf, false); err != nil {
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
func (dbms *DBMS) Create(instance *wt.ServiceInstance, cleanup bool) (err error) {
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
	}

	if db, err = NewDatabase(dbCfg, instance.Peers, instance.GenesisBlock); err != nil {
		return
	}

	// add to meta
	err = dbms.addMeta(instance.DatabaseID, db)

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
func (dbms *DBMS) Update(instance *wt.ServiceInstance) (err error) {
	var db *Database
	var exists bool

	if db, exists = dbms.getMeta(instance.DatabaseID); !exists {
		return ErrNotExists
	}

	// update peers
	return db.UpdatePeers(instance.Peers)
}

// Query handles query request in dbms.
func (dbms *DBMS) Query(req *wt.Request) (res *wt.Response, err error) {
	var db *Database
	var exists bool

	// find database
	if db, exists = dbms.getMeta(req.Header.DatabaseID); !exists {
		err = ErrNotExists
		return
	}

	// send query
	return db.Query(req)
}

// Ack handles ack of previous response.
func (dbms *DBMS) Ack(ack *wt.Ack) (err error) {
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

// GetRequest handles fetching original request of previous transactions.
func (dbms *DBMS) GetRequest(dbID proto.DatabaseID, offset uint64) (query *wt.Request, err error) {
	var db *Database
	var exists bool

	if db, exists = dbms.getMeta(dbID); !exists {
		err = ErrNotExists
		return
	}

	var reqBytes []byte
	if reqBytes, err = db.kayakRuntime.GetLog(offset); err != nil {
		return
	}

	// decode requests
	var q wt.Request
	if err = utils.DecodeMsgPack(reqBytes, &q); err != nil {
		return
	}

	query = &q

	return
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

func (dbms *DBMS) getMappedInstances() (instances []wt.ServiceInstance, err error) {
	var bpNodeID proto.NodeID
	if bpNodeID, err = rpc.GetCurrentBP(); err != nil {
		return
	}

	req := &wt.InitService{}
	res := new(wt.InitServiceResponse)

	if err = rpc.NewCaller().CallNode(bpNodeID, route.BPDBGetNodeDatabases.String(), req, res); err != nil {
		return
	}

	// verify response
	if err = res.Verify(); err != nil {
		return
	}

	instances = res.Header.Instances

	return
}

// Shutdown defines dbms shutdown logic.
func (dbms *DBMS) Shutdown() (err error) {
	dbms.dbMap.Range(func(_, rawDB interface{}) bool {
		db := rawDB.(*Database)

		if err = db.Shutdown(); err != nil {
			log.Errorf("shutdown database failed: %v", err)
		}

		return true
	})

	// persist meta
	err = dbms.writeMeta()

	return
}
