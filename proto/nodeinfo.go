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

	"gitlab.com/thunderdb/ThunderDB/crypto/asymmetric"
	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
	mine "gitlab.com/thunderdb/ThunderDB/pow/cpuminer"
	"gitlab.com/thunderdb/ThunderDB/utils/log"
)

var (
	// NewNodeIDDifficulty is exposed for easy testing
	NewNodeIDDifficulty = 40
	// NewNodeIDDifficultyTimeout is exposed for easy testing
	NewNodeIDDifficultyTimeout = 60 * time.Second
)

const (
	// NodeIDLen is the NodeID length
	NodeIDLen = 2 * hash.HashSize
)

// RawNodeID is node name, will be generated from Hash(nodePublicKey)
// RawNodeID length should be 32 bytes normally
type RawNodeID struct {
	hash.Hash
}

// NodeID is the Hex of RawNodeID
type NodeID string

// AccountAddress is wallet address, will be generated from Hash(nodePublicKey)
type AccountAddress hash.Hash

// NodeKey is node key on consistent hash ring, generate from Hash(NodeID)
type NodeKey RawNodeID

// Less return true if k is less than y
func (k *NodeKey) Less(y *NodeKey) bool {
	for idx, val := range k.Hash {
		if val == y.Hash[idx] {
			continue
		}
		return val < y.Hash[idx]
	}
	return false
}

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

// ToRawNodeID converts NodeID to RawNodeID
func (id *NodeID) ToRawNodeID() *RawNodeID {
	idHash, err := hash.NewHashFromStr(string(*id))
	if err != nil {
		log.Errorf("error node id %s %s", *id, err)
		return nil
	}
	return &RawNodeID{*idHash}
}

// InitNodeCryptoInfo generate Node asymmetric key pair and generate Node.NonceInfo
// Node.ID = Node.NonceInfo.Hash
func (node *Node) InitNodeCryptoInfo(timeThreshold time.Duration) (err error) {
	_, node.PublicKey, err = asymmetric.GenSecp256k1KeyPair()
	if err != nil {
		log.Error("Failed to generate key pair")
	}

	nonce := asymmetric.GetPubKeyNonce(node.PublicKey, NewNodeIDDifficulty, timeThreshold, nil)
	node.ID = NodeID(nonce.Hash.String())
	node.Nonce = nonce.Nonce
	log.Debugf("Node: %v", node)
	return
}

// ToNodeID converts RawNodeID to NodeID
func (id *RawNodeID) ToNodeID() NodeID {
	return NodeID(id.String())
}
