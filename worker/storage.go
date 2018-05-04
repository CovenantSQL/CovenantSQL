/*
 * Copyright (c) Philip O'Toole (http://www.philipotoole.com) 2015
 * Copyright (c) rqlite contributors 2015
 * Copyright (c) Google 2018
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

package worker

import (
	"path/filepath"
	"sync"
	"time"

	sdb "github.com/rqlite/rqlite/db"
	"github.com/thunderdb/ThunderDB/proto"
)

const (
	sqliteFile = "db.sqlite"
)

// ClusterState defines the possible Database states the current node can be in
type ClusterState int

// Represents the Database cluster states
const (
	Leader   ClusterState = iota
	Follower
	Shutdown
	Unknown
)

// Store is a storage instance for worker service
type Store struct {
	contextDir string // Storage instance context dir
	dbPath     string // SQLite database path
	db         *sdb.DB
	dbConn     *sdb.Conn
	nodeID     proto.NodeID
	dbID       proto.DatabaseID

	peers      []proto.Node
	leaderNode proto.Node

	restoreMu sync.RWMutex // Restore needs exclusive access to database.

	ReplicationTimeout time.Duration // ReplicationTimeout adjusted by BlockProducer between instances
	ApplyTimeout       time.Duration // ApplyTimeout adjusted by BlockProducer during transactions
	LeaderLeaseTimeout time.Duration // LeaderLeaseTimeout adjusted by BlockProducer as leader term duration
}

// StoreConfig is a worker service storage instance config
type StoreConfig struct {
	NodeID     proto.NodeID     // Node ID, FIXME, sql chain use common.Address, unified node id is required
	DatabaseID proto.DatabaseID // Database instance ID
	Dir        string           // The working directory for database
}

// New returns a new Store
func New(c *StoreConfig) *Store {
	return &Store{
		contextDir: c.Dir,
		dbPath:     filepath.Join(c.Dir, sqliteFile),
		nodeID:     c.NodeID,
		dbID:       c.DatabaseID,
	}
}

// Open open database storage for service
func (s *Store) Open() error {
	// TODO
	return nil
}

// Close database storage
func (s *Store) Close() error {
	// TODO
	return nil
}

// Destroy database storage
func (s *Store) Destroy() error {
	// TODO
	return nil
}

// IsLeader check if current node is leader node
func (s *Store) IsLeader() bool {
	// TODO
	return false
}

// State get current node cluster state
func (s *Store) State() ClusterState {
	// TODO
	return Unknown
}

// Path get current database data context dir
func (s *Store) Path() string {
	// TODO
	return s.contextDir
}

// NodeID get current node id
func (s *Store) NodeID() proto.NodeID {
	// TODO
	return s.nodeID
}

// ID get current database id
func (s *Store) ID() proto.DatabaseID {
	// TODO
	return s.dbID
}

// LeaderNode get current leader node id
func (s *Store) LeaderNode() proto.Node {
	// TODO
	return s.leaderNode
}

// Nodes get current nodes
func (s *Store) Nodes() []proto.Node {
	// TODO
	return s.peers[:]
}

// Execute real execute query
func (s *Store) Execute() {
	// TODO, for read-only query
}

// ExecuteOrAbort executes the requests with abort context
func (s *Store) ExecuteOrAbort() {
	// TODO, for consistent read/write query
}

// Backup return a checkpoint of the underlying database
func (s *Store) Backup() {
	// TODO, backup database to file, used by checkpoint logic
}
