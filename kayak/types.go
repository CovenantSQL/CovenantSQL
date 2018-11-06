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

package kayak

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/proto"
)

//go:generate hsp
//hsp:ignore RuntimeConfig

// Log entries are replicated to all members of the Kayak cluster
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

// ComputeHash updates Hash.
func (l *Log) ComputeHash() {
	l.Hash.SetBytes(hash.DoubleHashB(l.Serialize()))
}

// VerifyHash validates hash field.
func (l *Log) VerifyHash() bool {
	h := hash.DoubleHashH(l.Serialize())
	return h.IsEqual(&l.Hash)
}

// Serialize transform log structure to bytes.
func (l *Log) Serialize() []byte {
	if l == nil {
		return []byte{'\000'}
	}

	buf := new(bytes.Buffer)

	binary.Write(buf, binary.LittleEndian, l.Index)
	binary.Write(buf, binary.LittleEndian, l.Term)
	binary.Write(buf, binary.LittleEndian, uint64(len(l.Data)))
	buf.Write(l.Data)
	if l.LastHash != nil {
		buf.Write(l.LastHash[:])
	} else {
		buf.WriteRune('\000')
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
	GetLog(index uint64, l *Log) error

	// StoreLog stores a log entry.
	StoreLog(l *Log) error

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

// ServerState define the state of node to be checked by commit/peers update logic.
type ServerState int

// Note: Don't renumber these, since the numbers are written into the log.
const (
	// Idle indicates no running transaction.
	Idle ServerState = iota

	// Prepared indicates in-flight transaction prepared.
	Prepared

	// Shutdown state
	Shutdown
)

func (s ServerState) String() string {
	switch s {
	case Idle:
		return "Idle"
	case Prepared:
		return "Prepared"
	}
	return "Unknown"
}

// Server tracks the information about a single server in a configuration.
type Server struct {
	// Suffrage determines whether the server gets a vote.
	Role proto.ServerRole
	// ID is a unique string identifying this server for all time.
	ID proto.NodeID
	// Public key
	PubKey *asymmetric.PublicKey
}

func (s *Server) String() string {
	return fmt.Sprintf("Server id:%s role:%s pubKey:%s",
		s.ID, s.Role,
		base64.StdEncoding.EncodeToString(s.PubKey.Serialize()))
}

// Serialize server struct to bytes.
func (s *Server) Serialize() []byte {
	if s == nil {
		return []byte{'\000'}
	}

	buffer := new(bytes.Buffer)
	binary.Write(buffer, binary.LittleEndian, s.Role)
	binary.Write(buffer, binary.LittleEndian, uint64(len(s.ID)))
	buffer.WriteString(string(s.ID))
	if s.PubKey != nil {
		buffer.Write(s.PubKey.Serialize())
	} else {
		buffer.WriteRune('\000')
	}

	return buffer.Bytes()
}

// Peers defines peer configuration.
type Peers struct {
	Term      uint64
	Leader    *Server
	Servers   []*Server
	PubKey    *asymmetric.PublicKey
	Signature *asymmetric.Signature
}

// Clone makes a deep copy of a Peers.
func (c *Peers) Clone() (copy Peers) {
	copy.Term = c.Term
	copy.Leader = c.Leader
	copy.Servers = append(copy.Servers, c.Servers...)
	copy.PubKey = c.PubKey
	copy.Signature = c.Signature
	return
}

// Serialize peers struct to bytes.
func (c *Peers) Serialize() []byte {
	if c == nil {
		return []byte{'\000'}
	}

	buffer := new(bytes.Buffer)
	binary.Write(buffer, binary.LittleEndian, c.Term)
	binary.Write(buffer, binary.LittleEndian, c.Leader.Serialize())
	binary.Write(buffer, binary.LittleEndian, uint64(len(c.Servers)))
	for _, s := range c.Servers {
		binary.Write(buffer, binary.LittleEndian, s.Serialize())
	}
	if c.PubKey != nil {
		buffer.Write(c.PubKey.Serialize())
	} else {
		buffer.WriteRune('\000')
	}
	return buffer.Bytes()
}

// Sign generates signature.
func (c *Peers) Sign(signer *asymmetric.PrivateKey) error {
	c.PubKey = signer.PubKey()
	h := hash.THashB(c.Serialize())
	sig, err := signer.Sign(h)

	if err != nil {
		return fmt.Errorf("sign peer configuration failed: %s", err.Error())
	}

	c.Signature = sig

	return nil
}

// Verify verify signature.
func (c *Peers) Verify() bool {
	h := hash.THashB(c.Serialize())

	return c.Signature.Verify(h, c.PubKey)
}

func (c *Peers) String() string {
	return fmt.Sprintf("Peers term:%v nodesCnt:%v leader:%s signature:%s",
		c.Term, len(c.Servers), c.Leader.ID,
		base64.StdEncoding.EncodeToString(c.Signature.Serialize()))
}

// Find finds the index of the server with the specified key in the server list.
func (c *Peers) Find(key proto.NodeID) (index int32, found bool) {
	if c.Servers != nil {
		for i, s := range c.Servers {
			if s.ID == key {
				index = int32(i)
				found = true
				break
			}
		}
	}

	return
}

// RuntimeConfig defines minimal configuration fields for consensus runner.
type RuntimeConfig struct {
	// RootDir is the root dir for runtime
	RootDir string

	// LocalID is the unique ID for this server across all time.
	LocalID proto.NodeID

	// Runner defines the runner type
	Runner Runner

	// Transport defines the dialer type
	Transport Transport

	// ProcessTimeout defines whole process timeout
	ProcessTimeout time.Duration

	// AutoBanCount defines how many times a nodes will be banned from execution
	AutoBanCount uint32
}

// Config interface for abstraction.
type Config interface {
	// Get config returns runtime config
	GetRuntimeConfig() *RuntimeConfig
}

// Request defines a transport request payload.
type Request interface {
	GetPeerNodeID() proto.NodeID
	GetMethod() string
	GetLog() *Log
	SendResponse([]byte, error) error
}

// Transport adapter for abstraction.
type Transport interface {
	Init() error

	// Request
	Request(ctx context.Context, nodeID proto.NodeID, method string, log *Log) ([]byte, error)

	// Process
	Process() <-chan Request

	Shutdown() error
}

// Runner adapter for different consensus protocols including Eventual Consistency/2PC/3PC.
type Runner interface {
	// Init defines setup logic.
	Init(config Config, peers *Peers, logs LogStore, stable StableStore, transport Transport) error

	// UpdatePeers defines peer configuration update logic.
	UpdatePeers(peers *Peers) error

	// Apply defines log replication and log commit logic
	// and should be called by Leader role only.
	Apply(data []byte) (interface{}, uint64, error)

	// Shutdown defines destruct logic.
	Shutdown(wait bool) error
}
