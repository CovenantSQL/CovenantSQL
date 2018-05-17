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
