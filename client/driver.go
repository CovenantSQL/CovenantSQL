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

package client

import (
	"database/sql"
	"database/sql/driver"
	"path/filepath"

	bp "gitlab.com/thunderdb/ThunderDB/blockproducer"
	"gitlab.com/thunderdb/ThunderDB/conf"
	"gitlab.com/thunderdb/ThunderDB/crypto/kms"
	"gitlab.com/thunderdb/ThunderDB/pow/cpuminer"
	"gitlab.com/thunderdb/ThunderDB/proto"
	"gitlab.com/thunderdb/ThunderDB/route"
	"gitlab.com/thunderdb/ThunderDB/rpc"
	wt "gitlab.com/thunderdb/ThunderDB/worker/types"
)

const (
	// PubKeyStorePath defines public cache store.
	PubKeyStorePath = "./public.keystore"
)

func init() {
	sql.Register("thunderdb", new(thunderDBDriver))
}

// thunderDBDriver implements sql.Driver interface.
type thunderDBDriver struct {
}

// Open returns new db connection.
func (d *thunderDBDriver) Open(dsn string) (conn driver.Conn, err error) {
	var cfg *Config
	if cfg, err = ParseDSN(dsn); err != nil {
		return
	}

	return newConn(cfg)
}

// ResourceMeta defines new database resources requirement descriptions.
type ResourceMeta wt.ResourceMeta

// Init defines init process for client.
func Init(configFile string, masterKey []byte) (err error) {
	// load config
	if conf.GConf, err = conf.LoadConfig(configFile); err != nil {
		return
	}
	pubKeyFilePath := filepath.Join(conf.GConf.WorkingRoot, PubKeyStorePath)
	route.InitKMS(pubKeyFilePath)
	err = kms.InitLocalKeyPair(conf.GConf.PrivateKeyFile, masterKey)
	return
}

// Create send create database operation to block producer.
func Create(meta ResourceMeta) (dsn string, err error) {
	req := &bp.CreateDatabaseRequest{
		ResourceMeta: wt.ResourceMeta(meta),
	}
	res := new(bp.CreateDatabaseResponse)

	if err = requestBP(bp.DBServiceName+".CreateDatabase", req, res); err != nil {
		return
	}

	cfg := NewConfig()
	cfg.DatabaseID = res.InstanceMeta.DatabaseID
	dsn = cfg.FormatDSN()

	return
}

// Drop send drop database operation to block producer.
func Drop(dsn string) (err error) {
	var cfg *Config
	if cfg, err = ParseDSN(dsn); err != nil {
		return
	}

	req := &bp.DropDatabaseRequest{
		DatabaseID: cfg.DatabaseID,
	}
	res := new(bp.DropDatabaseResponse)
	err = requestBP(bp.DBServiceName+".DropDatabase", req, res)

	return
}

func requestBP(method string, request interface{}, response interface{}) (err error) {
	// TODO(xq262144), unify block producer calls
	// get bp node
	var nonce *cpuminer.Uint256
	nonce, err = kms.GetLocalNonce()
	var bps []proto.NodeID
	bps = route.GetBPs()

	// choose bp node by nonce
	bpIdx := int(nonce.A % uint64(len(bps)))

	return rpc.NewCaller().CallNode(bps[bpIdx], method, request, response)
}
