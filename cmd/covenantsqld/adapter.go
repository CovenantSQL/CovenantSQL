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
	"errors"
	"os"

	bp "github.com/CovenantSQL/CovenantSQL/blockproducer"
	"github.com/CovenantSQL/CovenantSQL/consistent"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/kayak"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/sqlchain/storage"
	"github.com/CovenantSQL/CovenantSQL/twopc"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	wt "github.com/CovenantSQL/CovenantSQL/worker/types"
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
		log.Errorf("create dht table %s failed: %s", utils.FJ(wd, dbFile), err)
		return
	}

	stor = &LocalStorage{
		Storage: st,
	}

	return
}

// Prepare implements twopc Worker.Prepare
func (s *LocalStorage) Prepare(ctx context.Context, wb twopc.WriteBatch) (err error) {
	payload, err := s.decodeLog(wb)
	if err != nil {
		log.Errorf("decode log failed: %s", err)
		return
	}
	execLog, err := s.compileExecLog(payload)
	if err != nil {
		log.Errorf("compile exec log failed: %s", err)
		return
	}
	return s.Storage.Prepare(ctx, execLog)
}

// Commit implements twopc Worker.Commit
func (s *LocalStorage) Commit(ctx context.Context, wb twopc.WriteBatch) (err error) {
	payload, err := s.decodeLog(wb)
	if err != nil {
		log.Errorf("decode log failed: %s", err)
		return
	}
	return s.commit(ctx, payload)
}

func (s *LocalStorage) commit(ctx context.Context, payload *KayakPayload) (err error) {
	var nodeToSet proto.Node
	err = utils.DecodeMsgPack(payload.Data, &nodeToSet)
	if err != nil {
		log.Errorf("unmarshal node from payload failed: %s", err)
		return
	}
	execLog, err := s.compileExecLog(payload)
	if err != nil {
		log.Errorf("compile exec log failed: %s", err)
		return
	}
	err = route.SetNodeAddrCache(nodeToSet.ID.ToRawNodeID(), nodeToSet.Addr)
	if err != nil {
		log.Errorf("set node addr %s %s cache failed: %v", nodeToSet.ID, nodeToSet.Addr, err)
	}
	err = kms.SetNode(&nodeToSet)
	if err != nil {
		log.Errorf("kms set node %v failed: %v", nodeToSet, err)
	}

	// if s.consistent == nil, it is called during Init. and AddCache will be called by consistent.InitConsistent
	if s.consistent != nil {
		err = s.consistent.AddCache(nodeToSet)
		if err != nil {
			//TODO(auxten) even no error will be returned, there may be some inconsistency and needs sync periodically
			log.Errorf("add consistent cache failed: %s", err)
		}
	}

	return s.Storage.Commit(ctx, execLog)
}

// Rollback implements twopc Worker.Rollback
func (s *LocalStorage) Rollback(ctx context.Context, wb twopc.WriteBatch) (err error) {
	payload, err := s.decodeLog(wb)
	if err != nil {
		log.Errorf("decode log failed: %s", err)
		return
	}
	execLog, err := s.compileExecLog(payload)
	if err != nil {
		log.Errorf("compile exec log failed: %s", err)
		return
	}

	return s.Storage.Rollback(ctx, execLog)
}

