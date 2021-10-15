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
	"context"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/kayak"
	kt "github.com/CovenantSQL/CovenantSQL/kayak/types"
	kl "github.com/CovenantSQL/CovenantSQL/kayak/wal"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/sqlchain"
	"github.com/CovenantSQL/CovenantSQL/storage"
	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	x "github.com/CovenantSQL/CovenantSQL/xenomint"
)

const (
	// StorageFileName defines storage file name of database instance.
	StorageFileName = "storage.db3"

	// KayakWalFileName defines log pool name of database instance.
	KayakWalFileName = "kayak.ldb"

	// SQLChainFileName defines sqlchain storage file name.
	SQLChainFileName = "chain.db"

	// MaxRecordedConnectionSequences defines the max connection slots to anti reply attack.
	MaxRecordedConnectionSequences = 1000

	// PrepareThreshold defines the prepare complete threshold.
	PrepareThreshold = 1.0

	// CommitThreshold defines the commit complete threshold.
	CommitThreshold = 0.0

	// PrepareTimeout defines the prepare timeout config.
	PrepareTimeout = 10 * time.Second

	// CommitTimeout defines the commit timeout config.
	CommitTimeout = time.Minute

	// LogWaitTimeout defines the missing log wait timeout config.
	LogWaitTimeout = 10 * time.Second

	// SlowQuerySampleSize defines the maximum slow query log size (default: 1KB).
	SlowQuerySampleSize = 1 << 10
)

// Database defines a single database instance in worker runtime.
type Database struct {
	cfg            *DBConfig
	dbID           proto.DatabaseID
	kayakWal       *kl.LevelDBWal
	kayakRuntime   *kayak.Runtime
	kayakConfig    *kt.RuntimeConfig
	connSeqs       sync.Map
	connSeqEvictCh chan uint64
	chain          *sqlchain.Chain
	nodeID         proto.NodeID
	mux            *DBKayakMuxService
	privateKey     *asymmetric.PrivateKey
	accountAddr    proto.AccountAddress
}

