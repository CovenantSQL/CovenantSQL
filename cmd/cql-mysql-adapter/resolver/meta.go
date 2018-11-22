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

package resolver

import (
	"database/sql"
	"fmt"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/pkg/errors"
	"sync"
	"time"
)

const (
	metaRefreshInterval = time.Minute
)

var (
	ErrDBNotExists    = errors.New("database not exists")
	ErrTableNotExists = errors.New("table not exists")
)

type dbHandler interface {
	Query(query string, args ...interface{}) (rows *sql.Rows, err error)
}

type dbMetaHandler struct {
	l            sync.RWMutex
	dbID         string
	conn         dbHandler
	tables       []string
	tableMapping map[string][]string
}

type metaHandler struct {
	l        sync.RWMutex
	dbMap    map[string]*dbMetaHandler
	wg       sync.WaitGroup
	stopCh   chan struct{}
	stopOnce sync.Once
}

func NewMetaHandler() (h *metaHandler) {
	h = &metaHandler{
		dbMap:  make(map[string]*dbMetaHandler),
		stopCh: make(chan struct{}),
	}

	go h.autoReloadMeta()

	return
}

func (h *metaHandler) autoReloadMeta() {
	for {
		select {
		case <-h.stopCh:
			return
		case <-time.After(metaRefreshInterval):
		}
		h.reloadMeta()
	}
}

func (h *metaHandler) stop() {
	h.stopOnce.Do(func() {
		if h.stopCh != nil {
			select {
			case <-h.stopCh:
				close(h.stopCh)
			default:
			}
			h.stopCh = nil
		}
	})
}

func (h *metaHandler) addConn(dbID string, conn dbHandler) {
	h.l.RLock()
	_, exists := h.dbMap[dbID]
	h.l.RUnlock()
	if !exists {
		h.setConn(dbID, conn)
	}
}

func (h *metaHandler) setConn(dbID string, conn dbHandler) {
	h.l.Lock()
	defer h.l.Unlock()

	h.dbMap[dbID] = NewDBMetaHandler(dbID, conn)
}

func (h *metaHandler) getConn(dbID string) (conn dbHandler, exists bool) {
	h.l.RLock()
	defer h.l.RUnlock()

	var dh *dbMetaHandler

	if dh, exists = h.dbMap[dbID]; exists && dh != nil {
		conn = dh.conn
	}

	return
}

func (h *metaHandler) getTables(dbID string) (tables []string, err error) {
	h.l.RLock()
	defer h.l.RUnlock()

	if v := h.dbMap[dbID]; v != nil {
		tables, err = v.cacheGetTables()
	} else {
		err = errors.Wrapf(ErrDBNotExists, "database %v not exists", dbID)
	}

	return
}

func (h *metaHandler) getTable(dbID string, tableName string) (columns []string, err error) {
	h.l.RLock()
	defer h.l.RUnlock()

	if v := h.dbMap[dbID]; v != nil {
		columns, err = v.cacheGetTable(tableName)
	} else {
		err = errors.Wrapf(ErrDBNotExists, "database %v not exists", dbID)
	}

	return
}

func (h *metaHandler) reloadMeta() {
	h.l.RLock()
	defer h.l.RUnlock()

	for _, v := range h.dbMap {
		v.reloadMeta()
	}
}

func NewDBMetaHandler(dbID string, conn dbHandler) *dbMetaHandler {
	return &dbMetaHandler{
		tableMapping: make(map[string][]string),
		dbID:         dbID,
		conn:         conn,
	}
}

func (h *dbMetaHandler) reloadMeta() {
	h.getTables()
	h.l.RLock()
	defer h.l.RUnlock()
	for t := range h.tableMapping {
		delete(h.tableMapping, t)
	}
}

func (h *dbMetaHandler) cacheGetTables() (tables []string, err error) {
	h.l.RLock()
	tables = append(tables, h.tables...)
	h.l.RUnlock()
	if len(tables) > 0 {
		return
	}

	return h.getTables()
}

func (h *dbMetaHandler) getTables() (tables []string, err error) {
	h.l.Lock()
	defer h.l.Unlock()

	rows, err := h.conn.Query("SHOW TABLES")
	if err != nil {
		log.WithError(err).Debug("load tables from database failed")
		return
	}

	defer rows.Close()

	var (
		tableName  string
		tempTables []string
	)

	for rows.Next() {
		if err = rows.Scan(&tableName); err != nil {
			return
		}
		tempTables = append(tempTables, tableName)
	}

	tables = tempTables
	h.tables = tables

	return
}

func (h *dbMetaHandler) cacheGetTable(tableName string) (columns []string, err error) {
	h.l.RLock()
	columns = append(columns, h.tableMapping[tableName]...)
	h.l.RUnlock()
	if len(columns) > 0 {
		return
	}

	// trigger get tables
	h.getTables()

	return h.getTable(tableName)
}

func (h *dbMetaHandler) getTable(tableName string) (columns []string, err error) {
	h.l.Lock()
	defer h.l.Unlock()
	rows, err := h.conn.Query(fmt.Sprintf("DESC `%s`", tableName))
	if err != nil {
		log.WithError(err).Debug("load columns from database table failed")
		return
	}

	defer rows.Close()

	var (
		f1, f3, f4, f5, f6 interface{}
		columnName         string
		tempColumns        []string
	)

	for rows.Next() {
		if err = rows.Scan(&f1, &columnName, &f3, &f4, &f5, &f6); err != nil {
			return
		}
		tempColumns = append(tempColumns, columnName)
	}

	if len(tempColumns) == 0 {
		err = errors.Wrapf(ErrTableNotExists, "table %v.%s not exists", h.dbID, tableName)
		return
	}

	columns = tempColumns
	h.tableMapping[tableName] = columns

	return
}
