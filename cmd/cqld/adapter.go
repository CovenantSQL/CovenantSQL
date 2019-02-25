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
	"context"
	"database/sql"
	"os"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/CovenantSQL/CovenantSQL/consistent"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	"github.com/CovenantSQL/CovenantSQL/storage"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

// LocalStorage holds consistent and storage struct.
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

// SetNode handles dht storage node update.
func (s *LocalStorage) SetNode(node *proto.Node) (err error) {
	query := "INSERT OR REPLACE INTO `dht` (`id`, `node`) VALUES (?, ?);"
	log.Debugf("sql: %#v", query)

	nodeBuf, err := utils.EncodeMsgPack(node)
	if err != nil {
		err = errors.Wrap(err, "encode node failed")
		return
	}

	err = route.SetNodeAddrCache(node.ID.ToRawNodeID(), node.Addr)
	if err != nil {
		log.WithFields(log.Fields{
			"id":   node.ID,
			"addr": node.Addr,
		}).WithError(err).Error("set node addr cache failed")
	}
	err = kms.SetNode(node)
	if err != nil {
		log.WithField("node", node).WithError(err).Error("kms set node failed")
	}

	// if s.consistent == nil, it is called during Init. and AddCache will be called by consistent.InitConsistent
	if s.consistent != nil {
		s.consistent.AddCache(*node)
	}

	// execute query
	if _, err = s.Storage.Exec(context.Background(), []storage.Query{
		{
			Pattern: query,
			Args: []sql.NamedArg{
				sql.Named("", node.ID),
				sql.Named("", nodeBuf.Bytes()),
			},
		},
	}); err != nil {
		err = errors.Wrap(err, "execute query in dht database failed")
	}

	return
}

// KVServer holds LocalStorage instance and implements consistent persistence interface.
type KVServer struct {
	current   proto.NodeID
	peers     *proto.Peers
	storage   *LocalStorage
	ctx       context.Context
	cancelCtx context.CancelFunc
	timeout   time.Duration
	wg        sync.WaitGroup
}

// NewKVServer returns the kv server instance.
func NewKVServer(currentNode proto.NodeID, peers *proto.Peers, storage *LocalStorage, timeout time.Duration) (s *KVServer) {
	ctx, cancelCtx := context.WithCancel(context.Background())

	return &KVServer{
		current:   currentNode,
		peers:     peers,
		storage:   storage,
		ctx:       ctx,
		cancelCtx: cancelCtx,
		timeout:   timeout,
	}
}

// Init implements consistent.Persistence
func (s *KVServer) Init(storePath string, initNodes []proto.Node) (err error) {
	for _, n := range initNodes {
		err = s.storage.SetNode(&n)
		if err != nil {
			log.WithError(err).Error("init dht kv server failed")
			return
		}
	}
	return
}

// SetNode implements consistent.Persistence
func (s *KVServer) SetNode(node *proto.Node) (err error) {
	return s.SetNodeEx(node, 1, s.current)
}

// SetNodeEx is used by gossip service to broadcast to other nodes.
func (s *KVServer) SetNodeEx(node *proto.Node, ttl uint32, origin proto.NodeID) (err error) {
	log.WithFields(log.Fields{
		"node":   node.ID,
		"ttl":    ttl,
		"origin": origin,
	}).Debug("update node to kv storage")

	// set local
	if err = s.storage.SetNode(node); err != nil {
		err = errors.Wrap(err, "set node failed")
		return
	}

	if ttl > 0 {
		s.nonBlockingSync(node, origin, ttl-1)
	}

	return
}

// DelNode implements consistent.Persistence.
func (s *KVServer) DelNode(nodeID proto.NodeID) (err error) {
	// no need to del node currently
	return
}

// Reset implements consistent.Persistence.
func (s *KVServer) Reset() (err error) {
	return
}

// GetAllNodeInfo implements consistent.Persistence.
func (s *KVServer) GetAllNodeInfo() (nodes []proto.Node, err error) {
	var result [][]interface{}
	query := "SELECT `node` FROM `dht`;"
	_, _, result, err = s.storage.Query(context.Background(), []storage.Query{
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

// Stop stops the dht node server and wait for inflight gossip requests.
func (s *KVServer) Stop() {
	if s.cancelCtx != nil {
		s.cancelCtx()
	}
	s.wg.Wait()
}

func (s *KVServer) nonBlockingSync(node *proto.Node, origin proto.NodeID, ttl uint32) {
	if s.peers == nil {
		return
	}

	c, cancel := context.WithTimeout(s.ctx, s.timeout)
	defer cancel()
	for _, n := range s.peers.Servers {
		if n != s.current && n != origin {
			// sync
			req := &GossipRequest{
				Node: node,
				TTL:  ttl,
			}

			s.wg.Add(1)
			go func(node proto.NodeID) {
				defer s.wg.Done()
				_ = rpc.NewCaller().CallNodeWithContext(c, node, route.DHTGSetNode.String(), req, nil)
			}(n)
		}
	}
}