// NewDatabase create a single database instance using config.
func NewDatabase(cfg *DBConfig, peers *proto.Peers,
	genesis *types.Block) (db *Database, err error) {
	// ensure dir exists
	if err = os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return
	}

	if peers == nil || genesis == nil {
		err = ErrInvalidDBConfig
		return
	}

	// get private key
	var privateKey *asymmetric.PrivateKey
	if privateKey, err = kms.GetLocalPrivateKey(); err != nil {
		return
	}

	var accountAddr proto.AccountAddress
	if accountAddr, err = crypto.PubKeyHash(privateKey.PubKey()); err != nil {
		return
	}

	// init database
	db = &Database{
		cfg:            cfg,
		dbID:           cfg.DatabaseID,
		mux:            cfg.KayakMux,
		connSeqEvictCh: make(chan uint64, 1),
		privateKey:     privateKey,
		accountAddr:    accountAddr,
	}

	defer func() {
		// on error recycle all resources
		if err != nil {
			// stop kayak runtime
			if db.kayakRuntime != nil {
				db.kayakRuntime.Shutdown()
			}

			// close chain
			if db.chain != nil {
				db.chain.Stop()
			}
		}
	}()

	// init storage
	storageFile := filepath.Join(cfg.DataDir, StorageFileName)
	storageDSN, err := storage.NewDSN(storageFile)
	if err != nil {
		return
	}

	if cfg.EncryptionKey != "" {
		storageDSN.AddParam("_crypto_key", cfg.EncryptionKey)
	}

	// init chain
	chainFile := filepath.Join(cfg.RootDir, SQLChainFileName)
	if db.nodeID, err = kms.GetLocalNodeID(); err != nil {
		return
	}

	chainCfg := &sqlchain.Config{
		DatabaseID:      cfg.DatabaseID,
		ChainFilePrefix: chainFile,
		DataFile:        storageDSN.Format(),
		Genesis:         genesis,
		Peers:           peers,

		// currently sqlchain package only use Server.ID as node id
		MuxService: cfg.ChainMux,
		Server:     db.nodeID,

		Period:            conf.GConf.SQLChainPeriod,
		Tick:              conf.GConf.SQLChainTick,
		QueryTTL:          conf.GConf.SQLChainTTL,
		LastBillingHeight: cfg.LastBillingHeight,
		UpdatePeriod:      cfg.UpdateBlockCount,
		IsolationLevel:    cfg.IsolationLevel,
	}
	if db.chain, err = sqlchain.NewChain(chainCfg); err != nil {
		return
	}
	if err = db.chain.Start(); err != nil {
		return
	}

	// init kayak config
	kayakWalPath := filepath.Join(cfg.DataDir, KayakWalFileName)
	if db.kayakWal, err = kl.NewLevelDBWal(kayakWalPath); err != nil {
		err = errors.Wrap(err, "init kayak log pool failed")
		return
	}

	db.kayakConfig = &kt.RuntimeConfig{
		Handler:          db,
		PrepareThreshold: PrepareThreshold,
		CommitThreshold:  CommitThreshold,
		PrepareTimeout:   PrepareTimeout,
		CommitTimeout:    CommitTimeout,
		LogWaitTimeout:   LogWaitTimeout,
		Peers:            peers,
		Wal:              db.kayakWal,
		NodeID:           db.nodeID,
		InstanceID:       string(db.dbID),
		ServiceName:      DBKayakRPCName,
		ApplyMethodName:  DBKayakApplyMethodName,
		FetchMethodName:  DBKayakFetchMethodName,
	}

	// create kayak runtime
	if db.kayakRuntime, err = kayak.NewRuntime(db.kayakConfig); err != nil {
		return
	}

	// register kayak runtime rpc
	db.mux.register(db.dbID, db.kayakRuntime)

	// start kayak runtime
	db.kayakRuntime.Start()

	// init sequence eviction processor
	go db.evictSequences()

	return
}

// Query defines database query interface.
func (db *Database) Query(request *types.Request) (response *types.Response, err error) {
	// Just need to verify signature in db.saveAck
	//if err = request.Verify(); err != nil {
	//	return
	//}

	var (
		isSlowQuery uint32
		tracker     *x.QueryTracker
		tmStart     = time.Now()
	)

	// log the query if the underlying storage layer take too long to response
	slowQueryTimer := time.AfterFunc(db.cfg.SlowQueryTime, func() {
		// mark as slow query
		atomic.StoreUint32(&isSlowQuery, 1)
		db.logSlow(request, false, tmStart)
	})
	defer slowQueryTimer.Stop()
	defer func() {
		if atomic.LoadUint32(&isSlowQuery) == 1 {
			// slow query
			db.logSlow(request, true, tmStart)
		}
	}()

	switch request.Header.QueryType {
	case types.ReadQuery:
		if tracker, response, err = db.chain.Query(request, false); err != nil {
			err = errors.Wrap(err, "failed to query read query")
			return
		}
	case types.WriteQuery:
		if db.cfg.UseEventualConsistency {
			// reset context
			request.SetContext(context.Background())
			if tracker, response, err = db.chain.Query(request, true); err != nil {
				err = errors.Wrap(err, "failed to execute with eventual consistency")
				return
			}
		} else {
			if tracker, response, err = db.writeQuery(request); err != nil {
				err = errors.Wrap(err, "failed to execute")
				return
			}
		}
	default:
		// TODO(xq262144): verbose errors with custom error structure
		return nil, errors.Wrap(ErrInvalidRequest, "invalid query type")
	}

	response.Header.ResponseAccount = db.accountAddr

	// build hash
	if err = response.BuildHash(); err != nil {
		err = errors.Wrap(err, "failed to build response hash")
		return
	}

	if err = db.chain.AddResponse(&response.Header); err != nil {
		log.WithError(err).Debug("failed to add response to index")
		return
	}
	tracker.UpdateResp(response)

	return
}

