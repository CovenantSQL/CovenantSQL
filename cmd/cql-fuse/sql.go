/*
 * Copyright 2018 The CovenantSQL Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Copyright 2015 The Cockroach Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License. See the AUTHORS file
// for names of contributors.
//
// Author: Marc Berhault (marc@cockroachlabs.com)

package main

import (
	"database/sql"
	"encoding/json"
	"syscall"

	"bazil.org/fuse"
)

// sqlExecutor is an interface needed for basic queries.
// It is implemented by both sql.DB and sql.Txn.
type sqlExecutor interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}

// getInode looks up an inode given its name and its parent ID.
// If not found, error will be sql.ErrNoRows.
func getInode(e sqlExecutor, parentID uint64, name string) (*Node, error) {
	var raw string
	const sql = `SELECT inode FROM fs_inode WHERE id = 
(SELECT id FROM fs_namespace WHERE (parentID, name) = (?, ?))`
	if err := e.QueryRow(sql, parentID, name).Scan(&raw); err != nil {
		return nil, err
	}

	node := &Node{}
	err := json.Unmarshal([]byte(raw), node)
	return node, err
}

// checkIsEmpty returns nil if 'id' has no children.
func checkIsEmpty(e sqlExecutor, id uint64) error {
	var count uint64
	const countSQL = `
SELECT COUNT(parentID) FROM fs_namespace WHERE parentID = ?`
	if err := e.QueryRow(countSQL, id).Scan(&count); err != nil {
		return err
	}
	if count != 0 {
		return fuse.Errno(syscall.ENOTEMPTY)
	}
	return nil
}

// updateNode updates an existing node descriptor.
func updateNode(e sqlExecutor, node *Node) error {
	inode := node.toJSON()
	const sql = `
UPDATE fs_inode SET inode = ? WHERE id = ?;
`
	if _, err := e.Exec(sql, inode, node.ID); err != nil {
		return err
	}
	return nil
}

// getBlockData returns the block data for a single block.
func getBlockData(e sqlExecutor, inodeID uint64, block int) ([]byte, error) {
	var data []byte
	const sql = `SELECT data FROM fs_block WHERE id = ? AND block = ?`
	if err := e.QueryRow(sql, inodeID, block).Scan(&data); err != nil {
		return nil, err
	}
	return data, nil
}

// updateBlockData overwrites the data for a single block.
func updateBlockData(e sqlExecutor, inodeID uint64, block int, data []byte) error {
	const sql = `UPDATE fs_block SET data = ? WHERE (id, block) = (?, ?)`
	if _, err := e.Exec(sql, data, inodeID, block); err != nil {
		return err
	}
	return nil
}

type blockInfo struct {
	block int
	data  []byte
}

// getBlocks fetches all the blocks for a given inode and returns
// a list of blockInfo objects.
func getBlocks(e sqlExecutor, inodeID uint64) ([]blockInfo, error) {
	stmt := `SELECT block, data FROM fs_block WHERE id = ?`
	rows, err := e.Query(stmt, inodeID)
	if err != nil {
		return nil, err
	}
	return buildBlockInfos(rows)
}

// getBlocksBetween fetches blocks with IDs [start, end] for a given inode
// and returns a list of blockInfo objects.
func getBlocksBetween(e sqlExecutor, inodeID uint64, start, end int) ([]blockInfo, error) {
	stmt := `SELECT block, data FROM fs_block WHERE id = ? AND block >= ? AND block <= ?`
	rows, err := e.Query(stmt, inodeID, start, end)
	if err != nil {
		return nil, err
	}
	return buildBlockInfos(rows)
}

func buildBlockInfos(rows *sql.Rows) ([]blockInfo, error) {
	var results []blockInfo
	for rows.Next() {
		b := blockInfo{}
		if err := rows.Scan(&b.block, &b.data); err != nil {
			return nil, err
		}
		results = append(results, b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}
