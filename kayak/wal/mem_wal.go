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
	"io"
	"sync"
	"sync/atomic"

	kt "github.com/CovenantSQL/CovenantSQL/kayak/types"
)

// MemWal defines a toy wal using memory as storage.
type MemWal struct {
	sync.RWMutex
	logs     []*kt.Log
	revIndex map[uint64]int
	offset   uint64
	closed   uint32
}

// NewMemWal returns new memory wal instance.
func NewMemWal() (p *MemWal) {
	p = &MemWal{
		revIndex: make(map[uint64]int, 100000),
		logs:     make([]*kt.Log, 0, 100000),
	}

	return
}

// Write implements Wal.Write.
func (p *MemWal) Write(l *kt.Log) (err error) {
	if atomic.LoadUint32(&p.closed) == 1 {
		err = ErrWalClosed
		return
	}

	if l == nil {
		err = ErrInvalidLog
		return
	}

	p.Lock()
	defer p.Unlock()

	if _, exists := p.revIndex[l.Index]; exists {
		err = ErrAlreadyExists
		return
	}

	offset := atomic.AddUint64(&p.offset, 1) - 1
	p.logs = append(p.logs, nil)
	copy(p.logs[offset+1:], p.logs[offset:])
	p.logs[offset] = l
	p.revIndex[l.Index] = int(offset)

	return
}

// Read implements Wal.Read.
func (p *MemWal) Read() (l *kt.Log, err error) {
	if atomic.LoadUint32(&p.closed) == 1 {
		err = ErrWalClosed
		return
	}

	err = io.EOF
	return
}

// Get implements Wal.Get.
func (p *MemWal) Get(index uint64) (l *kt.Log, err error) {
	if atomic.LoadUint32(&p.closed) == 1 {
		err = ErrWalClosed
		return
	}

	p.RLock()
	defer p.RUnlock()

	var i int
	var exists bool
	if i, exists = p.revIndex[index]; !exists {
		err = ErrNotExists
		return
	}

	l = p.logs[i]

	return
}

// Close implements Wal.Close.
func (p *MemWal) Close() {
	if !atomic.CompareAndSwapUint32(&p.closed, 0, 1) {
		return
	}
}
