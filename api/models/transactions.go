package models

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/go-gorp/gorp"
)

// TransactionsModel groups operations on Transactions.
type TransactionsModel struct{}

// Transaction is a transaction.
type Transaction struct {
	BlockHeight    int64       `db:"block_height" json:"block_height"` // pk1
	TxIndex        int         `db:"tx_index" json:"index"`            // pk2
	Hash           string      `db:"hash" json:"hash"`
	BlockHash      string      `db:"block_hash" json:"block_hash"`
	Timestamp      int64       `db:"timestamp" json:"timestamp"`
	TimestampHuman time.Time   `db:"-" json:"timestamp_human"`
	TxType         int         `db:"tx_type" json:"type"`
	Address        string      `db:"address" json:"address"`
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
	address, raw
	FROM indexed_transactions WHERE hash = ?`
	err = chaindb.SelectOne(tx, query, hash)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return tx, err
}

// GetTransactionListOfBlock get a transaction list of block.
func (m *TransactionsModel) GetTransactionListOfBlock(ofBlockHeight int, page, size int) (
	txs []*Transaction, pagination *Pagination, err error,
) {
	var (
		querySQL = `
		SELECT
			block_height,
			tx_index,
			hash,
			block_hash,
			timestamp,
			tx_type,
			address,
			raw
		FROM
			indexed_transactions
		`
		countSQL = buildCountSQL(querySQL)
		conds    []string
		args     []interface{}
	)

	pagination = NewPagination(page, size)
	conds = append(conds, "block_height = ?")
	args = append(args, ofBlockHeight)

	querySQL, countSQL = buildSQLWithConds(querySQL, countSQL, conds)
	count, err := chaindb.SelectInt(countSQL, args...)
	if err != nil {
		return nil, pagination, err
	}
	pagination.SetTotal(int(count))
	if pagination.Offset() > pagination.Total {
		return txs, pagination, nil
	}

	querySQL += " ORDER BY tx_index DESC"
	querySQL += " LIMIT ? OFFSET ?"
	args = append(args, pagination.Limit(), pagination.Offset())

	_, err = chaindb.Select(&txs, querySQL, args...)
	return txs, pagination, err
}

// GetTransactionList get a transaction list by hash marker.
func (m *TransactionsModel) GetTransactionList(since string, page, size int) (
	txs []*Transaction, pagination *Pagination, err error,
) {
	var (
		sinceBlockHeight int64
		sinceTxIndex     = 0
	)

	if since != "" {
		tx, err := m.GetTransactionByHash(since)
		if tx == nil {
			return txs, pagination, err
		}
		sinceBlockHeight = tx.BlockHeight
		sinceTxIndex = tx.TxIndex
	}

	var (
		querySQL = `
		SELECT
			block_height,
			tx_index,
			hash,
			block_hash,
			timestamp,
			tx_type,
			address,
			raw
		FROM
			indexed_transactions
		`
		countSQL = buildCountSQL(querySQL)
		conds    []string
		args     []interface{}
	)

	pagination = NewPagination(page, size)
	if sinceBlockHeight > 0 {
		conds = append(conds, "(block_height < ? or (block_height = ? and tx_index < ?))")
		args = append(args, sinceBlockHeight, sinceBlockHeight, sinceTxIndex)
	}

	querySQL, countSQL = buildSQLWithConds(querySQL, countSQL, conds)
	count, err := chaindb.SelectInt(countSQL, args...)
	if err != nil {
		return nil, pagination, err
	}
	pagination.SetTotal(int(count))
	if pagination.Offset() > pagination.Total {
		return txs, pagination, nil
	}

	querySQL += " ORDER BY block_height DESC, tx_index DESC"
	querySQL += " LIMIT ? OFFSET ?"
	args = append(args, pagination.Limit(), pagination.Offset())

	_, err = chaindb.Select(&txs, querySQL, args...)
	return txs, pagination, err
}
