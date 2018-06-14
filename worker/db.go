/*
 * Copyright 2018 The ThunderDB Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the “License”);
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an “AS IS” BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package worker

import (
	"context"
	"os"
	"path/filepath"

	"gitlab.com/thunderdb/ThunderDB/crypto/asymmetric"
	"gitlab.com/thunderdb/ThunderDB/kayak"
	"gitlab.com/thunderdb/ThunderDB/proto"
	"gitlab.com/thunderdb/ThunderDB/sqlchain/storage"
	"gitlab.com/thunderdb/ThunderDB/twopc"
)

var (
	storageFile = "db.sqlite"
)

// Database defines a single database instance in worker runtime.
type Database struct {
	cfg          *Config
	dbID         proto.DatabaseID
	storage      *storage.Storage
	kayakRuntime *kayak.Runtime
	kayakConfig  *kayak.TwoPCConfig
	privKey      *asymmetric.PrivateKey
}

// NewDatabase create a single database instance using config.
func NewDatabase(cfg *Config) (db *Database, err error) {
	// ensure dir exists
	if err = os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return
	}

	// init underlying storage
	var st *storage.Storage
	if st, err = storage.New(filepath.Join(cfg.DataDir, storageFile)); err != nil {
		return
	}

	// init kayak transport
	// TODO(xq262144)

	// init kayak config
	// TODO(xq262144)

	// create kayak runtime
	// TODO(xq262144)

	db = &Database{
		cfg:     cfg,
		dbID:    cfg.DatabaseID,
		storage: st,
	}

	return
}

// Query defines database read-only query interface.
func (db *Database) Query() (err error) {
	// TODO(xq262144)
	return
}

// Execute defines database write-only query interface.
func (db *Database) Execute() (err error) {
	// TODO(xq262144)
	return
}

// Shutdown stop database handles and stop service the database.
func (db *Database) Shutdown() (err error) {
	// shutdown
	return
}

// Prepare implements twopc.Worker.Prepare.
func (db *Database) Prepare(ctx context.Context, wb twopc.WriteBatch) (err error) {
	// wrap storage with signature check
	var log *storage.ExecLog
	if log, err = db.convertRequest(wb); err != nil {
		return
	}
	return db.storage.Prepare(ctx, log)
}

// Commit implements twopc.Worker.Commmit.
func (db *Database) Commit(ctx context.Context, wb twopc.WriteBatch) (err error) {
	// wrap storage with signature check
	var log *storage.ExecLog
	if log, err = db.convertRequest(wb); err != nil {
		return
	}
	return db.storage.Commit(ctx, log)
}

// Rollback implements twopc.Worker.Rollback.
func (db *Database) Rollback(ctx context.Context, wb twopc.WriteBatch) (err error) {
	// wrap storage with signature check
	var log *storage.ExecLog
	if log, err = db.convertRequest(wb); err != nil {
		return
	}
	return db.storage.Rollback(ctx, log)
}

func (db *Database) convertRequest(wb twopc.WriteBatch) (log *storage.ExecLog, err error) {
	var req *Request
	var ok bool

	// type assert
	if req, ok = wb.(*Request); !ok {
		// invalid request data
		err = ErrInvalidRequest
		return
	}

	// verify
	if err = req.Verify(); err != nil {
		req = nil
		return
	}

	// convert
	log = new(storage.ExecLog)
	log.ConnectionID = req.Header.ConnectionID
	log.SeqNo = req.Header.SeqNo
	log.Timestamp = req.Header.Timestamp.UnixNano()
	log.Queries = make([]string, len(req.Payload.Queries))
	copy(log.Queries, req.Payload.Queries)

	return
}
