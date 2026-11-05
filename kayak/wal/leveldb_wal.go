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
	"fmt"
	"io"
	"sync"
	"sync/atomic"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/util"

	kt "github.com/CovenantSQL/CovenantSQL/kayak/types"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

var (
	// logHeaderKeyPrefix defines the leveldb header key prefix.
	logHeaderKeyPrefix = []byte{'L', 'H'}
	// logDataKeyPrefix defines the leveldb data key prefix.
	logDataKeyPrefix = []byte{'L', 'D'}
)

// LevelDBWal defines a toy wal using leveldb as storage.
type LevelDBWal struct {
	db       *leveldb.DB
	it       iterator.Iterator
	closed   uint32
	readLock sync.Mutex
	read     uint32
}

// NewLevelDBWal returns new leveldb wal instance.
func NewLevelDBWal(filename string) (p *LevelDBWal, err error) {
	p = &LevelDBWal{}
	if p.db, err = leveldb.OpenFile(filename, nil); err != nil {
		err = fmt.Errorf("open database failed: %w", err)
		return
	}

	return
}

// Write implements Wal.Write.
func (p *LevelDBWal) Write(l *kt.Log) (err error) {
	if atomic.LoadUint32(&p.closed) == 1 {
		err = ErrWalClosed
		return
	}

	// mark wal as already read
	atomic.CompareAndSwapUint32(&p.read, 0, 1)

	if l == nil {
		err = ErrInvalidLog
		return
	}

	// build header headerKey
	headerKey := append(append([]byte(nil), logHeaderKeyPrefix...), p.uint64ToBytes(l.Index)...)

	if _, err = p.db.Get(headerKey, nil); err != nil && err != leveldb.ErrNotFound {
		err = fmt.Errorf("access leveldb failed: %w", err)
		return
	} else if err == nil {
		err = ErrAlreadyExists
		return
	}

	dataKey := append(append([]byte(nil), logDataKeyPrefix...), p.uint64ToBytes(l.Index)...)

	// write data first
	var enc *bytes.Buffer
	if enc, err = utils.EncodeMsgPack(l.Data); err != nil {
		err = fmt.Errorf("encode log data failed: %w", err)
		return
	}

	if err = p.db.Put(dataKey, enc.Bytes(), nil); err != nil {
		err = fmt.Errorf("write log data failed: %w", err)
		return
	}

	// write header
	l.DataLength = uint64(enc.Len())

	if enc, err = utils.EncodeMsgPack(l.LogHeader); err != nil {
		err = fmt.Errorf("encode log header failed: %w", err)
		return
	}

	// save header
	if err = p.db.Put(headerKey, enc.Bytes(), nil); err != nil {
		err = fmt.Errorf("encode log header failed: %w", err)
		return
	}

	return
}

// Read implements Wal.Read.
func (p *LevelDBWal) Read() (l *kt.Log, err error) {
	if atomic.LoadUint32(&p.closed) == 1 {
		err = ErrWalClosed
		return
	}

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
		return
	} else if err != nil {
		err = fmt.Errorf("get log header failed: %w", err)
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

func (p *LevelDBWal) load(logHeader []byte) (l *kt.Log, err error) {
	l = new(kt.Log)

	if err = utils.DecodeMsgPack(logHeader, &l.LogHeader); err != nil {
		log.WithField("header", logHeader).WithError(err).Debug("decode log header failed")
		err = fmt.Errorf("decode log header failed: %w", err)
		return
	}

	dataKey := append(append([]byte(nil), logDataKeyPrefix...), p.uint64ToBytes(l.Index)...)

	var encData []byte
	if encData, err = p.db.Get(dataKey, nil); err != nil {
		err = fmt.Errorf("get log data failed: %w", err)
		return
	}

	// load data
	if err = utils.DecodeMsgPack(encData, &l.Data); err != nil {
		err = fmt.Errorf("decode log data failed: %w", err)
	}

	return
}

func (p *LevelDBWal) uint64ToBytes(o uint64) (res []byte) {
	res = make([]byte, 8)
	binary.BigEndian.PutUint64(res, o)
	return
}
