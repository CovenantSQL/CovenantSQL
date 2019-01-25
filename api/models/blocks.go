package models

import (
	"database/sql"
	"time"

	"github.com/go-gorp/gorp"
)

// BlocksModel groups operations on Blocks.
type BlocksModel struct{}

// Block is a block.
type Block struct {
	Height         int64     `db:"height" json:"height"` // pk
	Hash           string    `db:"hash" json:"hash"`
	Timestamp      int64     `db:"timestamp" json:"timestamp"`
	TimestampHuman time.Time `db:"-" json:"timestamp_human"`
	Version        int32     `db:"version"  json:"version"`
	Producer       string    `db:"producer" json:"producer"`
	MerkleRoot     string    `db:"merkle_root" json:"merkle_root"`
	Parent         string    `db:"parent" json:"parent"`
	TxCount        int64     `db:"tx_count" json:"tx_count"`
}

// PostGet is the hook after SELECT query.
func (b *Block) PostGet(s gorp.SqlExecutor) error {
	b.TimestampHuman = time.Unix(0, b.Timestamp)
	return nil
}

// GetMaxHeight returns the max height of the confirmed blocks.
func (m *BlocksModel) GetMaxHeight() (height int64, err error) {
	querySQL := `SELECT MAX("height") FROM "indexed_blocks";`
	return chaindb.SelectInt(querySQL)
}

// GetBlockList get a list of blocks with height in [from, to).
func (m *BlocksModel) GetBlockList(since, page, size int) (blocks []*Block, pagination *Pagination, err error) {
	var (
		querySQL = `
		SELECT
			height,
			hash,
			timestamp,
			version,
			producer,
			merkle_root,
			parent,
			tx_count
		FROM
			indexed_blocks
		`
		countSQL = buildCountSQL(querySQL)
		conds    []string
		args     []interface{}
	)

	pagination = NewPagination(page, size)
	if since > 0 {
		conds = append(conds, "height < ?")
		args = append(args, since)
	}

	querySQL, countSQL = buildSQLWithConds(querySQL, countSQL, conds)

	count, err := chaindb.SelectInt(countSQL, args...)
	if err != nil {
		return nil, pagination, err
	}
	pagination.SetTotal(int(count))
	blocks = make([]*Block, 0)
	if pagination.Offset() > pagination.Total {
		return blocks, pagination, nil
	}

	querySQL += " ORDER BY height DESC"
	querySQL += " LIMIT ? OFFSET ?"
	args = append(args, pagination.Limit(), pagination.Offset())

	_, err = chaindb.Select(&blocks, querySQL, args...)
	return blocks, pagination, err
}

// GetBlockByHeight get a block by its height.
func (m *BlocksModel) GetBlockByHeight(height int) (block *Block, err error) {
	block = &Block{}
	query := `SELECT height, hash, timestamp, version, producer, merkle_root, parent, tx_count
	FROM indexed_blocks WHERE height = ?`
	err = chaindb.SelectOne(block, query, height)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return block, err
}

// GetBlockByHash get a block by its hash.
func (m *BlocksModel) GetBlockByHash(hash string) (block *Block, err error) {
	block = &Block{}
	query := `SELECT height, hash, timestamp, version, producer, merkle_root, parent, tx_count
	FROM indexed_blocks WHERE hash = ?`
	err = chaindb.SelectOne(block, query, hash)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return block, err
}
