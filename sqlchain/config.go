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
	ct "gitlab.com/thunderdb/ThunderDB/sqlchain/types"
	wt "gitlab.com/thunderdb/ThunderDB/worker/types"
)

// Config represents a sql-chain config.
type Config struct {
	DataDir string

	Genesis        *ct.Block
	Period         time.Duration
	TimeResolution time.Duration

	Peers  *kayak.Peers
	Server *kayak.Server

	// Price sets query price in gases.
	Price map[wt.QueryType]uint32

	// QueryTTL sets the unacknowledged query TTL in block periods.
	QueryTTL int32
}

// GetHeightFromTime calculates the height with this sql-chain config of a given time reading.
func (c *Config) GetHeightFromTime(t time.Time) int32 {
	return int32(t.Sub(c.Genesis.SignedHeader.Timestamp) / c.Period)
}

// GetQueryGas gets the consumption of gas for a specified query type.
func (c *Config) GetQueryGas(t wt.QueryType) uint32 {
	return c.Price[t]
}
