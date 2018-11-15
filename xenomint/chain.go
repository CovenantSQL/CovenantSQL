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
	ca "github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/types"
	xi "github.com/CovenantSQL/CovenantSQL/xenomint/interfaces"
	xs "github.com/CovenantSQL/CovenantSQL/xenomint/sqlite"
	xt "github.com/CovenantSQL/CovenantSQL/xenomint/types"
)

const (
	inCommandBufferLength  = 100000
	outCommandBufferLength = 100000
)

type applyRequest struct {
	request  *types.Request
	response *types.Response
}

type blockNode struct {
	parent *blockNode
	// Cached block fields
	hash   hash.Hash
	count  int32
	height int32
	// Cached block object, may be nil
	block *xt.Block
}

// Chain defines the xenomint chain structure.
type Chain struct {
	state *State
	// Cached fields
	priv *ca.PrivateKey
}

// NewChain returns new chain instance.
func NewChain(filename string) (c *Chain, err error) {
	var (
		strg  xi.Storage
		state *State
		priv  *ca.PrivateKey
	)
	// generate empty nodeId
	nodeID := proto.NodeID("0000000000000000000000000000000000000000000000000000000000000000")

	// TODO(leventeliu): add multiple storage engine support.
	if strg, err = xs.NewSqlite(filename); err != nil {
		return
	}
	if state, err = NewState(nodeID, strg); err != nil {
		return
	}
	if priv, err = kms.GetLocalPrivateKey(); err != nil {
		return
	}
	c = &Chain{
		state: state,
		priv:  priv,
	}
	return
}

// Query queries req from local chain state and returns the query results in resp.
func (c *Chain) Query(req *types.Request) (resp *types.Response, err error) {
	var ref *QueryTracker
	if ref, resp, err = c.state.Query(req); err != nil {
		return
	}
	if err = resp.Sign(c.priv); err != nil {
		return
	}
	ref.UpdateResp(resp)
	return
}

// Stop stops chain workers and RPC service.
func (c *Chain) Stop() (err error) {
	// Close all opened resources
	return c.state.Close(true)
}
