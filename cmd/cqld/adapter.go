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

package main

import (
	"bytes"
	"context"
	"database/sql"
	"os"

	bp "github.com/CovenantSQL/CovenantSQL/blockproducer"
	"github.com/CovenantSQL/CovenantSQL/consistent"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/kayak"
	kt "github.com/CovenantSQL/CovenantSQL/kayak/types"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/storage"
	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/pkg/errors"
)

const (
	// CmdSet is the command to set node
	CmdSet = "set"
	// CmdSetDatabase is the command to set database
	CmdSetDatabase = "set_database"
	// CmdDeleteDatabase is the command to del database
	CmdDeleteDatabase = "delete_database"
)

// LocalStorage holds consistent and storage struct
type LocalStorage struct {
	consistent *consistent.Consistent
	*storage.Storage
}

type compiledLog struct {
	cmdType   string
	queries   []storage.Query
	nodeToSet *proto.Node
}

func initStorage(dbFile string) (stor *LocalStorage, err error) {
	var st *storage.Storage
	if st, err = storage.New(dbFile); err != nil {
		return
	}
	//TODO(auxten): try BLOB for `id`, to performance test
	_, err = st.Exec(context.Background(), []storage.Query{
		{
			Pattern: "CREATE TABLE IF NOT EXISTS `dht` (`id` TEXT NOT NULL PRIMARY KEY, `node` BLOB);",
		},
		{
			Pattern: "CREATE TABLE IF NOT EXISTS `databases` (`id` TEXT NOT NULL PRIMARY KEY, `meta` BLOB);",
		},
	})
	if err != nil {
		wd, _ := os.Getwd()
		log.WithField("file", utils.FJ(wd, dbFile)).WithError(err).Error("create dht table failed")
		return
	}

	stor = &LocalStorage{
		Storage: st,
	}

	return
}

// EncodePayload implements kayak.types.Handler.EncodePayload.
func (s *LocalStorage) EncodePayload(request interface{}) (data []byte, err error) {
	var buf *bytes.Buffer
	if buf, err = utils.EncodeMsgPack(request); err != nil {
		err = errors.Wrap(err, "encode kayak payload failed")
		return
	}

	data = buf.Bytes()
	return
}

// DecodePayload implements kayak.types.Handler.DecodePayload.
func (s *LocalStorage) DecodePayload(data []byte) (request interface{}, err error) {
	var kp *KayakPayload

	if err = utils.DecodeMsgPack(data, &kp); err != nil {
		err = errors.Wrap(err, "decode kayak payload failed")
		return
	}

	request = kp
	return
}

// Check implements kayak.types.Handler.Check.
func (s *LocalStorage) Check(req interface{}) (err error) {
	return nil
}

// Commit implements kayak.types.Handler.Commit.
func (s *LocalStorage) Commit(req interface{}, isLeader bool) (_ interface{}, err error) {
	var kp *KayakPayload
	var cl *compiledLog
	var ok bool

	if kp, ok = req.(*KayakPayload); !ok || kp == nil {
		err = errors.Wrapf(kt.ErrInvalidLog, "invalid kayak payload %#v", req)
		return
	}

	if cl, err = s.compileLog(kp); err != nil {
		err = errors.Wrap(err, "compile log failed")
		return
	}

	if cl.nodeToSet != nil {
		err = route.SetNodeAddrCache(cl.nodeToSet.ID.ToRawNodeID(), cl.nodeToSet.Addr)
		if err != nil {
			log.WithFields(log.Fields{
				"id":   cl.nodeToSet.ID,
				"addr": cl.nodeToSet.Addr,
			}).WithError(err).Error("set node addr cache failed")
		}
		err = kms.SetNode(cl.nodeToSet)
		if err != nil {
			log.WithField("node", cl.nodeToSet).WithError(err).Error("kms set node failed")
		}

		// if s.consistent == nil, it is called during Init. and AddCache will be called by consistent.InitConsistent
		if s.consistent != nil {
			s.consistent.AddCache(*cl.nodeToSet)
		}
	}

	// execute query
	if _, err = s.Storage.Exec(context.Background(), cl.queries); err != nil {
		err = errors.Wrap(err, "execute query in dht database failed")
	}

	return
}

