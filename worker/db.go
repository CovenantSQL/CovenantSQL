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
	"sync"
	"time"

	"gitlab.com/thunderdb/ThunderDB/crypto/asymmetric"
	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
	"gitlab.com/thunderdb/ThunderDB/crypto/kms"
	"gitlab.com/thunderdb/ThunderDB/kayak"
	ka "gitlab.com/thunderdb/ThunderDB/kayak/api"
	"gitlab.com/thunderdb/ThunderDB/proto"
	"gitlab.com/thunderdb/ThunderDB/sqlchain/storage"
	"gitlab.com/thunderdb/ThunderDB/utils"
)

// Database defines a single database instance in worker runtime.
type Database struct {
	cfg          *DBConfig
	dbID         proto.DatabaseID
	storage      *storage.Storage
	kayakRuntime *kayak.Runtime
	kayakConfig  kayak.Config
	connSeqs     sync.Map
}

// NewDatabase create a single database instance using config.
func NewDatabase(cfg *DBConfig, peers *kayak.Peers, st *storage.Storage) (db *Database, err error) {
	// ensure dir exists
	if err = os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return
	}

	// init database
	db = &Database{
		cfg:     cfg,
		dbID:    cfg.DatabaseID,
		storage: st,
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

// Query defines database query interface.
func (db *Database) Query(request *Request) (response *Response, err error) {
	if err = request.Verify(); err != nil {
		return
	}

	switch request.Header.QueryType {
	case ReadQuery:
		return db.readQuery(request)
	case WriteQuery:
		return db.writeQuery(request)
	default:
		// TODO(xq262144) verbose errors with custom error structure
		return nil, ErrInvalidRequest
	}
}

// Ack defines client response ack interface.
func (db *Database) Ack(ack *Ack) (err error) {
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

func (db *Database) writeQuery(request *Request) (response *Response, err error) {
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

func (db *Database) readQuery(request *Request) (response *Response, err error) {
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

func (db *Database) buildQueryResponse(request *Request, columns []string, types []string, data [][]interface{}) (response *Response, err error) {
	// build response
	response = new(Response)
	response.Header.Request = request.Header
	if response.Header.NodeID, err = db.getLocalNodeID(); err != nil {
		return
	}
	response.Header.Timestamp = db.getLocalTime()
	response.Header.RowCount = uint64(len(data))
	if response.Header.Signee, err = db.getLocalPubKey(); err != nil {
		return
	}

	// set payload
	response.Payload.Columns = columns
	response.Payload.DeclTypes = types
	response.Payload.Rows = make([]ResponseRow, len(data))

	for i, d := range data {
		response.Payload.Rows[i].Values = d
	}

	// sign fields
	var privateKey *asymmetric.PrivateKey
	if privateKey, err = db.getLocalPrivateKey(); err != nil {
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

func (db *Database) saveRequest(request *Request) (err error) {
	// TODO(xq262144), to be integrated with sqlchain
	return
}

func (db *Database) saveAck(ack *Ack) (err error) {
	// TODO(xq262144), to be integrated with sqlchain
	return
}

func (db *Database) getLocalTime() time.Time {
	// TODO(xq262144), to use same time coordination logic with sqlchain
	return time.Now().UTC()
}

func (db *Database) getLocalNodeID() (nodeID proto.NodeID, err error) {
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

func (db *Database) getLocalPubKey() (pubKey *asymmetric.PublicKey, err error) {
	return kms.GetLocalPublicKey()
}

func (db *Database) getLocalPrivateKey() (privateKey *asymmetric.PrivateKey, err error) {
	return kms.GetLocalPrivateKey()
}
