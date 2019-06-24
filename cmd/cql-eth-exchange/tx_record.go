/*
 * Copyright 2019 The CovenantSQL Authors.
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

package main

import (
	"encoding/json"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"
	gorp "gopkg.in/gorp.v2"

	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
)

type ExchangeState int16
type ExchangeRefundState int16

const (
	ExchangeStateDetected ExchangeState = iota
	ExchangeStateTransferring
	ExchangeStateTransferred
	ExchangeStateInvalidTxData
	ExchangeStateFailed
)

const (
	ExchangeRefundStateNotAvailable ExchangeRefundState = iota
	ExchangeRefundStateRefunding
	ExchangeRefundStateRefunded
)

func (s ExchangeState) String() string {
	switch s {
	case ExchangeStateDetected:
		return "Detected"
	case ExchangeStateTransferring:
		return "Transferring"
	case ExchangeStateTransferred:
		return "Transferred"
	case ExchangeStateInvalidTxData:
		return "InvalidTxData"
	case ExchangeStateFailed:
		return "Failed"
	default:
		return "Unknown"
	}
}

func (s ExchangeRefundState) String() string {
	switch s {
	case ExchangeRefundStateNotAvailable:
		return "NotAvailable"
	case ExchangeRefundStateRefunding:
		return "Refunding"
	case ExchangeRefundStateRefunded:
		return "Refunded"
	default:
		return "Unknown"
	}
}

type TxRecord struct {
	// eth stuff
	Hash           string             `db:"hash"`
	RawTx          []byte             `db:"tx"`
	Tx             *types.Transaction `db:"-"`
	ETHBlockNumber uint64             `db:"eth_block_number"`
	ETHFromAddr    string             `db:"eth_from_addr"`
	ETHToAddr      string             `db:"eth_to_addr"`
	ETHAmount      uint64             `db:"eth_amount"`

	// CQL stuff
	CQLAccount string `db:"cql_account"`
	CQLAmount  uint64 `db:"cql_amount"`
	CQLTxHash  string `db:"cql_tx_hash"`

	// state
	State       ExchangeState       `db:"exchange_state"`
	IsReverted  int8                `db:"is_reverted"`
	RefundState ExchangeRefundState `db:"exchange_refund_state"`

	Created    int64 `db:"created"`
	LastUpdate int64 `db:"last_update"`
}

func (r *TxRecord) PostGet(gorp.SqlExecutor) (err error) {
	return r.Deserialize()
}

func (r *TxRecord) PreUpdate(gorp.SqlExecutor) (err error) {
	r.LastUpdate = time.Now().Unix()
	return r.Serialize()
}

func (r *TxRecord) PreInsert(gorp.SqlExecutor) (err error) {
	r.Created = time.Now().Unix()
	return r.Serialize()
}

func (r *TxRecord) Serialize() (err error) {
	r.RawTx, err = json.Marshal(r.Tx)
	return
}

func (r *TxRecord) Deserialize() (err error) {
	err = json.Unmarshal(r.RawTx, &r.Tx)
	if err != nil {
		return
	}
	return
}

func UpsertTx(db *gorp.DbMap, r *TxRecord) (d *TxRecord, err error) {
	defer func() {
		var auditErr = err

		if d != nil && d.State != ExchangeStateDetected {
			auditErr = errors.New("transaction already processing")
		}

		_ = AddAuditRecord(db, &AuditRecord{
			Hash:  r.Hash,
			Op:    "upsert_tx",
			Data:  r,
			Error: auditErr.Error(),
		})
	}()

	err = db.SelectOne(&d, `SELECT * FROM "record" WHERE "hash" = ? LIMIT 1`, r.Hash)
	if err != nil {
		// not exists
		err = db.Insert(r)
		d = r
		return
	}

	if d.State != ExchangeStateDetected {
		return
	}

	d.Tx = r.Tx
	d.ETHBlockNumber = r.ETHBlockNumber
	d.ETHFromAddr = r.ETHFromAddr
	d.ETHToAddr = r.ETHToAddr
	d.ETHAmount = r.ETHAmount
	d.CQLAccount = r.CQLAccount
	d.CQLAmount = r.CQLAmount
	d.IsReverted = r.IsReverted

	return
}

func InvalidateTx(db *gorp.DbMap, blkNumber uint64) (err error) {
	var records []*TxRecord
	defer func() {
		for _, r := range records {
			_ = AddAuditRecord(db, &AuditRecord{
				Hash:  r.Hash,
				Op:    "chain_reorganize",
				Data:  blkNumber,
				Error: err.Error(),
			})
		}

		if len(records) == 0 {
			_ = AddAuditRecord(db, &AuditRecord{
				Op:    "chain_reorganize",
				Data:  blkNumber,
				Error: err.Error(),
			})
		}
	}()
	var rawTxs []interface{}
	rawTxs, err = db.Select(&records, `SELECT * FROM "record" WHERE "is_reverted" = 0 AND "eth_block_number" = ?`, blkNumber)
	if err != nil {
		return
	}

	for _, r := range records {
		r.IsReverted = 1
	}

	_, err = db.Update(rawTxs...)

	return
}

func GetToProcessTx(db *gorp.DbMap, confirmedBlkNumber uint64) (records []*TxRecord, err error) {
	_, err = db.Select(&records, `SELECT * FROM "record" WHERE "eth_block_number" <= ? AND "state" = ?`,
		confirmedBlkNumber, ExchangeStateDetected)
	return
}

func GetTransferringTx(db *gorp.DbMap) (records []*TxRecord, err error) {
	_, err = db.Select(&records, `SELECT * FROM "record" WHERE "state" = ?`, ExchangeStateTransferring)
	return
}

func SetTxToTransferring(db *gorp.DbMap, r *TxRecord, tx hash.Hash) (err error) {
	r.CQLTxHash = tx.String()
	r.State = ExchangeStateTransferring
	_, err = db.Update(r)
	return
}

func SetTxConfirmed(db *gorp.DbMap, r *TxRecord) (err error) {
	r.State = ExchangeStateTransferred
	_, err = db.Update(r)
	return
}