func (s *LocalStorage) compileLog(payload *KayakPayload) (result *compiledLog, err error) {
	switch payload.Command {
	case CmdSet:
		var nodeToSet proto.Node
		err = utils.DecodeMsgPack(payload.Data, &nodeToSet)
		if err != nil {
			log.WithError(err).Error("compileLog: unmarshal node from payload failed")
			return
		}
		query := "INSERT OR REPLACE INTO `dht` (`id`, `node`) VALUES (?, ?);"
		log.Debugf("sql: %#v", query)
		result = &compiledLog{
			cmdType: payload.Command,
			queries: []storage.Query{
				{
					Pattern: query,
					Args: []sql.NamedArg{
						sql.Named("", nodeToSet.ID),
						sql.Named("", payload.Data),
					},
				},
			},
			nodeToSet: &nodeToSet,
		}
	case CmdSetDatabase:
		var instance types.ServiceInstance
		if err = utils.DecodeMsgPack(payload.Data, &instance); err != nil {
			log.WithError(err).Error("compileLog: unmarshal instance meta failed")
			return
		}
		query := "INSERT OR REPLACE INTO `databases` (`id`, `meta`) VALUES (? ,?);"
		result = &compiledLog{
			cmdType: payload.Command,
			queries: []storage.Query{
				{
					Pattern: query,
					Args: []sql.NamedArg{
						sql.Named("", string(instance.DatabaseID)),
						sql.Named("", payload.Data),
					},
				},
			},
		}
	case CmdDeleteDatabase:
		var instance types.ServiceInstance
		if err = utils.DecodeMsgPack(payload.Data, &instance); err != nil {
			log.WithError(err).Error("compileLog: unmarshal instance id failed")
			return
		}
		// TODO(xq262144), should add additional limit 1 after delete clause
		// however, currently the go-sqlite3
		query := "DELETE FROM `databases` WHERE `id` = ?"
		result = &compiledLog{
			cmdType: payload.Command,
			queries: []storage.Query{
				{
					Pattern: query,
					Args: []sql.NamedArg{
						sql.Named("", string(instance.DatabaseID)),
					},
				},
			},
		}
	default:
		err = errors.Errorf("undefined command: %v", payload.Command)
		log.WithError(err).Error("compile log failed")
	}
	return
}

// KayakKVServer holds kayak.Runtime and LocalStorage
type KayakKVServer struct {
	Runtime   *kayak.Runtime
	KVStorage *LocalStorage
}

// Init implements consistent.Persistence
func (s *KayakKVServer) Init(storePath string, initNodes []proto.Node) (err error) {
	for _, n := range initNodes {
		var nodeBuf *bytes.Buffer
		nodeBuf, err = utils.EncodeMsgPack(n)
		if err != nil {
			log.WithError(err).Error("marshal node failed")
			return
		}
		payload := &KayakPayload{
			Command: CmdSet,
			Data:    nodeBuf.Bytes(),
		}
		_, err = s.KVStorage.Commit(payload, true)
		if err != nil {
			log.WithError(err).Error("init kayak KV commit node failed")
			return
		}
	}
	return
}

// KayakPayload is the payload used in kayak Leader and Follower
type KayakPayload struct {
	Command string
	Data    []byte
}

// SetNode implements consistent.Persistence
func (s *KayakKVServer) SetNode(node *proto.Node) (err error) {
	nodeBuf, err := utils.EncodeMsgPack(node)
	if err != nil {
		log.WithError(err).Error("marshal node failed")
		return
	}
	payload := &KayakPayload{
		Command: CmdSet,
		Data:    nodeBuf.Bytes(),
	}

	_, _, err = s.Runtime.Apply(context.Background(), payload)
	if err != nil {
		log.Errorf("apply set node failed: %#v\nPayload:\n	%#v", err, payload)
	}

	return
}

// DelNode implements consistent.Persistence
func (s *KayakKVServer) DelNode(nodeID proto.NodeID) (err error) {
	// no need to del node currently
	return
}

// Reset implements consistent.Persistence
func (s *KayakKVServer) Reset() (err error) {
	// no need to reset for kayak
	return
}

