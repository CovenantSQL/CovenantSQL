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

package xenomint

import (
	"database/sql"
	"time"

	ca "github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	xi "github.com/CovenantSQL/CovenantSQL/xenomint/interfaces"
	xs "github.com/CovenantSQL/CovenantSQL/xenomint/sqlite"
)

// Chain defines the xenomint chain structure.
type Chain struct {
	state *State
	// Cached fields
	priv *ca.PrivateKey
}

// NewChain returns new chain instance.
func NewChain(filename string) (c *Chain, err error) {
	var (
		strg xi.Storage
		priv *ca.PrivateKey
	)
	// generate empty nodeId
	nodeID := proto.NodeID("0000000000000000000000000000000000000000000000000000000000000000")

	// TODO(leventeliu): add multiple storage engine support.
	if strg, err = xs.NewSqlite(filename); err != nil {
		return
	}
	if priv, err = kms.GetLocalPrivateKey(); err != nil {
		return
	}
	c = &Chain{
		state: NewState(sql.LevelReadUncommitted, nodeID, strg),
		priv:  priv,
	}
	return
}

// Query queries req from local chain state and returns the query results in resp.
func (c *Chain) Query(req *types.Request) (resp *types.Response, err error) {
	var (
		ref   *QueryTracker
		start = time.Now()

		queried, signed, updated time.Duration
	)
	defer func() {
		var fields = log.Fields{}
		if queried > 0 {
			fields["1#queried"] = float64(queried.Nanoseconds()) / 1000
		}
		if signed > 0 {
			fields["2#signed"] = float64((signed - queried).Nanoseconds()) / 1000
		}
		if updated > 0 {
			fields["3#updated"] = float64((updated - signed).Nanoseconds()) / 1000
		}
		log.WithFields(fields).Debug("Chain.Query duration stat (us)")
	}()
	if ref, resp, err = c.state.Query(req, true); err != nil {
		return
	}
	queried = time.Since(start)
	if err = resp.BuildHash(); err != nil {
		return
	}
	signed = time.Since(start)
	ref.UpdateResp(resp)
	updated = time.Since(start)
	return
}

// Stop stops chain workers and RPC service.
func (c *Chain) Stop() (err error) {
	// Close all opened resources
	return c.state.Close(true)
}
