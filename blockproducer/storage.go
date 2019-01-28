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
	"bytes"
	"database/sql"
	"encoding/json"

	pi "github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	xi "github.com/CovenantSQL/CovenantSQL/xenomint/interfaces"
	xs "github.com/CovenantSQL/CovenantSQL/xenomint/sqlite"
	"github.com/pkg/errors"
)

var (
	ddls = [...]string{
		// Chain state tables
		`CREATE TABLE IF NOT EXISTS "blocks" (
			"height"    INT,
			"hash"		TEXT,
			"parent"	TEXT,
			"encoded"	BLOB,
			UNIQUE ("hash")
		);`,

		`CREATE TABLE IF NOT EXISTS "txPool" (
			"type"		INT,
			"hash"		TEXT,
			"encoded"	BLOB,
			UNIQUE ("hash")
		);`,

		`CREATE TABLE IF NOT EXISTS "irreversible" (
			"id"		INT,
			"hash"		TEXT,
			UNIQUE ("id")
		);`,

		// Meta state tables
		`CREATE TABLE IF NOT EXISTS "accounts" (
			"address"	TEXT,
			"encoded"	BLOB,
			UNIQUE ("address")
		);`,

		`CREATE TABLE IF NOT EXISTS "shardChain" (
			"address"	TEXT,
			"id"		TEXT,
			"encoded"	BLOB,
			UNIQUE ("address", "id")
		);`,

		`CREATE TABLE IF NOT EXISTS "provider" (
			"address"	TEXT,
			"encoded"	BLOB,
			UNIQUE ("address")
		);`,

		`CREATE TABLE IF NOT EXISTS "indexed_blocks" (
			"height"		INTEGER PRIMARY KEY,
			"hash"			TEXT,
			"timestamp"		INTEGER,
			"version"		INTEGER,
			"producer"		TEXT,
			"merkle_root"	TEXT,
			"parent"		TEXT,
			"tx_count"		INTEGER
		);`,

		`CREATE INDEX IF NOT EXISTS "idx__indexed_blocks__hash" ON "indexed_blocks" ("hash");`,
		`CREATE INDEX IF NOT EXISTS "idx__indexed_blocks__timestamp" ON "indexed_blocks" ("timestamp" DESC);`,

		`CREATE TABLE IF NOT EXISTS "indexed_transactions" (
			"block_height"	INTEGER,
			"tx_index"		INTEGER,
			"hash"			TEXT,
			"block_hash"	TEXT,
			"timestamp"		INTEGER,
			"tx_type"		INTEGER,
			"address"		TEXT,
			"raw"			TEXT,
			PRIMARY KEY ("block_height", "tx_index")
		);`,

		`CREATE INDEX IF NOT EXISTS "idx__indexed_transactions__hash" ON "indexed_transactions" ("hash");`,
		`CREATE INDEX IF NOT EXISTS "idx__indexed_transactions__block_hash" ON "indexed_transactions" ("block_hash");`,
		`CREATE INDEX IF NOT EXISTS "idx__indexed_transactions__timestamp" ON "indexed_transactions" ("timestamp" DESC);`,
		`CREATE INDEX IF NOT EXISTS "idx__indexed_transactions__tx_type__timestamp" ON "indexed_transactions" ("tx_type", "timestamp" DESC);`,
		`CREATE INDEX IF NOT EXISTS "idx__indexed_transactions__address__timestamp" ON "indexed_transactions" ("address", "timestamp" DESC);`,
	}
)

type storageProcedure func(tx *sql.Tx) error
type storageCallback func()

func store(st xi.Storage, sps []storageProcedure, cb storageCallback) (err error) {
	var tx *sql.Tx
	// BEGIN
	if tx, err = st.Writer().Begin(); err != nil {
		return
	}
	log.Debugf("started database tx %p", tx)
	// ROLLBACK on failure
	defer tx.Rollback()
	// WRITE
	for _, sp := range sps {
		if err = sp(tx); err != nil {
			return
		}
	}
	// CALLBACK: MUST NOT FAIL
	if cb != nil {
		cb()
		log.Debugf("invoked storage callback %p in tx %p", cb, tx)
	}
	// COMMIT
	if err = tx.Commit(); err != nil {
		log.WithError(err).Fatal("failed to commit storage transaction")
	}
	log.Debugf("committed database tx %p", tx)
	return
}

func errPass(err error) storageProcedure {
	return func(_ *sql.Tx) error {
		return err
	}
}

