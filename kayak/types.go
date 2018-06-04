/*
 * Copyright 2018 The ThunderDB Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the “License”);
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an “AS IS” BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package kayak

import (
	"github.com/thunderdb/ThunderDB/crypto/hash"
	"bytes"
	"encoding/binary"
	"github.com/thunderdb/ThunderDB/crypto/signature"
	"net"
	"context"
)

// Log entries are replicated to all members of the Raft cluster
// and form the heart of the replicated state machine.
type Log struct {
	// Index holds the index of the log entry.
	Index uint64

	// Term holds the election term of the log entry.
	Term uint64

	// Data holds the log entry's type-specific data.
	Data []byte

	// LastHash is log entry hash
	LastHash *hash.Hash

	// Hash is current log entry hash
	Hash hash.Hash
}

// ReHash update Hash with generated hash.
func (l *Log) ReHash() {
	l.Hash.SetBytes(hash.DoubleHashB(l.getBytes()))
}

// VerifyHash validates hash field.
func (l *Log) VerifyHash() bool {
	h := hash.DoubleHashH(l.getBytes())
	return h.IsEqual(&l.Hash)
}

func (l *Log) getBytes() []byte {
	buf := new(bytes.Buffer)

	binary.Write(buf, binary.LittleEndian, &l.Index)
	binary.Write(buf, binary.LittleEndian, &l.Term)
	buf.Write(l.Data)
	if l.LastHash != nil {
		buf.Write(l.LastHash.CloneBytes())
	}

	return buf.Bytes()
}

// LogStore is used to provide an interface for storing
// and retrieving logs in a durable fashion.
type LogStore interface {
	// FirstIndex returns the first index written. 0 for no entries.
	FirstIndex() (uint64, error)

	// LastIndex returns the last index written. 0 for no entries.
	LastIndex() (uint64, error)

	// GetLog gets a log entry at a given index.
	GetLog(index uint64, log *Log) error

	// StoreLog stores a log entry.
	StoreLog(log *Log) error

	// StoreLogs stores multiple log entries.
	StoreLogs(logs []*Log) error

	// DeleteRange deletes a range of log entries. The range is inclusive.
	DeleteRange(min, max uint64) error
}

// StableStore is used to provide stable storage
// of key configurations to ensure safety.
type StableStore interface {
	Set(key []byte, val []byte) error

	// Get returns the value for key, or an empty byte slice if key was not found.
	Get(key []byte) ([]byte, error)

	SetUint64(key []byte, val uint64) error

	// GetUint64 returns the uint64 value for key, or 0 if key was not found.
	GetUint64(key []byte) (uint64, error)
}

// ServerID is a unique string identifying a server for all time.
type ServerID string

// ServerAddress is a network address for a server that a transport can contact.
type ServerAddress string

// ServerRole define the role of node to be leader/coordinator in peer set
type ServerRole int

// Note: Don't renumber these, since the numbers are written into the log.
const (
	// Peer is a server whose vote is counted in elections and whose match index
	// is used in advancing the leader's commit index.
	Leader ServerRole = iota
	// Learner is a server that receives log entries but is not considered for
	// elections or commitment purposes.
	Follower
)

func (s ServerRole) String() string {
	switch s {
	case Leader:
		return "Leader"
	case Follower:
		return "Follower"
	}
	return "Unknown"
}

// Server tracks the information about a single server in a configuration.
type Server struct {
	// Suffrage determines whether the server gets a vote.
	Role ServerRole
	// ID is a unique string identifying this server for all time.
	ID ServerID
	// Address is its network address that a transport can contact.
	Address ServerAddress
	// Public key
	PubKey *signature.PublicKey
}

// Peers defines peer configuration.
type Peers struct {
	Term      uint64
	Leader    Server
	Servers   []Server
	Signature *signature.Signature
}

// Clone makes a deep copy of a Peers.
func (c *Peers) Clone() (copy Peers) {
	copy.Term = c.Term
	copy.Leader = c.Leader
	copy.Servers = append(copy.Servers, c.Servers...)
	copy.Signature = c.Signature
	return
}

// Config defines minimal configuration fields for consensus runner.
type Config struct {
	// The unique ID for this server across all time.
	LocalID ServerID
}

// Dialer adapter for abstraction.
type Dialer interface {
	// Dial connects to the destination server.
	Dial(serverID ServerID) (net.Conn, error)

	// DialContext connects to the destination server using the provided context.
	DialContext(ctx context.Context, serverID ServerID) (net.Conn, error)
}

// Runner adapter for different consensus protocols including Eventual Consistency/2PC/3PC.
type Runner interface {
	// Init defines setup logic.
	Init(config *Config, peers *Peers, log LogStore, stable StableStore, dialer Dialer) error

	// UpdatePeers defines peer configuration update logic.
	UpdatePeers(peers *Peers) error

	// Process defines log replication and log commit logic
	// and should be called by Leader role only.
	Process(log *Log) error

	// Shutdown defines destruct logic.
	Shutdown() error
}
