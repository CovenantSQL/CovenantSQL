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

package blockproducer

import (
	"database/sql"
	pi "github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

// This file provides methods set for chain state read/write.

// loadBlock loads a BPBlock from chain storage.
func (c *Chain) loadBlock(h hash.Hash) (b *types.BPBlock, err error) {
	return loadBlock(c.storage, h)
}

func (c *Chain) fetchLastIrreversibleBlock() (
	b *types.BPBlock, count uint32, height uint32, err error,
) {
	var node = c.lastIrreversibleBlock()
	if b = node.load(); b != nil {
		height = node.height
		count = node.count
		return
	}
	// Not cached, read from database
	if b, err = c.loadBlock(node.hash); err != nil {
		return
	}
	height = node.height
	count = node.count
	return
}

func (c *Chain) fetchBlockByHeight(h uint32) (b *types.BPBlock, count uint32, err error) {
	var node = c.head().ancestor(h)
	// Not found
	if node == nil {
		return
	}
	// OK, and block is cached
	if b = node.load(); b != nil {
		count = node.count
		return
	}
	// Not cached, read from database
	if b, err = c.loadBlock(node.hash); err != nil {
		return
	}
	count = node.count
	return
}

func (c *Chain) fetchBlockByCount(count uint32) (b *types.BPBlock, height uint32, err error) {
	var node = c.head().ancestorByCount(count)
	// Not found
	if node == nil {
		return
	}
	// OK, and block is cached
	if b = node.load(); b != nil {
		height = node.height
		return
	}
	// Not cached, read from database
	if b, err = c.loadBlock(node.hash); err != nil {
		return
	}
	height = node.height
	return
}

func (c *Chain) nextNonce(addr proto.AccountAddress) (n pi.AccountNonce, err error) {
	c.RLock()
	defer c.RUnlock()
	n, err = c.headBranch.preview.nextNonce(addr)
	log.Debugf("nextNonce addr: %s, nonce %d", addr.String(), n)
	return
}

func (c *Chain) loadAccountTokenBalance(addr proto.AccountAddress, tt types.TokenType) (balance uint64, ok bool) {
	c.RLock()
	defer c.RUnlock()
	return c.immutable.loadAccountTokenBalance(addr, tt)
}

func (c *Chain) loadSQLChainProfile(databaseID proto.DatabaseID) (profile *types.SQLChainProfile, ok bool) {
	c.RLock()
	defer c.RUnlock()
	profile, ok = c.immutable.loadSQLChainObject(databaseID)
	if !ok {
		log.Warnf("cannot load sqlchain profile with databaseID: %s", databaseID)
		return
	}
	return
}

func (c *Chain) loadSQLChainProfiles(addr proto.AccountAddress) []*types.SQLChainProfile {
	c.RLock()
	defer c.RUnlock()
	return c.immutable.loadROSQLChains(addr)
}

func (c *Chain) queryTxState(hash hash.Hash) (state pi.TransactionState, err error) {
	c.RLock()
	defer c.RUnlock()
	var ok bool

	if state, ok = c.headBranch.queryTxState(hash); ok {
		return
	}

	var (
		count    int
		querySQL = `SELECT COUNT(*) FROM "indexed_transactions" WHERE "hash" = ?`
	)
	if err = c.storage.Reader().QueryRow(querySQL, hash.String()).Scan(&count); err != nil {
		return pi.TransactionStateNotFound, err
	}

	if count > 0 {
		return pi.TransactionStateConfirmed, nil
	}

	return pi.TransactionStateNotFound, nil
}

func (c *Chain) queryAccountSQLChainProfiles(account proto.AccountAddress) (profiles []*types.SQLChainProfile, err error) {
	var dbs []proto.DatabaseID

	dbs, err = func() (dbs []proto.DatabaseID, err error) {
		c.RLock()
		defer c.RUnlock()

		var (
			id       string
			rows     *sql.Rows
			querySQL = `SELECT "id" FROM "indexed_shardChains" WHERE "account" = ?`
		)

		rows, err = c.storage.Reader().Query(querySQL, account.String())

		if err != nil {
			return
		}

		defer func() {
			_ = rows.Close()
		}()

		for rows.Next() {
			err = rows.Scan(&id)
			if err != nil {
				return
			}
		}

		return
	}()

	if err != nil {
		return
	}

	var (
		profile *types.SQLChainProfile
		ok      bool
	)

	for _, db := range dbs {
		profile, ok = c.loadSQLChainProfile(db)
		if ok {
			profiles = append(profiles, profile)
		}
	}

	return
}

func (c *Chain) immutableNextNonce(addr proto.AccountAddress) (n pi.AccountNonce, err error) {
	c.RLock()
	defer c.RUnlock()
	return c.immutable.nextNonce(addr)
}