func openStorage(path string) (st xi.Storage, err error) {
	var ierr error
	if st, ierr = xs.NewSqlite(path); ierr != nil {
		return
	}
	for _, v := range ddls {
		if _, ierr = st.Writer().Exec(v); ierr != nil {
			err = errors.Wrap(ierr, v)
			return
		}
	}
	return
}

func addBlock(height uint32, b *types.BPBlock) storageProcedure {
	var (
		enc *bytes.Buffer
		err error
	)
	if enc, err = utils.EncodeMsgPack(b); err != nil {
		return errPass(err)
	}
	return func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(`INSERT OR REPLACE INTO "blocks" ("height", "hash", "parent", "encoded")
	VALUES (?, ?, ?, ?)`,
			height,
			b.BlockHash().String(),
			b.ParentHash().String(),
			enc.Bytes())
		return
	}
}

func addTx(t pi.Transaction) storageProcedure {
	var (
		enc *bytes.Buffer
		err error
	)
	if enc, err = utils.EncodeMsgPack(t); err != nil {
		return errPass(err)
	}
	return func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(`INSERT OR REPLACE INTO "txPool" ("type", "hash", "encoded")
	VALUES (?, ?, ?)`,
			uint32(t.GetTransactionType()),
			t.Hash().String(),
			enc.Bytes())
		return err
	}
}

func buildBlockIndex(height uint32, b *types.BPBlock) storageProcedure {
	return func(tx *sql.Tx) (err error) {
		var p = b.Producer()
		if _, err = tx.Exec(`INSERT OR REPLACE INTO "indexed_blocks"
			("height", "hash", "timestamp", "version", "producer",
			"merkle_root", "parent", "tx_count") VALUES (?,?,?,?,?,?,?,?)`,
			height,
			b.BlockHash().String(),
			b.Timestamp().UnixNano(),
			b.SignedHeader.Version,
			p.String(),
			b.SignedHeader.MerkleRoot.String(),
			b.ParentHash().String(),
			len(b.Transactions),
		); err != nil {
			return err
		}

		for txIndex, t := range b.Transactions {
			var (
				addr   = t.GetAccountAddress()
				raw, _ = json.Marshal(t)
			)
			if _, err := tx.Exec(`INSERT OR REPLACE INTO "indexed_transactions"
			("block_height", "tx_index", "hash", "block_hash", "timestamp",
			"tx_type", "address", "raw") VALUES (?,?,?,?,?,?,?,?)`,
				height,
				txIndex,
				t.Hash().String(),
				b.BlockHash().String(),
				t.GetTimestamp().UnixNano(),
				t.GetTransactionType(),
				addr.String(),
				string(raw),
			); err != nil {
				return err
			}
		}
		return nil
	}
}

func updateIrreversible(h hash.Hash) storageProcedure {
	return func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(`INSERT OR REPLACE INTO "irreversible" ("id", "hash")
	VALUES (?, ?)`, 0, h.String())
		return
	}
}

func deleteTxs(txs []pi.Transaction) storageProcedure {
	var hs = make([]hash.Hash, len(txs))
	for i, v := range txs {
		hs[i] = v.Hash()
	}
	return func(tx *sql.Tx) (err error) {
		var stmt *sql.Stmt
		if stmt, err = tx.Prepare(`DELETE FROM "txPool" WHERE "hash"=?`); err != nil {
			return
		}
		defer stmt.Close()
		for _, v := range hs {
			if _, err = stmt.Exec(v.String()); err != nil {
				return
			}
		}
		return
	}
}

func updateAccount(account *types.Account) storageProcedure {
	var (
		enc *bytes.Buffer
		err error
	)
	if enc, err = utils.EncodeMsgPack(account); err != nil {
		return errPass(err)
	}
	return func(tx *sql.Tx) (err error) {
		log.WithFields(log.Fields{
			"account_address":  account.Address.String(),
			"account_nonce":    account.NextNonce,
			"account_balances": account.TokenBalance,
		}).Debug("updating account")
		_, err = tx.Exec(`INSERT OR REPLACE INTO "accounts" ("address", "encoded")
	VALUES (?, ?)`,
			account.Address.String(),
			enc.Bytes())
		return
	}
}

func deleteAccount(address proto.AccountAddress) storageProcedure {
	return func(tx *sql.Tx) (err error) {
		log.WithFields(log.Fields{
			"account_address": address.String(),
		}).Debug("deleting account")
		_, err = tx.Exec(`DELETE FROM "accounts" WHERE "address"=?`, address.String())
		return
	}
}

