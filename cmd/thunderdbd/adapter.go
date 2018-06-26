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

package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/ugorji/go/codec"
	"gitlab.com/thunderdb/ThunderDB/consistent"
	"gitlab.com/thunderdb/ThunderDB/kayak"
	"gitlab.com/thunderdb/ThunderDB/proto"
	"gitlab.com/thunderdb/ThunderDB/sqlchain/storage"
	"gitlab.com/thunderdb/ThunderDB/twopc"
	"gitlab.com/thunderdb/ThunderDB/utils"
)

const cmdset = "set"

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
	_, err = st.Exec(context.Background(), []string{
		"CREATE TABLE IF NOT EXISTS `dht` (`id` TEXT NOT NULL PRIMARY KEY, `node` BLOB);",
	})
	if err != nil {
		log.Errorf("create dht table failed: %s", err)
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

	err = s.consistent.AddCache(nodeToSet)
	if err != nil {
		//TODO(auxten) log the error but not return, may cause some inconsistency
		// need sync periodly
		log.Errorf("compile exec log failed: %s", err)
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
	case cmdset:
		var nodeToSet proto.Node
		err = utils.DecodeMsgPack(payload.Data, &nodeToSet)
		if err != nil {
			log.Errorf("compileExecLog: unmarshal node from payload failed: %s", err)
			return
		}
		/*
		 * Add t prefix for node id, cause sqlite do not actually has a type.
		 * execute queries below:
		 *		create table t1(myint INTEGER, mystring STRING, mytext TEXT);
		 *		insert into t1 values ('0110', '0220', '0330');
		 *		insert into t1 values ('-0110', '-0220', '-0330');
		 *		insert into t1 values ('+0110', '+0220', '+0330');
		 *		insert into t1 values ('x0110', 'x0220', 'x0330');
		 *		insert into t1 values ('011.0', '022.0', '033.0');
		 *		select * from t1
		 *
		 * will output rows with values:
		 * 		myint mystring mytext
		 * 		  110      220   0330
		 * 		 -110     -220  -0330
		 * 		  110      220  +0330
		 * 		x0110    x0220  x0330
		 * 		   11       22  033.0
		 * 	--from: https://stackoverflow.com/a/42264331/896026
		 */
		sql := fmt.Sprintf("INSERT OR REPLACE INTO `dht`(`id`, `node`) VALUES ('t%s', '%s');",
			nodeToSet.ID, base64.StdEncoding.EncodeToString(payload.Data))
		log.Debugf("sql: %s", sql)
		execLog = &storage.ExecLog{
			Queries: []string{sql},
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
	Runtime *kayak.Runtime
	Storage *LocalStorage
}

// Init implements consistent.Persistence
func (s *KayakKVServer) Init(storePath string, initNode *proto.Node) (err error) {
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
		log.Errorf("marshal node failed: %s", err)
		return
	}
	payload := &KayakPayload{
		Command: cmdset,
		Data:    nodeBuf.Bytes(),
	}

	writeData, err := utils.EncodeMsgPack(payload)
	if err != nil {
		log.Errorf("marshal payload failed: %s", err)
		return err
	}

	err = s.Runtime.Apply(writeData.Bytes())
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

// GetAllNodeInfo implements consistent.Persistence
func (s *KayakKVServer) GetAllNodeInfo() (nodes []proto.Node, err error) {
	var result [][]interface{}
	mh := &codec.MsgpackHandle{}
	//sql := fmt.Sprintf("SELECT `node` FROM `dht` WHERE `id` = 't%s' LIMIT 1;", id)
	sql := fmt.Sprintf("SELECT `node` FROM `dht`;")
	_, _, result, err = s.Storage.Query(context.Background(), []string{sql})
	if err != nil {
		log.Errorf("Query: %s failed: %s", sql, err)
		return
	}
	log.Debugf("SQL: %v\nResults: %s", sql, result)
	for _, r := range result {
		nodes = make([]proto.Node, 0, len(result))
		if len(r) == 0 {
			continue
		}
		nodeSerial, ok := r[0].([]byte)
		log.Debugf("nodeSerial: %s, %v", nodeSerial, ok)
		if !ok {
			continue
		}
		nodeBytes, err := base64.StdEncoding.DecodeString(string(nodeSerial))
		if err != nil {
			log.Errorf("base64 decoding failed: %s", err)
			continue
		}

		nodeDec := proto.NewNode()
		reader := bytes.NewReader(nodeBytes)
		dec := codec.NewDecoder(reader, mh)
		err = dec.Decode(nodeDec)
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