func (db *Database) logSlow(request *types.Request, isFinished bool, tmStart time.Time) {
	if request == nil {
		return
	}

	// sample the queries
	querySample := ""

	for _, q := range request.Payload.Queries {
		if len(querySample) < SlowQuerySampleSize {
			querySample += "; "
			querySample += q.Pattern
		} else {
			break
		}
	}

	if len(querySample) >= SlowQuerySampleSize {
		querySample = querySample[:SlowQuerySampleSize-3]
		querySample += "..."
	}

	log.WithFields(log.Fields{
		"finished": isFinished,
		"db":       request.Header.DatabaseID,
		"req_time": request.Header.Timestamp.String(),
		"req_node": request.Header.NodeID,
		"count":    request.Header.BatchCount,
		"type":     request.Header.QueryType.String(),
		"sample":   querySample,
		"start":    tmStart.String(),
		"elapsed":  time.Now().Sub(tmStart).String(),
	}).Error("slow query detected")
}

// Ack defines client response ack interface.
func (db *Database) Ack(ack *types.Ack) (err error) {
	// Just need to verify signature in db.saveAck
	//if err = ack.Verify(); err != nil {
	//	return
	//}

	return db.saveAck(&ack.Header)
}

// Shutdown stop database handles and stop service the database.
func (db *Database) Shutdown() (err error) {
	if db.kayakRuntime != nil {
		// shutdown, stop kayak
		if err = db.kayakRuntime.Shutdown(); err != nil {
			return
		}

		// unregister
		db.mux.unregister(db.dbID)
	}

	if db.kayakWal != nil {
		// shutdown, stop kayak
		db.kayakWal.Close()
	}

	if db.chain != nil {
		// stop chain
		if err = db.chain.Stop(); err != nil {
			return
		}
	}

	if db.connSeqEvictCh != nil {
		// stop connection sequence evictions
		select {
		case _, ok := <-db.connSeqEvictCh:
			if ok {
				close(db.connSeqEvictCh)
			}
		default:
			close(db.connSeqEvictCh)
		}
	}

	return
}

// Destroy stop database instance and destroy all data/meta.
func (db *Database) Destroy() (err error) {
	if err = db.Shutdown(); err != nil {
		return
	}

	// TODO(xq262144): remove database files, now simply remove whole root dir
	os.RemoveAll(db.cfg.DataDir)

	return
}

func (db *Database) writeQuery(request *types.Request) (tracker *x.QueryTracker, response *types.Response, err error) {
	// check database size first, wal/kayak/chain database size is not included
	if db.cfg.SpaceLimit > 0 {
		path := filepath.Join(db.cfg.DataDir, StorageFileName)
		var statInfo os.FileInfo
		if statInfo, err = os.Stat(path); err != nil {
			if !os.IsNotExist(err) {
				return
			}
		} else {
			if uint64(statInfo.Size()) > db.cfg.SpaceLimit {
				// rejected
				err = ErrSpaceLimitExceeded
				return
			}
		}
	}

	// call kayak runtime Process
	var result interface{}
	if result, _, err = db.kayakRuntime.Apply(request.GetContext(), request); err != nil {
		err = errors.Wrap(err, "apply failed")
		return
	}

	var (
		tr *TrackerAndResponse
		ok bool
	)
	if tr, ok = (result).(*TrackerAndResponse); !ok {
		err = errors.Wrap(err, "invalid response type")
		return
	}
	tracker = tr.Tracker
	response = tr.Response
	return
}

func (db *Database) saveAck(ackHeader *types.SignedAckHeader) (err error) {
	return db.chain.VerifyAndPushAckedQuery(ackHeader)
}

func getLocalTime() time.Time {
	return time.Now().UTC()
}
