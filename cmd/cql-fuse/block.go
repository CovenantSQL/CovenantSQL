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
	"fmt"
	"strings"
)

// BlockSize is the size of each data block. It must not
// change throughout the lifetime of the filesystem.
const BlockSize = 128 << 10 // 128KB

func min(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}

// blockRange describes a range of blocks.
// If the first and last block are the same, the effective data range
// will be: [startOffset, lastLength).
type blockRange struct {
	start       int    // index of the start block
	startOffset uint64 // starting offset within the first block
	startLength uint64 // length of data in first block
	last        int    // index of the last block
	lastLength  uint64 // length of the last block
}

// newBlockRange returns the block range for 'size' bytes from 'from'.
func newBlockRange(from, length uint64) blockRange {
	end := from + length
	return blockRange{
		start:       int(from / BlockSize),
		startOffset: from % BlockSize,
		startLength: min(length, BlockSize-(from%BlockSize)),
		last:        int(end / BlockSize),
		lastLength:  end % BlockSize,
	}
}

// shrink resizes the data to a smaller length.
// Requirement: from > to.
// If truncates are done on block boundaries, this is reasonably
// efficient. However, if truncating in the middle of a block,
// we need to fetch the block first, truncate it, and write it again.
func shrink(e sqlExecutor, inodeID, from, to uint64) error {
	delRange := newBlockRange(to, from-to)
	deleteFrom := delRange.start

	if delRange.startOffset > 0 {
		// We're truncating in the middle of a block, fetch it, truncate its
		// data, and write it again.
		// TODO(marc): this would be more efficient if we had LEFT for bytes.
		data, err := getBlockData(e, inodeID, delRange.start)
		if err != nil {
			return err
		}
		data = data[:delRange.startOffset]
		if err := updateBlockData(e, inodeID, delRange.start, data); err != nil {
			return err
		}
		// We don't need to delete this block.
		deleteFrom++
	}

	deleteTo := delRange.last
	if delRange.lastLength == 0 {
		// The last block did not previously exist.
		deleteTo--
	}
	if deleteTo < deleteFrom {
		return nil
	}

	// There is something to delete.
	// TODO(marc): would it be better to pass the block IDs?
	delStmt := `DELETE FROM fs_BLOCK WHERE id = ? AND block >= ?`
	if _, err := e.Exec(delStmt, inodeID, deleteFrom); err != nil {
		return err
	}

	return nil
}

// grow resizes the data to a larger length.
// Requirement: to > from.
// If the file ended in a partial block, we fetch it, grow it,
// and write it back.
func grow(e sqlExecutor, inodeID, from, to uint64) error {
	addRange := newBlockRange(from, to-from)
	insertFrom := addRange.start

	if addRange.startOffset > 0 {
		// We need to extend the original 'last block'.
		// Fetch it, grow it, and update it.
		// TODO(marc): this would be more efficient if we had RPAD for bytes.
		data, err := getBlockData(e, inodeID, addRange.start)
		if err != nil {
			return err
		}
		data = append(data, make([]byte, addRange.startLength, addRange.startLength)...)
		if err := updateBlockData(e, inodeID, addRange.start, data); err != nil {
			return err
		}
		// We don't need to insert this block.
		insertFrom++
	}

	insertTo := addRange.last
	if insertTo < insertFrom {
		return nil
	}

	// Build the sql statement and blocks to insert.
	// We don't share this functionality with 'write' because we can repeat empty blocks.
	// This would be shorter if we weren't trying to be efficient.
	// TODO(marc): this would also be better if we supported sparse files.
	paramStrings := []string{}
	params := []interface{}{}
	count := 1 // placeholder count starts at 1.
	if insertFrom != insertTo {
		// We have full blocks. Only send a full block once.
		for i := insertFrom; i < insertTo; i++ {
			params = append(params, make([]byte, BlockSize, BlockSize))
		}
		count++
	}

	// Go over all blocks that are certainly full.
	for i := insertFrom; i < insertTo; i++ {
		paramStrings = append(paramStrings, fmt.Sprintf("(%d, %d, ?)", inodeID, i))
	}

	// Check the last block.
	if addRange.lastLength > 0 {
		// Not empty, write it. It can't be a full block, because we
		// would have an empty block right after.
		params = append(params, make([]byte, addRange.lastLength, addRange.lastLength))
		paramStrings = append(paramStrings, fmt.Sprintf("(%d, %d, ?)",
			inodeID, addRange.last))
		count++
	}

	if len(paramStrings) == 0 {
		// We had only one block, and it was empty. Nothing do to.
		return nil
	}

	insStmt := fmt.Sprintf(`INSERT INTO fs_block VALUES %s`, strings.Join(paramStrings, ","))
	if _, err := e.Exec(insStmt, params...); err != nil {
		return err
	}

	return nil
}

