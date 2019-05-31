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

package consistent

import (
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

// Persistence is the interface for consistent persistence.
type Persistence interface {
	Init(storePath string, initNode []proto.Node) (err error)
	SetNode(node *proto.Node) (err error)
	DelNode(nodeID proto.NodeID) (err error)
	Reset() error
	GetAllNodeInfo() (nodes []proto.Node, err error)
}

// KMSStorage implements Persistence.
type KMSStorage struct{}

// Init implements Persistence interface.
func (s *KMSStorage) Init(storePath string, initNodes []proto.Node) (err error) {
	return kms.InitPublicKeyStore(storePath, initNodes)
}

// SetNode implements Persistence interface.
func (s *KMSStorage) SetNode(node *proto.Node) (err error) {
	return kms.SetNode(node)
}

// DelNode implements Persistence interface.
func (s *KMSStorage) DelNode(nodeID proto.NodeID) (err error) {
	return kms.DelNode(nodeID)
}

// Reset implements Persistence interface.
func (s *KMSStorage) Reset() (err error) {
	return kms.ResetBucket()
}

// GetAllNodeInfo implements Persistence interface.
func (s *KMSStorage) GetAllNodeInfo() (nodes []proto.Node, err error) {
	IDs, err := kms.GetAllNodeID()
	if err != nil {
		log.WithError(err).Error("get all node id failed")
		return
	}
	nodes = make([]proto.Node, 0, len(IDs))

	for _, id := range IDs {
		node, err := kms.GetNodeInfo(id)
		if err != nil {
			// this may happen, just continue
			log.WithField("node", node).WithError(err).Error("get node info failed")
			continue
		}
		nodes = append(nodes, *node)
	}
	return
}
