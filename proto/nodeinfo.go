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

package proto

import (
	"strings"
	"time"

	hsp "github.com/CovenantSQL/HashStablePack/marshalhash"

	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	mine "github.com/CovenantSQL/CovenantSQL/pow/cpuminer"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

//go:generate hsp

var (
	// NewNodeIDDifficulty is exposed for easy testing.
	NewNodeIDDifficulty = 40
	// NewNodeIDDifficultyTimeout is exposed for easy testing.
	NewNodeIDDifficultyTimeout = 60 * time.Second
)

const (
	// NodeIDLen is the NodeID length.
	NodeIDLen = 2 * hash.HashSize
)

// RawNodeID is node name, will be generated from Hash(nodePublicKey)
// RawNodeID length should be 32 bytes normally.
type RawNodeID struct {
	hash.Hash
}

// NodeID is the Hex of RawNodeID.
type NodeID string

// AccountAddress is wallet address, will be generated from Hash(nodePublicKey).
type AccountAddress hash.Hash

// UnmarshalJSON implements the json.Unmarshaler interface.
func (z *AccountAddress) UnmarshalJSON(data []byte) error {
	return ((*hash.Hash)(z)).UnmarshalJSON(data)
}

// MarshalJSON implements the json.Marshaler interface.
func (z AccountAddress) MarshalJSON() ([]byte, error) {
	return ((hash.Hash)(z)).MarshalJSON()
}

// MarshalYAML implements the yaml.Marshaler interface.
func (z AccountAddress) MarshalYAML() (interface{}, error) {
	return ((hash.Hash)(z)).MarshalYAML()
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (z *AccountAddress) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return ((*hash.Hash)(z)).UnmarshalYAML(unmarshal)
}

// NodeKey is node key on consistent hash ring, generate from Hash(NodeID).
type NodeKey RawNodeID

// DatabaseID converts AccountAddress to DatabaseID.
func (z *AccountAddress) DatabaseID() (d DatabaseID) {
	d = DatabaseID(z.String())
	return
}

// MarshalHash marshals for hash.
func (z *AccountAddress) MarshalHash() (o []byte, err error) {
	return (*hash.Hash)(z).MarshalHash()
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message.
func (z *AccountAddress) Msgsize() (s int) {
	return hsp.BytesPrefixSize + hash.HashSize
}

// String is a string variable.
func (z AccountAddress) String() string {
	return (hash.Hash)(z).String()
}

// Less return true if k is less than y.
func (k *NodeKey) Less(y *NodeKey) bool {
	for idx, val := range k.Hash {
		if val == y.Hash[idx] {
			continue
		}
		return val < y.Hash[idx]
	}
	return false
}

// Node is all node info struct.
type Node struct {
	ID        NodeID                `yaml:"ID"`
	Role      ServerRole            `yaml:"Role"`
	Addr      string                `yaml:"Addr"`
	PublicKey *asymmetric.PublicKey `yaml:"PublicKey"`
	Nonce     mine.Uint256          `yaml:"Nonce"`
}

// NewNode just return a new node struct.
func NewNode() *Node {
	return &Node{}
}

// Difficulty returns NodeID difficulty, returns -1 on length mismatch or any error.
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

// ToRawNodeID converts NodeID to RawNodeID.
func (id NodeID) ToRawNodeID() *RawNodeID {
	idHash, err := hash.NewHashFromStr(string(id))
	if err != nil {
		log.WithField("id", id).WithError(err).Error("error node id")
		return nil
	}
	return &RawNodeID{*idHash}
}

// IsEmpty test if a nodeID is empty.
func (id *NodeID) IsEmpty() bool {
	return id == nil || "" == string(*id) || id.ToRawNodeID().IsEqual(&hash.Hash{})
}

// IsEqual returns if two node id is equal.
func (id *NodeID) IsEqual(target *NodeID) bool {
	return strings.Compare(string(*id), string(*target)) == 0
}

// MarshalBinary does the serialization.
func (id *NodeID) MarshalBinary() (keyBytes []byte, err error) {
	if id == nil {
		return []byte{}, nil
	}
	h, err := hash.NewHashFromStr(string(*id))
	return h[:], err
}

// UnmarshalBinary does the deserialization.
func (id *NodeID) UnmarshalBinary(keyBytes []byte) (err error) {
	// for backward compatible
	if len(keyBytes) == 64 {
		*id = NodeID(keyBytes)
		return
	}
	h, err := hash.NewHash(keyBytes)
	if err != nil {
		log.Error("load 32 bytes nodeID failed")
		return
	}
	*id = NodeID(h.String())
	return
}

// InitNodeCryptoInfo generate Node asymmetric key pair and generate Node.NonceInfo.
// Node.ID = Node.NonceInfo.Hash.
func (node *Node) InitNodeCryptoInfo(timeThreshold time.Duration) (err error) {
	_, node.PublicKey, err = asymmetric.GenSecp256k1KeyPair()
	if err != nil {
		log.Error("failed to generate key pair")
	}

	nonce := asymmetric.GetPubKeyNonce(node.PublicKey, NewNodeIDDifficulty, timeThreshold, nil)
	node.ID = NodeID(nonce.Hash.String())
	node.Nonce = nonce.Nonce
	log.Debugf("node: %#v", node)
	return
}

// ToNodeID converts RawNodeID to NodeID.
func (id *RawNodeID) ToNodeID() NodeID {
	if id == nil {
		return NodeID("")
	}
	return NodeID(id.String())
}

// AddrAndGas records each node's address, node id and gas that node receives.
type AddrAndGas struct {
	AccountAddress AccountAddress
	RawNodeID      RawNodeID
	GasAmount      uint64
}

// ServerRole defines the role of node to be leader/coordinator in peer set.
type ServerRole int

const (
	// Unknown is the zero value
	Unknown ServerRole = iota
	// Leader is a server that have the ability to organize committing requests.
	Leader
	// Follower is a server that follow the leader log commits.
	Follower
	// Miner is a server that run sql database.
	Miner
	// Client is a client that send sql query to database>
	Client
)

// String is a string variable of ServerRole.
func (s ServerRole) String() string {
	switch s {
	case Leader:
		return "Leader"
	case Follower:
		return "Follower"
	case Miner:
		return "Miner"
	case Client:
		return "Client"
	}
	return "Unknown"
}

// MarshalYAML implements the yaml.Marshaler interface.
func (s ServerRole) MarshalYAML() (interface{}, error) {
	return s.String(), nil
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (s *ServerRole) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var str string
	if err := unmarshal(&str); err != nil {
		return err
	}
	dur, err := parseServerRole(str)
	if err != nil {
		return err
	}
	*s = dur

	return nil
}

func parseServerRole(roleStr string) (role ServerRole, err error) {
	switch strings.ToLower(roleStr) {
	case "leader":
		role = Leader
		return
	case "follower":
		role = Follower
		return
	case "miner":
		role = Miner
		return
	case "client":
		role = Client
		return
	}

	return Unknown, nil
}

// ServerRoles is []ServerRole.
type ServerRoles []ServerRole

// Contains returns if given role is in the ServerRoles>.
func (ss *ServerRoles) Contains(role ServerRole) bool {
	for _, s := range *ss {
		if s == role {
			return true
		}
	}
	return false
}
