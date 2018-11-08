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

package blockproducer

import (
	"sync"

	"github.com/CovenantSQL/CovenantSQL/proto"
	wt "github.com/CovenantSQL/CovenantSQL/worker/types"
	"github.com/pkg/errors"
)

// DBMetaPersistence defines database meta persistence api.
type DBMetaPersistence interface {
	GetDatabase(dbID proto.DatabaseID) (wt.ServiceInstance, error)
	SetDatabase(meta wt.ServiceInstance) error
	DeleteDatabase(dbID proto.DatabaseID) error
	GetAllDatabases() ([]wt.ServiceInstance, error)
}

// DBServiceMap defines database instance meta.
type DBServiceMap struct {
	dbMap   map[proto.DatabaseID]wt.ServiceInstance
	nodeMap map[proto.NodeID]map[proto.DatabaseID]bool
	persist DBMetaPersistence
	sync.RWMutex
}

// InitServiceMap init database service map.
func InitServiceMap(persistImpl DBMetaPersistence) (s *DBServiceMap, err error) {
	s = &DBServiceMap{
		persist: persistImpl,
		dbMap:   make(map[proto.DatabaseID]wt.ServiceInstance),
		nodeMap: make(map[proto.NodeID]map[proto.DatabaseID]bool),
	}

	// load from persistence
	s.Lock()
	defer s.Unlock()

	var allDatabases []wt.ServiceInstance

	if allDatabases, err = s.persist.GetAllDatabases(); err != nil {
		return
	}

	for _, meta := range allDatabases {
		s.dbMap[meta.DatabaseID] = meta

		for _, server := range meta.Peers.Servers {
			if s.nodeMap[server] == nil {
				s.nodeMap[server] = make(map[proto.DatabaseID]bool)
			}
			s.nodeMap[server][meta.DatabaseID] = true
		}
	}

	return
}

// Set add database to meta.
func (c *DBServiceMap) Set(meta wt.ServiceInstance) (err error) {
	c.Lock()
	defer c.Unlock()

	if err = meta.Peers.Verify(); err != nil {
		return errors.Wrap(err, "verify peers failed")
	}

	// remove previous records
	var oldMeta wt.ServiceInstance
	var ok bool

	if oldMeta, ok = c.dbMap[meta.DatabaseID]; ok {
		for _, s := range oldMeta.Peers.Servers {
			if c.nodeMap[s] != nil {
				delete(c.nodeMap[s], meta.DatabaseID)
			}
		}
	}

	// set new records
	c.dbMap[meta.DatabaseID] = meta

	for _, s := range meta.Peers.Servers {
		if c.nodeMap[s] == nil {
			c.nodeMap[s] = make(map[proto.DatabaseID]bool)
		}
		c.nodeMap[s][meta.DatabaseID] = true
	}

	// set to persistence
	err = c.persist.SetDatabase(meta)

	return
}

// Get find database from meta.
func (c *DBServiceMap) Get(dbID proto.DatabaseID) (meta wt.ServiceInstance, err error) {
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

	var meta wt.ServiceInstance
	var ok bool

	// delete from cache
	if meta, ok = c.dbMap[dbID]; ok {
		for _, s := range meta.Peers.Servers {
			if c.nodeMap[s] != nil {
				delete(c.nodeMap[s], dbID)
			}
		}
	}

	delete(c.dbMap, dbID)

	// delete from persistence
	err = c.persist.DeleteDatabase(dbID)

	return
}

// GetDatabases return database config.
func (c *DBServiceMap) GetDatabases(nodeID proto.NodeID) (dbs []wt.ServiceInstance, err error) {
	c.RLock()
	defer c.RUnlock()

	dbs = make([]wt.ServiceInstance, 0)

	for dbID, ok := range c.nodeMap[nodeID] {
		if ok {
			var db wt.ServiceInstance
			if db, ok = c.dbMap[dbID]; ok {
				dbs = append(dbs, db)
			}
		}
	}

	return
}
