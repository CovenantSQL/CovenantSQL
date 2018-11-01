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

package sqlchain

import (
	"time"

	"github.com/CovenantSQL/CovenantSQL/proto"
	ct "github.com/CovenantSQL/CovenantSQL/sqlchain/types"
	wt "github.com/CovenantSQL/CovenantSQL/worker/types"
)

// Config represents a sql-chain config.
type Config struct {
	DatabaseID proto.DatabaseID
	DataFile   string

	Genesis *ct.Block
	Period  time.Duration
	Tick    time.Duration

	MuxService *MuxService
	Peers      *proto.Peers
	Server     proto.NodeID

	// Price sets query price in gases.
	Price           map[wt.QueryType]uint64
	ProducingReward uint64
	BillingPeriods  int32

	// QueryTTL sets the unacknowledged query TTL in block periods.
	QueryTTL int32
}
