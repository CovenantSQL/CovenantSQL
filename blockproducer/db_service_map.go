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

package blockproducer

import (
	"sync"

	"gitlab.com/thunderdb/ThunderDB/kayak"
	"gitlab.com/thunderdb/ThunderDB/proto"
)

// DBResourceMeta defines resource meta info required by database.
type DBResourceMeta struct {
	Node   uint16 // reserved node count
	Space  uint64 // reserved storage space in bytes
	Memory uint64 // reserved memory in bytes
}

// DBInstanceMeta defines database instance meta.
type DBInstanceMeta struct {
	DatabaseID   proto.DatabaseID
	Peers        *kayak.Peers
	ResourceMeta DBResourceMeta
}

// DBMetaPersistence defines database meta persistence api.
type DBMetaPersistence interface {
	GetDatabase(dbID proto.DatabaseID) (DBInstanceMeta, error)
	SetDatabase(dbID proto.DatabaseID, meta DBInstanceMeta) error
	DeleteDatabase(dbID proto.DatabaseID) error
	GetAllDatabases() (map[proto.DatabaseID]DBInstanceMeta, error)
}

// DBServiceMap defines database instance meta.
type DBServiceMap struct {
	dbMap   map[proto.DatabaseID]DBInstanceMeta
	nodeMap map[proto.NodeID]map[proto.DatabaseID]bool
	persist DBMetaPersistence
	sync.RWMutex
}

// InitServiceMap init database service map.
func InitServiceMap(persistImpl DBMetaPersistence) (s *DBServiceMap, err error) {
	s = &DBServiceMap{
		persist: persistImpl,
		dbMap:   make(map[proto.DatabaseID]DBInstanceMeta),
		nodeMap: make(map[proto.NodeID]map[proto.DatabaseID]bool),
	}

	// load from persistence
	s.Lock()
	defer s.Unlock()

	var dbMap map[proto.DatabaseID]DBInstanceMeta

	if dbMap, err = s.persist.GetAllDatabases(); err != nil {
		return
	}

	for dbID, meta := range dbMap {
		s.dbMap[dbID] = meta

		for _, server := range meta.Peers.Servers {
			s.nodeMap[server.ID][dbID] = true
		}
	}

	return
}

// Set add database to meta.
func (c *DBServiceMap) Set(dbID proto.DatabaseID, meta DBInstanceMeta) (err error) {
	c.Lock()
	defer c.Unlock()

	if !meta.Peers.Verify() {
		return ErrInvalidDBPeersConfig
	}

	// remove previous records
	var oldMeta DBInstanceMeta
	var ok bool

	if oldMeta, ok = c.dbMap[dbID]; ok {
		for _, s := range oldMeta.Peers.Servers {
			delete(c.nodeMap[s.ID], dbID)
		}
	}

	// set new records
	c.dbMap[dbID] = meta

	for _, s := range meta.Peers.Servers {
		c.nodeMap[s.ID][dbID] = true
	}

	// set to persistence
	err = c.persist.SetDatabase(dbID, meta)

	return
}

// Get find database from meta.
func (c *DBServiceMap) Get(dbID proto.DatabaseID) (meta DBInstanceMeta, err error) {
	c.RLock()
	defer c.RUnlock()

	var ok bool

	// get from cache
	if meta, ok = c.dbMap[dbID]; ok {
		return
	}

	// get from persistence
	if meta, err = c.persist.GetDatabase(dbID); err == nil {
		// set from persistence
		c.dbMap[dbID] = meta
	}

	return
}

// Delete removes database from meta.
func (c *DBServiceMap) Delete(dbID proto.DatabaseID) (err error) {
	c.Lock()
	defer c.Unlock()

	var meta DBInstanceMeta
	var ok bool

	// delete from cache
	if meta, ok = c.dbMap[dbID]; ok {
		for _, s := range meta.Peers.Servers {
			delete(c.nodeMap[s.ID], dbID)
		}
	}

	delete(c.dbMap, dbID)

	// delete from persistence
	err = c.persist.DeleteDatabase(dbID)

	return
}

// GetDatabases return database config.
func (c *DBServiceMap) GetDatabases(nodeID proto.NodeID) (dbs map[proto.DatabaseID]DBInstanceMeta, err error) {
	c.RLock()
	defer c.RUnlock()

	dbs = make(map[proto.DatabaseID]DBInstanceMeta)

	for dbID, ok := range c.nodeMap[nodeID] {
		if ok {
			dbs[dbID] = c.dbMap[dbID]
		}
	}

	return
}
