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
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/kayak"
	ka "github.com/CovenantSQL/CovenantSQL/kayak/api"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/sqlchain"
	"github.com/CovenantSQL/CovenantSQL/sqlchain/storage"
	ct "github.com/CovenantSQL/CovenantSQL/sqlchain/types"
	"github.com/CovenantSQL/CovenantSQL/utils"
	wt "github.com/CovenantSQL/CovenantSQL/worker/types"
)

const (
	// StorageFileName defines storage file name of database instance.
	StorageFileName = "storage.db3"

	// SQLChainFileName defines sqlchain storage file name.
	SQLChainFileName = "chain.db"

	// MaxRecordedConnectionSequences defines the max connection slots to anti reply attack.
	MaxRecordedConnectionSequences = 1000
)

// Database defines a single database instance in worker runtime.
type Database struct {
	cfg            *DBConfig
	dbID           proto.DatabaseID
	storage        *storage.Storage
	kayakRuntime   *kayak.Runtime
	kayakConfig    kayak.Config
	connSeqs       sync.Map
	connSeqEvictCh chan uint64
	chain          *sqlchain.Chain
}

// NewDatabase create a single database instance using config.
func NewDatabase(cfg *DBConfig, peers *kayak.Peers, genesisBlock *ct.Block) (db *Database, err error) {
	// ensure dir exists
	if err = os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return
	}

	if peers == nil || genesisBlock == nil {
		err = ErrInvalidDBConfig
		return
	}

	// init database
	db = &Database{
		cfg:            cfg,
		dbID:           cfg.DatabaseID,
		connSeqEvictCh: make(chan uint64, 1),
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

			// close storage
			if db.storage != nil {
				db.storage.Close()
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

	if db.storage, err = storage.New(storageDSN.Format()); err != nil {
		return
	}

	// init chain
	var nodeID proto.NodeID
	chainFile := filepath.Join(cfg.DataDir, SQLChainFileName)
	if nodeID, err = kms.GetLocalNodeID(); err != nil {
		return
	}

	// TODO(xq262144): make sqlchain config use of global config object
	chainCfg := &sqlchain.Config{
		DatabaseID: cfg.DatabaseID,
		DataFile:   chainFile,
		Genesis:    genesisBlock,
		Peers:      peers,

		// TODO(xq262144): should refactor server/node definition to conf/proto package
		// currently sqlchain package only use Server.ID as node id
		MuxService: cfg.ChainMux,
		Server: &kayak.Server{
			ID: nodeID,
		},

		// TODO(xq262144): currently using fixed period/resolution from sqlchain test case
		Period:   60 * time.Second,
		Tick:     10 * time.Second,
		QueryTTL: 10,
	}
	if db.chain, err = sqlchain.NewChain(chainCfg); err != nil {
		return
	} else if err = db.chain.Start(); err != nil {
		return
	}

	// init kayak config
	options := ka.NewDefaultTwoPCOptions().WithTransportID(string(cfg.DatabaseID))
	db.kayakConfig = ka.NewTwoPCConfigWithOptions(cfg.DataDir, cfg.KayakMux, db, options)

	// create kayak runtime
	if db.kayakRuntime, err = ka.NewTwoPCKayak(peers, db.kayakConfig); err != nil {
		return
	}

	// init kayak runtime
	if err = db.kayakRuntime.Init(); err != nil {
		return
	}

	// init sequence eviction processor
	go db.evictSequences()

	return
}

// UpdatePeers defines peers update query interface.
func (db *Database) UpdatePeers(peers *kayak.Peers) (err error) {
	if err = db.kayakRuntime.UpdatePeers(peers); err != nil {
		return
	}

	return db.chain.UpdatePeers(peers)
}

// Query defines database query interface.
func (db *Database) Query(request *wt.Request) (response *wt.Response, err error) {
	if err = request.Verify(); err != nil {
		return
	}

	switch request.Header.QueryType {
	case wt.ReadQuery:
		return db.readQuery(request)
	case wt.WriteQuery:
		return db.writeQuery(request)
	default:
		// TODO(xq262144): verbose errors with custom error structure
		return nil, ErrInvalidRequest
	}
}

// Ack defines client response ack interface.
func (db *Database) Ack(ack *wt.Ack) (err error) {
	if err = ack.Verify(); err != nil {
		return
	}

	return db.saveAck(&ack.Header)
}

// Shutdown stop database handles and stop service the database.
func (db *Database) Shutdown() (err error) {
	if db.kayakRuntime != nil {
		// shutdown, stop kayak
		if err = db.kayakRuntime.Shutdown(); err != nil {
			return
		}
	}

	if db.chain != nil {
		// stop chain
		if err = db.chain.Stop(); err != nil {
			return
		}
	}

	if db.storage != nil {
		// stop storage
		if err = db.storage.Close(); err != nil {
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

func (db *Database) writeQuery(request *wt.Request) (response *wt.Response, err error) {
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
	var buf *bytes.Buffer
	if buf, err = utils.EncodeMsgPack(request); err != nil {
		return
	}

	var logOffset uint64
	logOffset, err = db.kayakRuntime.Apply(buf.Bytes())

	if err != nil {
		return
	}

	return db.buildQueryResponse(request, logOffset, []string{}, []string{}, [][]interface{}{})
}

func (db *Database) readQuery(request *wt.Request) (response *wt.Response, err error) {
	// call storage query directly
	// TODO(xq262144): add timeout logic basic of client options
	var columns, types []string
	var data [][]interface{}

	columns, types, data, err = db.storage.Query(context.Background(), convertQuery(request.Payload.Queries))
	if err != nil {
		return
	}

	return db.buildQueryResponse(request, 0, columns, types, data)
}

func (db *Database) buildQueryResponse(request *wt.Request, offset uint64,
	columns []string, types []string, data [][]interface{}) (response *wt.Response, err error) {
	// build response
	response = new(wt.Response)
	response.Header.Request = request.Header
	if response.Header.NodeID, err = kms.GetLocalNodeID(); err != nil {
		return
	}
	response.Header.LogOffset = offset
	response.Header.Timestamp = getLocalTime()
	response.Header.RowCount = uint64(len(data))
	if response.Header.Signee, err = getLocalPubKey(); err != nil {
		return
	}

	// set payload
	response.Payload.Columns = columns
	response.Payload.DeclTypes = types
	response.Payload.Rows = make([]wt.ResponseRow, len(data))

	for i, d := range data {
		response.Payload.Rows[i].Values = d
	}

	// sign fields
	var privateKey *asymmetric.PrivateKey
	if privateKey, err = getLocalPrivateKey(); err != nil {
		return
	}
	if err = response.Sign(privateKey); err != nil {
		return
	}

	// record response for future ack process
	err = db.saveResponse(&response.Header)
	return
}

func (db *Database) saveResponse(respHeader *wt.SignedResponseHeader) (err error) {
	return db.chain.VerifyAndPushResponsedQuery(respHeader)
}

func (db *Database) saveAck(ackHeader *wt.SignedAckHeader) (err error) {
	return db.chain.VerifyAndPushAckedQuery(ackHeader)
}

func getLocalTime() time.Time {
	return time.Now().UTC()
}

func getLocalPubKey() (pubKey *asymmetric.PublicKey, err error) {
	return kms.GetLocalPublicKey()
}

func getLocalPrivateKey() (privateKey *asymmetric.PrivateKey, err error) {
	return kms.GetLocalPrivateKey()
}

func convertQuery(inQuery []wt.Query) (outQuery []storage.Query) {
	outQuery = make([]storage.Query, len(inQuery))
	for i, q := range inQuery {
		outQuery[i] = storage.Query(q)
	}
	return
}
