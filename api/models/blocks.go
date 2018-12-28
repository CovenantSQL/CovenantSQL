package models

import (
	"database/sql"
	"time"
)

// BlocksModel groups operations on Blocks.
type BlocksModel struct{}

// Block is a block.
type Block struct {
	Height         int       `db:"height" json:"height"` // pk
	Hash           string    `db:"hash" json:"hash"`
	Timestamp      int64     `db:"timestamp" json:"-"`
	TimestampHuman time.Time `db:"-" json:"timestamp"`
	Version        int32     `db:"version"  json:"version"`
	Producer       string    `db:"producer" json:"producer"`
	MerkleRoot     string    `db:"merkle_root" json:"merkle_root"`
	Parent         string    `db:"parent" json:"parent"`
	TxCount        int       `db:"tx_count" json:"tx_count"`
}

// GetBlockList get a list of blocks with height in [from, to).
func (m *BlocksModel) GetBlockList(from, to int) (blocks []*Block, err error) {
	query := `SELECT height, hash, timestamp, version, producer, merkle_root, parent, tx_count
	FROM indexed_blocks WHERE height >= ? and height < ?`
	_, err = chaindb.Select(&blocks, query, from, to)
	return blocks, err
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
