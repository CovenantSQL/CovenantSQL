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

package client

import (
	"database/sql"
	"database/sql/driver"
	"path/filepath"

	bp "github.com/CovenantSQL/CovenantSQL/blockproducer"
	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	wt "github.com/CovenantSQL/CovenantSQL/worker/types"
)

const (
	// PubKeyStorePath defines public cache store.
	PubKeyStorePath = "./public.keystore"
)

func init() {
	sql.Register("covenantsql", new(covenantSQLDriver))
}

// covenantSQLDriver implements sql.Driver interface.
type covenantSQLDriver struct {
}

// Open returns new db connection.
func (d *covenantSQLDriver) Open(dsn string) (conn driver.Conn, err error) {
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
	if err = kms.InitLocalKeyPair(conf.GConf.PrivateKeyFile, masterKey); err != nil {
		return
	}

	// ping block producer to register node
	err = registerNode()

	return
}

// Create send create database operation to block producer.
func Create(meta ResourceMeta) (dsn string, err error) {
	req := new(bp.CreateDatabaseRequest)
	req.Header.ResourceMeta = wt.ResourceMeta(meta)
	if req.Header.Signee, err = kms.GetLocalPublicKey(); err != nil {
		return
	}
	var privateKey *asymmetric.PrivateKey
	if privateKey, err = kms.GetLocalPrivateKey(); err != nil {
		return
	}
	if err = req.Sign(privateKey); err != nil {
		return
	}
	res := new(bp.CreateDatabaseResponse)

	if err = requestBP(route.BPDBCreateDatabase, req, res); err != nil {
		return
	}
	if err = res.Verify(); err != nil {
		return
	}

	cfg := NewConfig()
	cfg.DatabaseID = string(res.Header.InstanceMeta.DatabaseID)
	dsn = cfg.FormatDSN()

	return
}

// Drop send drop database operation to block producer.
func Drop(dsn string) (err error) {
	var cfg *Config
	if cfg, err = ParseDSN(dsn); err != nil {
		return
	}

	req := new(bp.DropDatabaseRequest)
	req.Header.DatabaseID = proto.DatabaseID(cfg.DatabaseID)
	if req.Header.Signee, err = kms.GetLocalPublicKey(); err != nil {
		return
	}
	var privateKey *asymmetric.PrivateKey
	if privateKey, err = kms.GetLocalPrivateKey(); err != nil {
		return
	}
	if err = req.Sign(privateKey); err != nil {
		return
	}
	res := new(bp.DropDatabaseResponse)
	err = requestBP(route.BPDBDropDatabase, req, res)

	return
}

func requestBP(method route.RemoteFunc, request interface{}, response interface{}) (err error) {
	var bpNodeID proto.NodeID
	if bpNodeID, err = rpc.GetCurrentBP(); err != nil {
		return
	}

	return rpc.NewCaller().CallNode(bpNodeID, method.String(), request, response)
}

func registerNode() (err error) {
	var nodeID proto.NodeID

	if nodeID, err = kms.GetLocalNodeID(); err != nil {
		return
	}

	var nodeInfo *proto.Node
	if nodeInfo, err = kms.GetNodeInfo(nodeID); err != nil {
		return
	}

	err = rpc.PingBP(nodeInfo, conf.GConf.BP.NodeID)

	return
}
