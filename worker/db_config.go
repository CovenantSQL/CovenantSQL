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

package worker

import (
	"time"

	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/sqlchain"
)

// DBConfig defines the database config.
type DBConfig struct {
	DatabaseID             proto.DatabaseID
	RootDir                string
	DataDir                string
	KayakMux               *DBKayakMuxService
	ChainMux               *sqlchain.MuxService
	MaxWriteTimeGap        time.Duration
	EncryptionKey          string
	SpaceLimit             uint64
	UpdateBlockCount       uint64
	LastBillingHeight      int32
	UseEventualConsistency bool
	ConsistencyLevel       float64
	IsolationLevel         int
	SlowQueryTime          time.Duration
}
