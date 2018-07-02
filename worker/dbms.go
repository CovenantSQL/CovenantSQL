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
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"gitlab.com/thunderdb/ThunderDB/utils"
)

const (
	// DBKayakRPCName defines rpc service name of database internal consensus.
	DBKayakRPCName = "DBC" // aka. database consensus

	// DBServiceRPCName defines rpc service name of database external query api.
	DBServiceRPCName = "DBS" // aka. database service

	// DBMetaFileName defines dbms meta file name.
	DBMetaFileName = "db.meta"
)

// DBMS defines a database management instance.
type DBMS struct {
	cfg      *DBMSConfig
	metaLock sync.Mutex
	meta     *DBMSMeta
}

// NewDBMS returns new database management instance.
func NewDBMS(cfg *DBMSConfig) (dbms *DBMS, err error) {
	dbms = &DBMS{
		cfg: cfg,
	}

	return
}

func (dbms *DBMS) readMeta() (err error) {
	dbms.metaLock.Lock()
	defer dbms.metaLock.Unlock()

	filePath := filepath.Join(dbms.cfg.RootDir, DBMetaFileName)
	dbms.meta = new(DBMSMeta)

	var fileContent []byte

	if fileContent, err = ioutil.ReadFile(filePath); err != nil {
		// if not exists
		if err == os.ErrNotExist {
			// new without meta
			err = nil
			return
		}

		return
	}

	err = utils.DecodeMsgPack(fileContent, dbms.meta)

	return
}

func (dbms *DBMS) writeMeta() (err error) {
	dbms.metaLock.Lock()
	defer dbms.metaLock.Unlock()

	if dbms.meta == nil {
		dbms.meta = new(DBMSMeta)
	}

	var buf *bytes.Buffer
	if buf, err = utils.EncodeMsgPack(dbms.meta); err != nil {
		return
	}

	filePath := filepath.Join(dbms.cfg.RootDir, DBMetaFileName)
	err = ioutil.WriteFile(filePath, buf.Bytes(), 0644)

	return
}

// Init defines dbms init logic.
func (dbms *DBMS) Init() (err error) {
	// read meta
	err = dbms.readMeta()

	return
}

func (dbms *DBMS) create() (err error) {
	// called by rpc to call create database logic
	return
}

func (dbms *DBMS) drop() (err error) {
	// called by rpc to call drop database logic
	return
}

func (dbms *DBMS) update() (err error) {
	// called by rpc to call drop database logic
	return
}

// Shutdown defines dbms shutdown logic.
func (dbms *DBMS) Shutdown() (err error) {
	// shutdown databases
	// TODO(xq262144), shutdown databases

	// persist meta
	err = dbms.writeMeta()

	return
}
