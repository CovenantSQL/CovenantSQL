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
	"errors"
	"time"

	ec "github.com/btcsuite/btcd/btcec"
	pb "github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
	"github.com/thunderdb/ThunderDB/crypto/asymmetric"
	"github.com/thunderdb/ThunderDB/crypto/hash"
	mine "github.com/thunderdb/ThunderDB/pow/cpuminer"
	"github.com/thunderdb/ThunderDB/types"
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
	PublicKey *ec.PublicKey
	Nonce     mine.Uint256
}

// NewNode just return a new node struct
func NewNode() *Node {
	return &Node{}
}

// NodeBytes is the marshal output of proto.Node.Marshal
type NodeBytes []byte

// Marshal the proto.Node with protobuf
func (n *Node) Marshal() (nodeBytes NodeBytes, err error) {
	if n == nil {
		return nil, errors.New("nil node")
	}
	if n.PublicKey == nil {
		return nil, errors.New("nil public key")
	}

	nodeBuf, err := pb.Marshal(&types.Node{
		ID: &types.NodeID{
			NodeID: string(n.ID),
		},
		Addr: n.Addr,
		PublicKey: &types.PublicKey{
			PublicKey: n.PublicKey.SerializeCompressed(),
		},
		Nonce: &types.Nonce{
			A: n.Nonce.A,
			B: n.Nonce.B,
			C: n.Nonce.C,
			D: n.Nonce.D,
		},
	})
	if err != nil {
		log.Errorf("marshal node failed: %s", err)
		return nil, err
	}

	return NodeBytes(nodeBuf), err
}

// UnmarshalNode the proto.Node with protobuf
func UnmarshalNode(nodeBytes NodeBytes) (node *Node, err error) {
	if nodeBytes == nil {
		return nil, errors.New("unmarshal got nil input")
	}
	nodeInfoTypes := &types.Node{}
	err = pb.Unmarshal(nodeBytes, nodeInfoTypes)
	if err != nil {
		log.Errorf("unmarshal node failed: %s", err)
		return
	}

	publicKey, err := ec.ParsePubKey(nodeInfoTypes.PublicKey.PublicKey, ec.S256())
	if err != nil {
		log.Errorf("parse public key failed: %s", err)
		return
	}
	node = &Node{
		ID:        NodeID(nodeInfoTypes.ID.NodeID),
		Addr:      nodeInfoTypes.Addr,
		PublicKey: publicKey,
		Nonce: mine.Uint256{
			A: nodeInfoTypes.Nonce.A,
			B: nodeInfoTypes.Nonce.B,
			C: nodeInfoTypes.Nonce.C,
			D: nodeInfoTypes.Nonce.D,
		},
	}
	return
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
