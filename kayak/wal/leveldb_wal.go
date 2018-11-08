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

package wal

import (
	"bytes"
	"encoding/binary"
	"io"
	"sort"
	"sync"
	"sync/atomic"

	kt "github.com/CovenantSQL/CovenantSQL/kayak/types"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/util"
)

var (
	// logHeaderKeyPrefix defines the leveldb header key prefix.
	logHeaderKeyPrefix = []byte{'L', 'H'}
	// logDataKeyPrefix defines the leveldb data key prefix.
	logDataKeyPrefix = []byte{'L', 'D'}
	// baseIndexKey defines the base index key.
	baseIndexKey = []byte{'B', 'I'}
)

// LevelDBWal defines a toy wal using leveldb as storage.
type LevelDBWal struct {
	db          *leveldb.DB
	it          iterator.Iterator
	base        uint64
	closed      uint32
	readLock    sync.Mutex
	read        uint32
	pending     []uint64
	pendingLock sync.Mutex
}

// NewLevelDBWal returns new leveldb wal instance.
func NewLevelDBWal(filename string) (p *LevelDBWal, err error) {
	p = &LevelDBWal{}
	if p.db, err = leveldb.OpenFile(filename, nil); err != nil {
		err = errors.Wrap(err, "open database failed")
		return
	}

	// load current base
	var baseValue []byte
	if baseValue, err = p.db.Get(baseIndexKey, nil); err == nil {
		// decode base
		p.base = p.bytesToUint64(baseValue)
	} else {
		err = nil
	}

	return
}

// Write implements Wal.Write.
func (p *LevelDBWal) Write(l *kt.Log) (err error) {
	if atomic.LoadUint32(&p.closed) == 1 {
		err = ErrWalClosed
		return
	}

	if l == nil {
		err = ErrInvalidLog
		return
	}

	if l.Index < p.base {
		// already exists
		err = ErrAlreadyExists
		return
	}

	// build header headerKey
	headerKey := append(append([]byte(nil), logHeaderKeyPrefix...), p.uint64ToBytes(l.Index)...)

	if _, err = p.db.Get(headerKey, nil); err != nil && err != leveldb.ErrNotFound {
		err = errors.Wrap(err, "access leveldb failed")
		return
	} else if err == nil {
		err = ErrAlreadyExists
		return
	}

	dataKey := append(append([]byte(nil), logDataKeyPrefix...), p.uint64ToBytes(l.Index)...)

	// write data first
	var enc *bytes.Buffer
	if enc, err = utils.EncodeMsgPack(l.Data); err != nil {
		err = errors.Wrap(err, "encode log data failed")
		return
	}

	if err = p.db.Put(dataKey, enc.Bytes(), nil); err != nil {
		err = errors.Wrap(err, "write log data failed")
		return
	}

	// write header
	l.DataLength = uint64(enc.Len())

	if enc, err = utils.EncodeMsgPack(l.LogHeader); err != nil {
		err = errors.Wrap(err, "encode log header failed")
		return
	}

	// save header
	if err = p.db.Put(headerKey, enc.Bytes(), nil); err != nil {
		err = errors.Wrap(err, "encode log header failed")
		return
	}

	p.updatePending(l.Index)

	return
}

// Read implements Wal.Read.
func (p *LevelDBWal) Read() (l *kt.Log, err error) {
	if atomic.LoadUint32(&p.read) == 1 {
		err = io.EOF
		return
	}

	p.readLock.Lock()
	defer p.readLock.Unlock()

	// start with base, use iterator to read
	if p.it == nil {
		keyRange := util.BytesPrefix(logHeaderKeyPrefix)
		p.it = p.db.NewIterator(keyRange, nil)
	}

	if p.it.Next() {
		// load
		l, err = p.load(p.it.Value())
		// update base and pending
		if err == nil {
			p.updatePending(l.Index)
		}
		return
	}

	p.it.Release()
	if err = p.it.Error(); err == nil {
		err = io.EOF
	}
	p.it = nil

	// log read complete, could not read again
	atomic.StoreUint32(&p.read, 1)

	return
}

// Get implements Wal.Get.
func (p *LevelDBWal) Get(i uint64) (l *kt.Log, err error) {
	if atomic.LoadUint32(&p.closed) == 1 {
		err = ErrWalClosed
		return
	}

	headerKey := append(append([]byte(nil), logHeaderKeyPrefix...), p.uint64ToBytes(i)...)

	var headerData []byte
	if headerData, err = p.db.Get(headerKey, nil); err == leveldb.ErrNotFound {
		err = ErrNotExists
	} else if err != nil {
		err = errors.Wrap(err, "get log header failed")
		return
	}

	return p.load(headerData)
}

// Close implements Wal.Close.
func (p *LevelDBWal) Close() {
	if !atomic.CompareAndSwapUint32(&p.closed, 0, 1) {
		return
	}

	if p.it != nil {
		p.it.Release()
		p.it = nil
	}

	if p.db != nil {
		p.db.Close()
	}
}

func (p *LevelDBWal) updatePending(index uint64) {
	p.pendingLock.Lock()
	defer p.pendingLock.Unlock()

	if atomic.CompareAndSwapUint64(&p.base, index, index+1) {
		// process pending
		for len(p.pending) > 0 {
			if !atomic.CompareAndSwapUint64(&p.base, p.pending[0], p.pending[0]+1) {
				break
			}
			p.pending = p.pending[1:]
		}

		// commit base index to database
		_ = p.db.Put(baseIndexKey, p.uint64ToBytes(atomic.LoadUint64(&p.base)), nil)
	} else {
		i := sort.Search(len(p.pending), func(i int) bool {
			return p.pending[i] >= index
		})

		if len(p.pending) == i || p.pending[i] != index {
			p.pending = append(p.pending, 0)
			copy(p.pending[i+1:], p.pending[i:])
			p.pending[i] = index
		}
	}
}

func (p *LevelDBWal) load(logHeader []byte) (l *kt.Log, err error) {
	l = new(kt.Log)

	if err = utils.DecodeMsgPack(logHeader, &l.LogHeader); err != nil {
		err = errors.Wrap(err, "decode log header failed")
		return
	}

	dataKey := append(append([]byte(nil), logDataKeyPrefix...), p.uint64ToBytes(l.Index)...)

	var encData []byte
	if encData, err = p.db.Get(dataKey, nil); err != nil {
		err = errors.Wrap(err, "get log data failed")
		return
	}

	// load data
	if err = utils.DecodeMsgPack(encData, &l.Data); err != nil {
		err = errors.Wrap(err, "decode log data failed")
	}

	return
}

func (p *LevelDBWal) uint64ToBytes(o uint64) (res []byte) {
	res = make([]byte, 8)
	binary.BigEndian.PutUint64(res, o)
	return
}

func (p *LevelDBWal) bytesToUint64(b []byte) uint64 {
	return binary.BigEndian.Uint64(b)
}
