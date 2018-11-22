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
	"strings"
	"sync"
	"time"

	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/pkg/errors"
)

const (
	MetaRefreshInterval = time.Minute
)

var (
	ErrDBNotExists    = errors.New("database not exists")
	ErrTableNotExists = errors.New("Table not exists")
)

type DBHandler interface {
	Query(query string, args ...interface{}) (rows *sql.Rows, err error)
}

type DBMetaHandler struct {
	l            sync.RWMutex
	dbID         string
	conn         DBHandler
	tables       []string
	tableMapping map[string][]string
}

type MetaHandler struct {
	l        sync.RWMutex
	dbMap    map[string]*DBMetaHandler
	wg       sync.WaitGroup
	stopCh   chan struct{}
	stopOnce sync.Once
}

func NewMetaHandler() (h *MetaHandler) {
	h = &MetaHandler{
		dbMap:  make(map[string]*DBMetaHandler),
		stopCh: make(chan struct{}),
	}

	go h.autoReloadMeta()

	return
}

func (h *MetaHandler) autoReloadMeta() {
	for {
		select {
		case <-h.stopCh:
			return
		case <-time.After(MetaRefreshInterval):
		}
		h.ReloadMeta()
	}
}

func (h *MetaHandler) Stop() {
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

func (h *MetaHandler) AddConn(dbID string, conn DBHandler) {
	h.l.RLock()
	_, exists := h.dbMap[dbID]
	h.l.RUnlock()
	if !exists {
		h.SetConn(dbID, conn)
	}
}

func (h *MetaHandler) SetConn(dbID string, conn DBHandler) {
	h.l.Lock()
	defer h.l.Unlock()

	h.dbMap[dbID] = NewDBMetaHandler(dbID, conn)
}

func (h *MetaHandler) GetConn(dbID string) (conn DBHandler, exists bool) {
	h.l.RLock()
	defer h.l.RUnlock()

	var dh *DBMetaHandler

	if dh, exists = h.dbMap[dbID]; exists && dh != nil {
		conn = dh.conn
	}

	return
}

func (h *MetaHandler) GetTables(dbID string) (tables []string, err error) {
	h.l.RLock()
	defer h.l.RUnlock()

	if v := h.dbMap[dbID]; v != nil {
		tables, err = v.CacheGetTables()
	} else {
		err = errors.Wrapf(ErrDBNotExists, "database %v not exists", dbID)
	}

	return
}

func (h *MetaHandler) GetTable(dbID string, tableName string) (columns []string, err error) {
	h.l.RLock()
	defer h.l.RUnlock()

	if v := h.dbMap[dbID]; v != nil {
		columns, err = v.CacheGetTable(tableName)
	} else {
		err = errors.Wrapf(ErrDBNotExists, "database %v not exists", dbID)
	}

	return
}

func (h *MetaHandler) ReloadMeta() {
	h.l.RLock()
	defer h.l.RUnlock()

	for _, v := range h.dbMap {
		v.ReloadMeta()
	}
}

func NewDBMetaHandler(dbID string, conn DBHandler) *DBMetaHandler {
	return &DBMetaHandler{
		tableMapping: make(map[string][]string),
		dbID:         dbID,
		conn:         conn,
	}
}

func (h *DBMetaHandler) ReloadMeta() {
	h.GetTables()
	h.l.RLock()
	defer h.l.RUnlock()
	for t := range h.tableMapping {
		delete(h.tableMapping, t)
	}
}

func (h *DBMetaHandler) CacheGetTables() (tables []string, err error) {
	h.l.RLock()
	tables = append(tables, h.tables...)
	h.l.RUnlock()
	if len(tables) > 0 {
		return
	}

	return h.GetTables()
}

func (h *DBMetaHandler) GetTables() (tables []string, err error) {
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

func (h *DBMetaHandler) CacheGetTable(tableName string) (columns []string, err error) {
	h.l.RLock()
	tableName = strings.ToLower(tableName)
	columns = append(columns, h.tableMapping[tableName]...)
	h.l.RUnlock()
	if len(columns) > 0 {
		return
	}

	// trigger get tables
	h.GetTables()

	return h.GetTable(tableName)
}

func (h *DBMetaHandler) GetTable(tableName string) (columns []string, err error) {
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
	h.tableMapping[strings.ToLower(tableName)] = columns

	return
}
