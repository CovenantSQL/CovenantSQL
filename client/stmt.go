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

package client

import (
	"context"
	"database/sql/driver"
	"sync/atomic"
)

type stmt struct {
	c       *conn
	closed  int32
	pattern string
}

func newStmt(c *conn, query string) (s *stmt) {
	s = &stmt{c: c, pattern: query}
	return
}

// Query executes a query that may return rows, such as SELECT.
func (s *stmt) Query(args []driver.Value) (driver.Rows, error) {
	// convert bind parameters to named bind parameters.
	return s.QueryContext(context.Background(), convertOldArgs(args))
}

// Exec executes a query that doesn't return rows, such as INSERT.
func (s *stmt) Exec(args []driver.Value) (driver.Result, error) {
	// convert bind parameters to named bind parameters.
	return s.ExecContext(context.Background(), convertOldArgs(args))
}

// QueryContext implements the driver.StmtQueryContext.QueryContext.
func (s *stmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	// build query from pattern
	if atomic.LoadInt32(&s.closed) != 0 || s.c == nil {
		return nil, driver.ErrBadConn
	}

	return s.c.QueryContext(ctx, s.pattern, args)
}

// ExecContext implements the driver.StmtExecContext.ExecContext.
func (s *stmt) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	// build query from pattern
	if atomic.LoadInt32(&s.closed) != 0 || s.c == nil {
		return nil, driver.ErrBadConn
	}

	return s.c.ExecContext(ctx, s.pattern, args)
}

// Close closes the statement.
func (s *stmt) Close() error {
	if atomic.CompareAndSwapInt32(&s.closed, 0, 1) {
		s.c = nil
	}
	return nil
}

// NumInput returns the number of placeholder parameters.
func (s *stmt) NumInput() int {
	// Sanity check of the placeholder argument count is not required, do it remotely.
	return -1
}

func convertOldArgs(args []driver.Value) (dargs []driver.NamedValue) {
	dargs = make([]driver.NamedValue, len(args))

	for i, v := range args {
		dargs[i] = driver.NamedValue{
			Ordinal: i + 1,
			Value:   v,
		}
	}

	return
}