func (s *LocalStorage) compileExecLog(payload *KayakPayload) (execLog *storage.ExecLog, err error) {
	switch payload.Command {
	case CmdSet:
		var nodeToSet proto.Node
		err = utils.DecodeMsgPack(payload.Data, &nodeToSet)
		if err != nil {
			log.Errorf("compileExecLog: unmarshal node from payload failed: %s", err)
			return
		}
		query := "INSERT OR REPLACE INTO `dht` (`id`, `node`) VALUES (?, ?);"
		log.Debugf("sql: %s", query)
		execLog = &storage.ExecLog{
			Queries: []storage.Query{
				{
					Pattern: query,
					Args: []sql.NamedArg{
						sql.Named("", nodeToSet.ID),
						sql.Named("", payload.Data),
					},
				},
			},
		}
	case CmdSetDatabase:
		var instance wt.ServiceInstance
		if err = utils.DecodeMsgPack(payload.Data, &instance); err != nil {
			log.Errorf("compileExecLog: unmarshal instance meta failed: %v", err)
			return
		}
		query := "INSERT OR REPLACE INTO `databases` (`id`, `meta`) VALUES (? ,?);"
		execLog = &storage.ExecLog{
			Queries: []storage.Query{
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
		var instance wt.ServiceInstance
		if err = utils.DecodeMsgPack(payload.Data, &instance); err != nil {
			log.Errorf("compileExecLog: unmarshal instance id failed: %v", err)
			return
		}
		// TODO(xq262144), should add additional limit 1 after delete clause
		// however, currently the go-sqlite3
		query := "DELETE FROM `databases` WHERE `id` = ?"
		execLog = &storage.ExecLog{
			Queries: []storage.Query{
				{
					Pattern: query,
					Args: []sql.NamedArg{
						sql.Named("", string(instance.DatabaseID)),
					},
				},
			},
		}
	default:
		err = errors.New("undefined command: " + payload.Command)
		log.Error(err)
	}
	return
}

func (s *LocalStorage) decodeLog(wb twopc.WriteBatch) (payload *KayakPayload, err error) {
	var bytesPayload []byte
	var ok bool
	payload = new(KayakPayload)

	if bytesPayload, ok = wb.([]byte); !ok {
		err = kayak.ErrInvalidLog
		return
	}
	err = utils.DecodeMsgPack(bytesPayload, payload)
	if err != nil {
		log.Errorf("unmarshal payload failed: %s", err)
		return
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
			log.Errorf("marshal node failed: %v", err)
			return
		}
		payload := &KayakPayload{
			Command: CmdSet,
			Data:    nodeBuf.Bytes(),
		}

		var execLog *storage.ExecLog
		execLog, err = s.KVStorage.compileExecLog(payload)
		if err != nil {
			log.Errorf("compile exec log failed: %s", err)
			return
		}
		err = s.KVStorage.Storage.Prepare(context.Background(), execLog)
		if err != nil {
			log.Errorf("init kayak KV prepare node failed: %v", err)
			return
		}

		err = s.KVStorage.commit(context.Background(), payload)
		if err != nil {
			log.Errorf("init kayak KV commit node failed: %v", err)
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
		log.Errorf("marshal node failed: %v", err)
		return
	}
	payload := &KayakPayload{
		Command: CmdSet,
		Data:    nodeBuf.Bytes(),
	}

	writeData, err := utils.EncodeMsgPack(payload)
	if err != nil {
		log.Errorf("marshal payload failed: %v", err)
		return err
	}

	_, err = s.Runtime.Apply(writeData.Bytes())
	if err != nil {
		log.Errorf("Apply set node failed: %s\nPayload:\n	%s", err, writeData)
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
func (s *KayakKVServer) GetDatabase(dbID proto.DatabaseID) (instance wt.ServiceInstance, err error) {
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
		log.Errorf("Query database %v instance meta failed: %v", dbID, err)
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
func (s *KayakKVServer) SetDatabase(meta wt.ServiceInstance) (err error) {
	var metaBuf *bytes.Buffer
	if metaBuf, err = utils.EncodeMsgPack(meta); err != nil {
		return
	}

	payload := &KayakPayload{
		Command: CmdSetDatabase,
		Data:    metaBuf.Bytes(),
	}

	writeData, err := utils.EncodeMsgPack(payload)
	if err != nil {
		log.Errorf("marshal payload failed: %s", err)
		return err
	}

	_, err = s.Runtime.Apply(writeData.Bytes())
	if err != nil {
		log.Errorf("Apply set database failed: %s\nPayload:\n	%s", err, writeData)
	}

	return
}

// DeleteDatabase implements blockproducer.DBMetaPersistence.
func (s *KayakKVServer) DeleteDatabase(dbID proto.DatabaseID) (err error) {
	meta := wt.ServiceInstance{
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

	writeData, err := utils.EncodeMsgPack(payload)
	if err != nil {
		log.Errorf("marshal payload failed: %s", err)
		return err
	}

	_, err = s.Runtime.Apply(writeData.Bytes())
	if err != nil {
		log.Errorf("Apply set database failed: %s\nPayload:\n	%s", err, writeData)
	}

	return
}

// GetAllDatabases implements blockproducer.DBMetaPersistence.
func (s *KayakKVServer) GetAllDatabases() (instances []wt.ServiceInstance, err error) {
	var result [][]interface{}
	query := "SELECT `meta` FROM `databases`"
	_, _, result, err = s.KVStorage.Query(context.Background(), []storage.Query{
		{
			Pattern: query,
		},
	})
	if err != nil {
		log.Errorf("Query all database instance meta failed: %v", err)
		return
	}

	instances = make([]wt.ServiceInstance, 0, len(result))

	for _, row := range result {
		if len(row) <= 0 {
			continue
		}

		var instance wt.ServiceInstance
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
		log.Errorf("Query: %s failed: %s", query, err)
		return
	}
	log.Debugf("SQL: %v\nResults: %s", query, result)

	nodes = make([]proto.Node, 0, len(result))

	for _, r := range result {
		if len(r) == 0 {
			continue
		}
		nodeBytes, ok := r[0].([]byte)
		log.Debugf("nodeBytes: %s, %v", nodeBytes, ok)
		if !ok {
			continue
		}

		nodeDec := proto.NewNode()
		err = utils.DecodeMsgPack(nodeBytes, nodeDec)
		if err != nil {
			log.Errorf("unmarshal node info failed: %s", err)
			continue
		}
		nodes = append(nodes, *nodeDec)
	}

	if len(nodes) > 0 {
		err = nil
	}
	return
}
