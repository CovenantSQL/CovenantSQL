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

package proto

import (
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/thunderdb/ThunderDB/crypto/asymmetric"
)

var (
	// NewNodeIDDifficulty is simply exposed for easy testing
	NewNodeIDDifficulty = 40
	// NewNodeIDDifficultyTimeout is simply exposed for easy testing
	NewNodeIDDifficultyTimeout = 60 * time.Second
)

// NewNode just return a new node struct
func NewNode() *Node {
	return &Node{}
}

// InitNodeCryptoInfo generate Node asymmetric key pair and generate Node.Nonce
// Node.ID = Node.Nonce.Hash
func (node *Node) InitNodeCryptoInfo() (err error) {
	node.privateKey, node.PublicKey, err = asymmetric.GenSecp256k1KeyPair()
	if err != nil {
		log.Error("Failed to generate key pair")
	}

	node.Nonce = asymmetric.GetPubKeyNonce(node.PublicKey, NewNodeIDDifficulty, NewNodeIDDifficultyTimeout, nil)
	node.ID = NodeID(node.Nonce.Hash.String())
	log.Debugf("Node: %v", node)
	return
}
