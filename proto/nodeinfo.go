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
	"github.com/thunderdb/ThunderDB/crypto/hash"
	mine "github.com/thunderdb/ThunderDB/pow/cpuminer"
)

var (
	// NewNodeIDDifficulty is exposed for easy testing
	NewNodeIDDifficulty = 40
	// NewNodeIDDifficultyTimeout is exposed for easy testing
	NewNodeIDDifficultyTimeout = 60 * time.Second
)

// NodeID is node name, will be generated from Hash(nodePublicKey)
type NodeID string

// AccountAddress is wallet address, will be generated from Hash(nodePublicKey)
type AccountAddress string

// NodeKey is node key on consistent hash ring, generate from Hash(NodeID)
type NodeKey uint64

// Node is all node info struct
type Node struct {
	ID        NodeID
	Addr      string
	PublicKey *asymmetric.PublicKey
	Nonce     mine.Uint256
}

// NewNode just return a new node struct
func NewNode() *Node {
	return &Node{}
}

// Difficulty returns NodeID difficulty, returns -1 on length mismatch or any error
func (id *NodeID) Difficulty() (difficulty int) {
	if id == nil {
		return -1
	}
	if len(*id) != hash.HashSize*2 {
		return -1
	}
	idHash, err := hash.NewHashFromStr(string(*id))
	if err != nil {
		return -1
	}
	return idHash.Difficulty()
}

// InitNodeCryptoInfo generate Node asymmetric key pair and generate Node.NonceInfo
// Node.ID = Node.NonceInfo.Hash
func (node *Node) InitNodeCryptoInfo() (err error) {
	_, node.PublicKey, err = asymmetric.GenSecp256k1KeyPair()
	if err != nil {
		log.Error("Failed to generate key pair")
	}

	nonce := asymmetric.GetPubKeyNonce(node.PublicKey, NewNodeIDDifficulty, NewNodeIDDifficultyTimeout, nil)
	node.ID = NodeID(nonce.Hash.String())
	node.Nonce = nonce.Nonce
	log.Debugf("Node: %v", node)
	return
}
