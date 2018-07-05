/*
 * Copyright 2018 The ThunderDB Authors.
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

	"gitlab.com/thunderdb/ThunderDB/crypto/asymmetric"
	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
	"gitlab.com/thunderdb/ThunderDB/crypto/kms"
	"gitlab.com/thunderdb/ThunderDB/kayak"
	ka "gitlab.com/thunderdb/ThunderDB/kayak/api"
	"gitlab.com/thunderdb/ThunderDB/proto"
	"gitlab.com/thunderdb/ThunderDB/sqlchain"
	"gitlab.com/thunderdb/ThunderDB/sqlchain/storage"
	ct "gitlab.com/thunderdb/ThunderDB/sqlchain/types"
	"gitlab.com/thunderdb/ThunderDB/utils"
	wt "gitlab.com/thunderdb/ThunderDB/worker/types"
)

const (
	// StorageFileName defines storage file name of database instance.
	StorageFileName = "storage.db3"

	// SQLChainFileName defines sqlchain storage file name.
	SQLChainFileName = "chain.db"
)

// Database defines a single database instance in worker runtime.
type Database struct {
	cfg          *DBConfig
	dbID         proto.DatabaseID
	storage      *storage.Storage
	kayakRuntime *kayak.Runtime
	kayakConfig  kayak.Config
	connSeqs     sync.Map
	chain        *sqlchain.Chain
}

// NewDatabase create a single database instance using config.
func NewDatabase(cfg *DBConfig, peers *kayak.Peers, genesisBlock *ct.Block) (db *Database, err error) {
	// ensure dir exists
	if err = os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return
	}

	// init database
	db = &Database{
		cfg:  cfg,
		dbID: cfg.DatabaseID,
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
				// TODO(xq262144), close chain
			}

			// close storage
			if db.storage != nil {
				db.storage.Close()
			}
		}
	}()

	// init storage
	storageFile := filepath.Join(cfg.DataDir, StorageFileName)
	if db.storage, err = storage.New(storageFile); err != nil {
		return
	}

	// init chain
	var nodeID proto.NodeID
	chainFile := filepath.Join(cfg.DataDir, SQLChainFileName)
	if nodeID, err = getLocalNodeID(); err != nil {
		return
	}

	// TODO(xq262144), make sqlchain config use of global config object
	chainCfg := &sqlchain.Config{
		DataDir: chainFile,
		Genesis: genesisBlock,
		Peers:   peers,

		// TODO(xq262144), should refactor server/node definition to conf/proto package
		// currently sqlchain package only use Server.ID as node id
		Server: &kayak.Server{
			ID: nodeID,
		},

		// TODO(xq261244), currently using fixed period/resolution from sqlchain test case
		Period:   1 * time.Second,
		Tick:     100 * time.Millisecond,
		QueryTTL: 10,
	}
	if db.chain, err = sqlchain.NewChain(chainCfg); err != nil {
		return
	}

	// init kayak config
	options := ka.NewDefaultTwoPCOptions().WithTransportID(string(cfg.DatabaseID))
	db.kayakConfig = ka.NewTwoPCConfigWithOptions(cfg.DataDir, cfg.MuxService, db, options)

	// create kayak runtime
	if db.kayakRuntime, err = ka.NewTwoPCKayak(peers, db.kayakConfig); err != nil {
		return
	}

	// init kayak runtime
	if err = db.kayakRuntime.Init(); err != nil {
		return
	}

	return
}

// UpdatePeers defines peers update query interface.
func (db *Database) UpdatePeers(peers *kayak.Peers) error {
	return db.kayakRuntime.UpdatePeers(peers)
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
		// TODO(xq262144) verbose errors with custom error structure
		return nil, ErrInvalidRequest
	}
}

// Ack defines client response ack interface.
func (db *Database) Ack(ack *wt.Ack) (err error) {
	if err = ack.Verify(); err != nil {
		return
	}

	return db.saveAck(ack)
}

// Shutdown stop database handles and stop service the database.
func (db *Database) Shutdown() (err error) {
	// shutdown, stop kayak
	if err = db.kayakRuntime.Shutdown(); err != nil {
		return
	}

	// stop chain
	// TODO(xq262144), stop chain if possible

	// stop storage
	if err = db.storage.Close(); err != nil {
		return
	}

	return
}

// Destroy stop database instance and destroy all data/meta.
func (db *Database) Destroy() (err error) {
	if err = db.Shutdown(); err != nil {
		return
	}

	// TODO(xq262144), remove database files, now simply remove whole root dir
	os.RemoveAll(db.cfg.DataDir)

	return
}

func (db *Database) writeQuery(request *wt.Request) (response *wt.Response, err error) {
	// call kayak runtime Process
	var buf *bytes.Buffer
	if buf, err = utils.EncodeMsgPack(request); err != nil {
		return
	}

	err = db.kayakRuntime.Apply(buf.Bytes())

	if err != nil {
		return
	}

	return db.buildQueryResponse(request, []string{}, []string{}, [][]interface{}{})
}

func (db *Database) readQuery(request *wt.Request) (response *wt.Response, err error) {
	// call storage query directly
	// TODO(xq262144), add timeout logic basic of client options
	var columns, types []string
	var data [][]interface{}

	columns, types, data, err = db.storage.Query(context.Background(), request.Payload.Queries)
	if err != nil {
		return
	}

	return db.buildQueryResponse(request, columns, types, data)
}

func (db *Database) buildQueryResponse(request *wt.Request, columns []string, types []string, data [][]interface{}) (response *wt.Response, err error) {
	// build response
	response = new(wt.Response)
	response.Header.Request = request.Header
	if response.Header.NodeID, err = getLocalNodeID(); err != nil {
		return
	}
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
	err = db.saveRequest(request)

	return
}

// TODO(xq262144), following are function to be filled and revised for integration in the future

func (db *Database) saveRequest(request *wt.Request) (err error) {
	// TODO(xq262144), to be integrated with sqlchain
	return
}

func (db *Database) saveAck(ack *wt.Ack) (err error) {
	// TODO(xq262144), to be integrated with sqlchain
	return
}

func getLocalTime() time.Time {
	// TODO(xq262144), to use same time coordination logic with sqlchain
	return time.Now().UTC()
}

func getLocalNodeID() (nodeID proto.NodeID, err error) {
	// TODO(xq262144), to use refactored node id interface by kms
	var rawNodeID []byte
	if rawNodeID, err = kms.GetLocalNodeID(); err != nil {
		return
	}
	var h *hash.Hash
	if h, err = hash.NewHash(rawNodeID); err != nil {
		return
	}
	nodeID = proto.NodeID(h.String())
	return
}

func getLocalPubKey() (pubKey *asymmetric.PublicKey, err error) {
	return kms.GetLocalPublicKey()
}

func getLocalPrivateKey() (privateKey *asymmetric.PrivateKey, err error) {
	return kms.GetLocalPrivateKey()
}