// read returns the data [from, to).
// Requires: to > from and [to, from) is contained in the file.
func read(e sqlExecutor, inodeID, from, to uint64) ([]byte, error) {
	readRange := newBlockRange(from, to-from)
	end := readRange.last
	if readRange.lastLength == 0 {
		end--
	}

	blockInfos, err := getBlocksBetween(e, inodeID, readRange.start, end)
	if err != nil {
		return nil, err
	}
	if len(blockInfos) != end-readRange.start+1 {
		return nil, fmt.Errorf("wrong number of blocks, asked for [%d-%d], got %d back",
			readRange.start, end, len(blockInfos))
	}

	if readRange.lastLength != 0 {
		// We have a last partial block, truncate it.
		last := len(blockInfos) - 1
		blockInfos[last].data = blockInfos[last].data[:readRange.lastLength]
	}
	blockInfos[0].data = blockInfos[0].data[readRange.startOffset:]

	var data []byte
	for _, b := range blockInfos {
		data = append(data, b.data...)
	}

	return data, nil
}

// write commits data to the blocks starting at 'offset'
// Amount of data to write must be non-zero.
// If offset is greated than 'originalSize', the file is grown first.
// We always write all or nothing.
func write(e sqlExecutor, inodeID, originalSize, offset uint64, data []byte) error {
	if offset > originalSize {
		diff := offset - originalSize
		if diff > BlockSize*2 {
			// we need to grow the file by at least two blocks. Use growing method
			// which only sends empty blocks once.
			if err := grow(e, inodeID, originalSize, offset); err != nil {
				return err
			}
			originalSize = offset
		} else if diff > 0 {
			// don't grow the file first, just change what we need to write.
			data = append(make([]byte, diff, diff), data...)
			offset = originalSize
		}
	}

	// Now we know that offset is <= originalSize.
	writeRange := newBlockRange(offset, uint64(len(data)))
	writeFrom := writeRange.start

	if writeRange.startOffset > 0 {
		// We're partially overwriting a block (this includes appending
		// to the last block): fetch it, grow it, and update it.
		// TODO(marc): this would be more efficient if we had RPAD for bytes.
		blockData, err := getBlockData(e, inodeID, writeRange.start)
		if err != nil {
			return err
		}
		blockData = append(blockData[:writeRange.startOffset], data[:writeRange.startLength]...)
		data = data[writeRange.startLength:]
		if err := updateBlockData(e, inodeID, writeRange.start, blockData); err != nil {
			return err
		}
		// We don't need to insert this block.
		writeFrom++
	}

	writeTo := writeRange.last
	if writeRange.lastLength == 0 {
		// Last block is empty, don't update/insert it.
		writeTo--
	}
	if writeTo < writeFrom {
		return nil
	}

	// Figure out last existing block. Needed to tell the difference
	// between insert and update.
	lastBlock := int(originalSize / BlockSize)
	if originalSize%BlockSize == 0 {
		// Empty blocks do not exist (size=0 -> lastblock=-1).
		lastBlock--
	}

	// Process updates first.
	for i := writeFrom; i <= writeTo; i++ {
		if i > lastBlock {
			// We've reached the end of existing blocks, no more UPDATE.
			break
		}
		if len(data) == 0 {
			panic(fmt.Sprintf("reached end of data, but still have %d blocks to write",
				writeTo-i))
		}
		toWrite := min(BlockSize, uint64(len(data)))
		blockData := data[:toWrite]
		data = data[toWrite:]
		if toWrite != BlockSize {
			// This is the last block, and it's partial, fetch the original
			// data from this block and append.
			// TODO(marc): we could fetch this at the same time as the first
			// partial block, if any. This would make overwriting in the middle
			// of the file on non-block boundaries a bit more efficient.
			origData, err := getBlockData(e, inodeID, i)
			if err != nil {
				return err
			}
			toWrite = min(toWrite, uint64(len(origData)))
			blockData = append(blockData, origData[toWrite:]...)
		}
		// TODO(marc): is there a way to do batch updates?
		if err := updateBlockData(e, inodeID, i, blockData); err != nil {
			return err
		}
	}

	if len(data) == 0 {
		return nil
	}

	paramStrings := []string{}
	params := []interface{}{}
	count := 1 // placeholder count starts at 1.

	for i := lastBlock + 1; i <= writeTo; i++ {
		if len(data) == 0 {
			panic(fmt.Sprintf("reached end of data, but still have %d blocks to write",
				writeTo-i))
		}
		toWrite := min(BlockSize, uint64(len(data)))
		blockData := data[:toWrite]
		data = data[toWrite:]
		paramStrings = append(paramStrings, fmt.Sprintf("(%d, %d, ?)",
			inodeID, i))
		params = append(params, blockData)
		count++
	}

	if len(data) != 0 {
		panic(fmt.Sprintf("processed all blocks, but still have %d of data to write", len(data)))
	}

	insStmt := fmt.Sprintf(`INSERT INTO fs_block VALUES %s`, strings.Join(paramStrings, ","))
	if _, err := e.Exec(insStmt, params...); err != nil {
		return err
	}

	return nil
}

// resize changes the size of the data for the inode with id 'inodeID'
// from 'from' to 'to'. This may grow or shrink.
func resizeBlocks(e sqlExecutor, inodeID, from, to uint64) error {
	if to < from {
		return shrink(e, inodeID, from, to)
	} else if to > from {
		return grow(e, inodeID, from, to)
	}
	return nil
}