func updateShardChain(profile *types.SQLChainProfile) storageProcedure {
	var (
		enc *bytes.Buffer
		err error
	)
	if enc, err = utils.EncodeMsgPack(profile); err != nil {
		return errPass(err)
	}
	return func(tx *sql.Tx) (err error) {
		log.WithFields(log.Fields{
			"profile_owner":         profile.Owner.String(),
			"profile_address":       profile.Address.String(),
			"profile_database_id":   profile.ID,
			"profile_token_type":    profile.TokenType,
			"profile_miners_number": len(profile.Miners),
		}).Debug("updating profile")
		_, err = tx.Exec(`INSERT OR REPLACE INTO "shardChain" ("address", "id", "encoded")
	VALUES (?, ?, ?)`,
			profile.Address.String(),
			string(profile.ID),
			enc.Bytes())
		return
	}
}

func deleteShardChain(id proto.DatabaseID) storageProcedure {
	return func(tx *sql.Tx) (err error) {
		log.WithFields(log.Fields{
			"profile_database_id": id,
		}).Debug("deleting profile")
		_, err = tx.Exec(`DELETE FROM "shardChain" WHERE "id"=?`, id)
		return
	}
}

func updateProvider(profile *types.ProviderProfile) storageProcedure {
	var (
		enc *bytes.Buffer
		err error
	)
	if enc, err = utils.EncodeMsgPack(profile); err != nil {
		return errPass(err)
	}
	return func(tx *sql.Tx) (err error) {
		log.WithFields(log.Fields{
			"provider_address":    profile.Provider.String(),
			"provider_token_type": profile.TokenType,
			"provider_node_id":    profile.NodeID,
		}).Debug("updating provider")
		_, err = tx.Exec(`INSERT OR REPLACE INTO "provider" ("address", "encoded") VALUES (?, ?)`,
			profile.Provider.String(),
			enc.Bytes())
		return
	}
}

func deleteProvider(address proto.AccountAddress) storageProcedure {
	return func(tx *sql.Tx) (err error) {
		log.WithFields(log.Fields{
			"provider_address": address.String(),
		}).Debug("deleting provider")
		_, err = tx.Exec(`DELETE FROM "provider" WHERE "address"=?`, address.String())
		return
	}
}

func loadIrreHash(st xi.Storage) (irre hash.Hash, err error) {
	var hex string
	// Load last irreversible block hash
	if err = st.Reader().QueryRow(
		`SELECT "hash" FROM "irreversible" WHERE "id"=0`,
	).Scan(&hex); err != nil {
		return
	}
	if err = hash.Decode(&irre, hex); err != nil {
		return
	}
	return
}

func loadTxPool(st xi.Storage) (txPool map[hash.Hash]pi.Transaction, err error) {
	var (
		th   hash.Hash
		rows *sql.Rows
		tt   uint32
		hex  string
		enc  []byte
		pool = make(map[hash.Hash]pi.Transaction)
	)

	if rows, err = st.Reader().Query(
		`SELECT "type", "hash", "encoded" FROM "txPool"`,
	); err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		if err = rows.Scan(&tt, &hex, &enc); err != nil {
			return
		}
		if err = hash.Decode(&th, hex); err != nil {
			return
		}
		var dec pi.Transaction
		if err = utils.DecodeMsgPack(enc, &dec); err != nil {
			return
		}
		pool[th] = dec
	}

	txPool = pool
	return
}

