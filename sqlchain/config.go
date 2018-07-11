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

package sqlchain

import (
	"time"

	"gitlab.com/thunderdb/ThunderDB/kayak"
	"gitlab.com/thunderdb/ThunderDB/proto"
	ct "gitlab.com/thunderdb/ThunderDB/sqlchain/types"
	wt "gitlab.com/thunderdb/ThunderDB/worker/types"
)

// Config represents a sql-chain config.
type Config struct {
	DatabaseID proto.DatabaseID
	DataFile   string

	Genesis *ct.Block
	Period  time.Duration
	Tick    time.Duration

	MuxService *MuxService
	Peers      *kayak.Peers
	Server     *kayak.Server

	// Price sets query price in gases.
	Price map[wt.QueryType]uint32

	// QueryTTL sets the unacknowledged query TTL in block periods.
	QueryTTL int32
}
