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
	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	xi "github.com/CovenantSQL/CovenantSQL/xenomint/interfaces"
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

func initStorage(tx *sql.Tx) (err error) {
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
		_, err = tx.Exec(`?`, enc.Bytes())
		return
	}
}

func addTx(tx pi.Transaction) storageProcedure {
	var (
		enc *bytes.Buffer
		err error
	)
	if enc, err = utils.EncodeMsgPack(tx); err != nil {
		return errPass(err)
	}
	return func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(`INSERT INTO "txPool" VALUES (?, ?)`, enc.Bytes())
		return
	}
}

func updateImmutable(tx []pi.Transaction) storageProcedure {
	return nil
}

func updateIrreversible(h hash.Hash) storageProcedure {
	return func(tx *sql.Tx) (err error) {
		_, err = tx.Exec(`INSERT INTO "irreversible" VALUES (?)`, h)
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
			if _, err = stmt.Exec(v); err != nil {
				return
			}
		}
		return
	}
}
