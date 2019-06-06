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

package handler

import (
	"context"
	"database/sql"
	"sync"

	"github.com/pkg/errors"

	"github.com/CovenantSQL/CovenantSQL/client"
	"github.com/CovenantSQL/CovenantSQL/cmd/cql-mysql-adapter/cursor"
	"github.com/CovenantSQL/CovenantSQL/cmd/cql-mysql-adapter/resolver"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
)

// Handler defines the mysql adapter query handler.
type Handler struct {
	l sync.Mutex
	r *resolver.Resolver
}

// NewHandler returns the new mysql adapter handler instance.
func NewHandler() *Handler {
	return &Handler{
		r: resolver.NewResolver(),
	}
}

// NewHandlerWithResolver returns the new mysql handler with resolver.
func NewHandlerWithResolver(r *resolver.Resolver) *Handler {
	return &Handler{
		r: r,
	}
}

// EnsureDatabase returns whether a database is valid or not.
func (h *Handler) EnsureDatabase(dbID string) (err error) {
	_, err = h.ensureDatabase(dbID)
	return
}

// Resolve resolves query to ast and various parse results.
func (h *Handler) Resolve(user string, dbID string, query string) (q cursor.Query, err error) {
	if _, err = h.ensureDatabase(dbID); err != nil {
		return
	}
	if q, err = h.r.ResolveSingleQuery(dbID, query); err != nil {
		return
	}
	return
}

// Query executes a resolved read query.
func (h *Handler) Query(q cursor.Query, args ...interface{}) (rh hash.Hash, rows cursor.Rows, err error) {
	if !q.IsRead() {
		err = errors.Wrapf(resolver.ErrQueryLogicError, "not a read query")
		return
	}
	return h.QueryString(q.GetDatabase(), q.GetQuery(), args...)
}

// Exec executes a resolved write query.
func (h *Handler) Exec(q cursor.Query, args ...interface{}) (rh hash.Hash, res sql.Result, err error) {
	if q.IsRead() {
		err = errors.Wrapf(resolver.ErrQueryLogicError, "not a write query")
		return
	}
	if q.IsDDL() {
		defer h.r.ReloadMeta()
	}
	return h.ExecString(q.GetDatabase(), q.GetQuery(), args...)
}

// QueryString executes a string query without resolving.
func (h *Handler) QueryString(dbID string, query string, args ...interface{}) (rh hash.Hash, rows cursor.Rows, err error) {
	var db resolver.DBHandler
	if db, err = h.ensureDatabase(dbID); err != nil {
		return
	}
	ctx := client.WithReceipt(context.Background())
	rows, err = db.QueryContext(ctx, query, args...)
	if r, ok := client.GetReceipt(ctx); ok {
		rh = r.RequestHash
	}
	return
}

// ExecString executes a string query without resolving.
func (h *Handler) ExecString(dbID string, query string, args ...interface{}) (rh hash.Hash, result sql.Result, err error) {
	var db resolver.DBHandler
	if db, err = h.ensureDatabase(dbID); err != nil {
		return
	}
	ctx := client.WithReceipt(context.Background())
	result, err = db.ExecContext(ctx, query, args...)
	if r, ok := client.GetReceipt(ctx); ok {
		rh = r.RequestHash
	}
	return
}

func (h *Handler) ensureDatabase(dbID string) (db resolver.DBHandler, err error) {
	var exists bool
	if db, exists = h.r.GetDB(dbID); !exists {
		// new connection
		cfg := client.NewConfig()
		cfg.DatabaseID = dbID
		if db, err = sql.Open("covenantsql", cfg.FormatDSN()); err != nil {
			return
		}
		if !h.r.RegisterDB(dbID, db) {
			db.Close()
		}
		db, _ = h.r.GetDB(dbID)
	}

	return
}

// Close close the resolver.
func (h *Handler) Close() {
	h.r.Close()
}
