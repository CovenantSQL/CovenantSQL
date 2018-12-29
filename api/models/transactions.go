package models

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"
)

// TransactionsModel groups operations on Transactions.
type TransactionsModel struct{}

// Transaction is a transaction.
type Transaction struct {
	BlockHeight    int         `db:"block_height" json:"block_height"` // pk1
	TxIndex        int         `db:"tx_index" json:"index"`            // pk2
	Hash           string      `db:"hash" json:"hash"`
	BlockHash      string      `db:"block_hash" json:"block_hash"`
	Timestamp      int64       `db:"timestamp" json:"-"`
	TimestampHuman time.Time   `db:"-" json:"timestamp"`
	TxType         int         `db:"tx_type" json:"type"`
	Signee         string      `db:"signee" json:"signee"`
	Address        string      `db:"address" json:"address"`
	Signature      string      `db:"signature" json:"signature"`
	Raw            string      `db:"raw" json:"raw"`
	Tx             interface{} `db:"-" json:"tx"`
}

// PostGet is the hook after SELECT query.
func (tx *Transaction) PostGet(s gorp.SqlExecutor) error {
	tx.TimestampHuman = time.Unix(0, tx.Timestamp)
	return json.Unmarshal([]byte(tx.Raw), &tx.Tx)
}

// GetTransactionByHash get a transaction by its hash.
func (m *TransactionsModel) GetTransactionByHash(hash string) (tx *Transaction, err error) {
	tx = &Transaction{}
	query := `SELECT block_height, tx_index, hash, block_hash, timestamp, tx_type,
	signee, address, signature, raw
	FROM indexed_transactions WHERE hash = ?`
	err = chaindb.SelectOne(tx, query, hash)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return tx, err
}

// GetTransactionList get a transaction list by hash marker.
func (m *TransactionsModel) GetTransactionList(since, direction string, limit int) (
	txs []*Transaction, err error,
) {
	tx, err := m.GetTransactionByHash(since)
	if tx == nil {
		return txs, err
	}

	orderBy := "DESC"
	compare := "<"
	if direction == "forward" {
		orderBy = "ASC"
		compare = ">"
	}

	query := fmt.Sprintf(`SELECT block_height, tx_index, hash, block_hash,
	timestamp, tx_type, signee, address, signature, raw
	FROM indexed_transactions
	WHERE block_height %s ? and tx_index %s ?
	ORDER BY block_height %s, tx_index %s
	LIMIT ?`, compare, compare, orderBy, orderBy)

	_, err = chaindb.Select(&txs, query, tx.BlockHeight, tx.TxIndex, limit)
	return txs, err
}