// GetDatabase implements blockproducer.DBMetaPersistence.
func (s *KayakKVServer) GetDatabase(dbID proto.DatabaseID) (instance types.ServiceInstance, err error) {
	var result [][]interface{}
	query := "SELECT `meta` FROM `databases` WHERE `id` = ? LIMIT 1"
	_, _, result, err = s.KVStorage.Query(context.Background(), []storage.Query{
		{
			Pattern: query,
			Args: []sql.NamedArg{
				sql.Named("", string(dbID)),
			},
		},
	})
	if err != nil {
		log.WithField("db", dbID).WithError(err).Error("query database instance meta failed")
		return
	}

	if len(result) <= 0 || len(result[0]) <= 0 {
		err = bp.ErrNoSuchDatabase
		return
	}

	var rawInstanceMeta []byte
	var ok bool
	if rawInstanceMeta, ok = result[0][0].([]byte); !ok {
		err = bp.ErrNoSuchDatabase
		return
	}

	err = utils.DecodeMsgPack(rawInstanceMeta, &instance)
	return
}

// SetDatabase implements blockproducer.DBMetaPersistence.
func (s *KayakKVServer) SetDatabase(meta types.ServiceInstance) (err error) {
	var metaBuf *bytes.Buffer
	if metaBuf, err = utils.EncodeMsgPack(meta); err != nil {
		return
	}

	payload := &KayakPayload{
		Command: CmdSetDatabase,
		Data:    metaBuf.Bytes(),
	}

	_, _, err = s.Runtime.Apply(context.Background(), payload)
	if err != nil {
		log.Errorf("apply set database failed: %#v\nPayload:\n	%#v", err, payload)
	}

	return
}

// DeleteDatabase implements blockproducer.DBMetaPersistence.
func (s *KayakKVServer) DeleteDatabase(dbID proto.DatabaseID) (err error) {
	meta := types.ServiceInstance{
		DatabaseID: dbID,
	}

	var metaBuf *bytes.Buffer
	if metaBuf, err = utils.EncodeMsgPack(meta); err != nil {
		return
	}
	payload := &KayakPayload{
		Command: CmdDeleteDatabase,
		Data:    metaBuf.Bytes(),
	}

	_, _, err = s.Runtime.Apply(context.Background(), payload)
	if err != nil {
		log.Errorf("apply set database failed: %#v\nPayload:\n	%#v", err, payload)
	}

	return
}

// GetAllDatabases implements blockproducer.DBMetaPersistence.
func (s *KayakKVServer) GetAllDatabases() (instances []types.ServiceInstance, err error) {
	var result [][]interface{}
	query := "SELECT `meta` FROM `databases`"
	_, _, result, err = s.KVStorage.Query(context.Background(), []storage.Query{
		{
			Pattern: query,
		},
	})
	if err != nil {
		log.WithError(err).Error("query all database instance meta failed")
		return
	}

	instances = make([]types.ServiceInstance, 0, len(result))

	for _, row := range result {
		if len(row) <= 0 {
			continue
		}

		var instance types.ServiceInstance
		var rawInstanceMeta []byte
		var ok bool
		if rawInstanceMeta, ok = row[0].([]byte); !ok {
			err = bp.ErrNoSuchDatabase
			continue
		}

		if err = utils.DecodeMsgPack(rawInstanceMeta, &instance); err != nil {
			continue
		}

		instances = append(instances, instance)
	}

	if len(instances) > 0 {
		err = nil
	}

	return
}

// GetAllNodeInfo implements consistent.Persistence
func (s *KayakKVServer) GetAllNodeInfo() (nodes []proto.Node, err error) {
	var result [][]interface{}
	query := "SELECT `node` FROM `dht`;"
	_, _, result, err = s.KVStorage.Query(context.Background(), []storage.Query{
		{
			Pattern: query,
		},
	})
	if err != nil {
		log.WithField("query", query).WithError(err).Error("query failed")
		return
	}
	log.Debugf("SQL: %#v\nResults: %#v", query, result)

	nodes = make([]proto.Node, 0, len(result))

	for _, r := range result {
		if len(r) == 0 {
			continue
		}
		nodeBytes, ok := r[0].([]byte)
		log.Debugf("nodeBytes: %#v, %#v", nodeBytes, ok)
		if !ok {
			continue
		}

		nodeDec := proto.NewNode()
		err = utils.DecodeMsgPack(nodeBytes, nodeDec)
		if err != nil {
			log.WithError(err).Error("unmarshal node info failed")
			continue
		}
		nodes = append(nodes, *nodeDec)
	}

	if len(nodes) > 0 {
		err = nil
	}
	return
}
