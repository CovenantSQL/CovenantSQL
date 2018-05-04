/*
 * MIT License
 *
 * Copyright (c) 2016-2018. ThunderDB
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

// Package sqlchain provides a blockchain implementation for database state tracking.
package sqlchain

import (
	"github.com/thunderdb/ThunderDB/common"
)

// QueryType enumerates the basic types of SQL query, i.e., ReadQuery or WriteQuery.
// XXX(leventeliu): this may be defined in database implementation modules later.
type QueryType int

const (
	// ReadQuery represents a read query like SELECT
	ReadQuery QueryType = iota
	// WriteQuery represents a write query like UPDATE/DELETE
	WriteQuery
)

// Query represents a SQL query log
type Query struct {
	TxnID common.UUID
}

// Queries is a Query (reference) array
type Queries []*Query
