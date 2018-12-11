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

	pi "github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	xi "github.com/CovenantSQL/CovenantSQL/xenomint/interfaces"
	xs "github.com/CovenantSQL/CovenantSQL/xenomint/sqlite"
)

var (
	ddls = [...]string{
		// Chain state tables
		`CREATE TABLE IF NOT EXISTS "blocks" (
	"hash"		TEXT
	"parent"	TEXT
	"encoded"	BLOB
	UNIQUE INDEX ("hash")
)`,
		`CREATE TABLE IF NOT EXISTS "txPool" (
	"type"		INT
	"hash"		TEXT
	"encoded"	BLOB
	UNIQUE INDEX ("hash")
)`,
		`CREATE TABLE IF NOT EXISTS "irreversible" (
	"id"		INT
	"hash"		TEXT
)`,
		// Meta state tables
		`CREATE TABLE IF NOT EXISTS "accounts" (
	"address"	TEXT
	"encoded"	BLOB
	UNIQUE INDEX ("address")
)`,
		`CREATE TABLE IF NOT EXISTS "shardChain" (
	"address"	TEXT
	"id"		TEXT
	"encoded"	BLOB
	UNIQUE INDEX ("address", "id")
)`,
		`CREATE TABLE IF NOT EXISTS "provider" (
	"address"	TEXT
	"encoded"	BLOB
	UNIQUE INDEX ("address")
)`,
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
	// ROLLBACK on failure
	defer tx.Rollback()
	// WRITE
	for _, sp := range sps {
		if err = sp(tx); err != nil {
			return
		}
	}
	// CALLBACK: MUST NOT FAIL
	cb()
	// COMMIT
	if err = tx.Commit(); err != nil {
		log.WithError(err).Fatalf("Failed to commit storage transaction")
	}
	return
}

func errPass(err error) storageProcedure {
	return func(_ *sql.Tx) error {
		return err
	}
}

func openStorage(path string) (st xi.Storage, err error) {
	if st, err = xs.NewSqlite(path); err != nil {
		return
	}
	for _, v := range ddls {
		if _, err = st.Writer().Exec(v); err != nil {
			return
		}
	}
	return
}

func addBlock(b *types.BPBlock) storageProcedure {
	var (
		enc *bytes.Buffer
		err error
	)
	if enc, err = utils.EncodeMsgPack(b); err != nil {
		return errPass(err)
	}
	return func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(`INSERT OR REPLACE INTO "blocks" VALUES (?, ?, ?)`,
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
		_, err = tx.Exec(`INSERT OR REPLACE INTO "txPool" VALUES (?, ?, ?)`,
			uint32(t.GetTransactionType()),
			t.Hash().String(),
			enc.Bytes())
		return
	}
}

func updateImmutable(tx []pi.Transaction) storageProcedure {
	return nil
}

func updateIrreversible(h hash.Hash) storageProcedure {
	return func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(`INSERT OR REPLACE INTO "irreversible" VALUES (?, ?)`, 0, h.String())
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
		_, err = tx.Exec(`INSERT OR REPLACE INTO "accounts" VALUES (?, ?)`,
			account.Address.String(),
			enc.Bytes())
		return
	}
}

func deleteAccount(address proto.AccountAddress) storageProcedure {
	return func(tx *sql.Tx) (err error) {
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
		_, err = tx.Exec(`INSERT OR REPLACE INTO "shardChain" VALUES (?, ?, ?)`,
			profile.Address.String(),
			string(profile.ID),
			enc.Bytes())
		return
	}
}

func deleteShardChain(id proto.DatabaseID) storageProcedure {
	return func(tx *sql.Tx) (err error) {
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
		_, err = tx.Exec(`INSERT OR REPLACE INTO "provider" VALUES (?, ?)`,
			profile.Provider.String(),
			enc.Bytes())
		return
	}
}

func deleteProvider(address proto.AccountAddress) storageProcedure {
	return func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(`DELETE FROM "provider" WHERE "address"=?`, address.String())
		return
	}
}
