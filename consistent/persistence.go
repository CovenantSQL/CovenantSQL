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

package consistent

import (
	log "github.com/sirupsen/logrus"
	"gitlab.com/thunderdb/ThunderDB/crypto/kms"
	"gitlab.com/thunderdb/ThunderDB/proto"
)

type Persistence interface {
	Init(storePath string, initNode *proto.Node) (err error)
	SetNode(node *proto.Node) (err error)
	DelNode(nodeID proto.NodeID) (err error)
	Reset() error
	GetAllNodeInfo() (nodes []proto.Node, err error)
}

type KMSStorage struct{}

func (s *KMSStorage) Init(storePath string, initNode *proto.Node) (err error) {
	return kms.InitPublicKeyStore(storePath, initNode)
}

func (s *KMSStorage) SetNode(node *proto.Node) (err error) {
	return kms.SetNode(node)
}

func (s *KMSStorage) DelNode(nodeID proto.NodeID) (err error) {
	return kms.DelNode(nodeID)
}

func (s *KMSStorage) Reset() (err error) {
	return kms.ResetBucket()
}

func (s *KMSStorage) GetAllNodeInfo() (nodes []proto.Node, err error) {
	IDs, err := kms.GetAllNodeID()
	if err != nil {
		log.Errorf("get all node id failed: %s", err)
		return
	}
	nodes = make([]proto.Node, 0, len(IDs))

	if IDs != nil {
		for _, id := range IDs {
			node, err := kms.GetNodeInfo(id)
			if err != nil {
				// this may happen, just continue
				log.Errorf("get node info for %s failed: %s", id, err)
				continue
			}
			nodes = append(nodes, *node)
		}
	}
	return
}