func loadBlocks(
	st xi.Storage, irreHash hash.Hash) (irre *blockNode, heads []*blockNode, err error,
) {
	var (
		rows *sql.Rows

		index      = make(map[hash.Hash]*blockNode)
		headsIndex = make(map[hash.Hash]*blockNode)

		// Scan buffer
		id           uint32
		height       uint32
		bnHex, pnHex string
		enc          []byte

		ok     bool
		bh, ph hash.Hash
		bn, pn *blockNode
	)

	// Load blocks
	if rows, err = st.Reader().Query(
		`SELECT "rowid", "height", "hash", "parent", "encoded" FROM "blocks" ORDER BY "rowid"`,
	); err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		// Scan and decode block
		if err = rows.Scan(&id, &height, &bnHex, &pnHex, &enc); err != nil {
			return
		}
		if err = hash.Decode(&bh, bnHex); err != nil {
			return
		}
		if err = hash.Decode(&ph, pnHex); err != nil {
			return
		}
		var dec = &types.BPBlock{}
		if err = utils.DecodeMsgPack(enc, dec); err != nil {
			return
		}
		log.WithFields(log.Fields{
			"rowid":  id,
			"height": height,
			"hash":   bh.Short(4),
			"parent": ph.Short(4),
		}).Debug("loaded new block")
		// Add genesis block
		if height == 0 {
			if len(index) != 0 {
				err = ErrMultipleGenesis
				return
			}
			bn = newBlockNode(0, dec, nil)
			index[bh] = bn
			headsIndex[bh] = bn
			log.WithFields(log.Fields{
				"rowid":  id,
				"height": height,
				"hash":   bh.Short(4),
				"parent": ph.Short(4),
			}).Debug("set genesis block")
			continue
		}
		// Add normal block
		if pn, ok = index[ph]; !ok {
			err = errors.Wrapf(ErrParentNotFound, "parent %s not found", ph.Short(4))
			return
		}
		bn = newBlockNode(height, dec, pn)
		index[bh] = bn
		if _, ok = headsIndex[ph]; ok {
			delete(headsIndex, ph)
		}
		headsIndex[bh] = bn
	}

	if irre, ok = index[irreHash]; !ok {
		err = errors.Wrapf(ErrParentNotFound, "irreversible block %s not found", ph.Short(4))
		return
	}

	for _, v := range headsIndex {
		heads = append(heads, v)
	}

	return
}

func loadAndCacheAccounts(st xi.Storage, view *metaState) (err error) {
	var (
		rows *sql.Rows
		hex  string
		addr hash.Hash
		enc  []byte
	)

	if rows, err = st.Reader().Query(`SELECT "address", "encoded" FROM "accounts"`); err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		if err = rows.Scan(&hex, &enc); err != nil {
			return
		}
		if err = hash.Decode(&addr, hex); err != nil {
			return
		}
		var dec = &types.Account{}
		if err = utils.DecodeMsgPack(enc, dec); err != nil {
			return
		}
		view.readonly.accounts[proto.AccountAddress(addr)] = dec
	}

	return
}

func loadAndCacheShardChainProfiles(st xi.Storage, view *metaState) (err error) {
	var (
		rows *sql.Rows
		id   string
		enc  []byte
	)

	if rows, err = st.Reader().Query(`SELECT "id", "encoded" FROM "shardChain"`); err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		if err = rows.Scan(&id, &enc); err != nil {
			return
		}
		var dec = &types.SQLChainProfile{}
		if err = utils.DecodeMsgPack(enc, dec); err != nil {
			return
		}
		view.readonly.databases[proto.DatabaseID(id)] = dec
	}

	return
}

func loadAndCacheProviders(st xi.Storage, view *metaState) (err error) {
	var (
		rows *sql.Rows
		hex  string
		addr hash.Hash
		enc  []byte
	)

	if rows, err = st.Reader().Query(`SELECT "address", "encoded" FROM "provider"`); err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		if err = rows.Scan(&hex, &enc); err != nil {
			return
		}
		if err = hash.Decode(&addr, hex); err != nil {
			return
		}
		var dec = &types.ProviderProfile{}
		if err = utils.DecodeMsgPack(enc, dec); err != nil {
			return
		}
		view.readonly.provider[proto.AccountAddress(addr)] = dec
	}

	return
}

func loadImmutableState(st xi.Storage) (immutable *metaState, err error) {
	immutable = newMetaState()
	if err = loadAndCacheAccounts(st, immutable); err != nil {
		return
	}
	if err = loadAndCacheShardChainProfiles(st, immutable); err != nil {
		return
	}
	if err = loadAndCacheProviders(st, immutable); err != nil {
		return
	}
	return
}

func loadDatabase(st xi.Storage) (
	irre *blockNode,
	heads []*blockNode,
	immutable *metaState,
	txPool map[hash.Hash]pi.Transaction,
	err error,
) {
	var irreHash hash.Hash
	// Load last irreversible block hash
	if irreHash, err = loadIrreHash(st); err != nil {
		return
	}
	// Load blocks
	if irre, heads, err = loadBlocks(st, irreHash); err != nil {
		return
	}
	// Load immutable state
	if immutable, err = loadImmutableState(st); err != nil {
		return
	}
	// Load tx pool
	if txPool, err = loadTxPool(st); err != nil {
		return
	}

	return
}
