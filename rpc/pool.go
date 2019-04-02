/*
 * Copyright 2018-2019 The CovenantSQL Authors.
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

package rpc

import (
	"net"
	"strings"
	"sync"

	"github.com/pkg/errors"

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/proto"
)

type pooledConn struct {
	net.Conn
	freelist *freelist
}

func (c *pooledConn) Close() error {
	return c.freelist.put(c.Conn) // unwrap
}

// freelist is the freelist type of ConnPool.
type freelist struct {
	sync.RWMutex
	target proto.NodeID
	freeCh chan net.Conn
}

// Close closes the freelist.
func (l *freelist) Close() error {
	l.Lock()
	defer l.Unlock()
	close(l.freeCh)
	var errmsgs []string
	for s := range l.freeCh {
		if err := s.Close(); err != nil {
			errmsgs = append(errmsgs, err.Error())
		}
	}
	l.freeCh = nil // set to nil explicitly to force close in freelist.put
	if len(errmsgs) > 0 {
		return errors.Wrapf(errors.New(strings.Join(errmsgs, ", ")), "close free list %s", l.target)
	}
	return nil
}

func (l *freelist) put(conn net.Conn) error {
	l.Lock()
	defer l.Unlock()
	select {
	case l.freeCh <- conn:
	default:
		return conn.Close()
	}
	return nil
}

func (l *freelist) get() (conn net.Conn, ok bool) {
	l.RLock()
	defer l.RUnlock()
	conn, ok = <-l.freeCh
	return
}

// Get returns new connection from freelist.
func (l *freelist) Get() (conn net.Conn, err error) {
	var (
		raw net.Conn
		ok  bool
	)
	if raw, ok = l.get(); !ok {
		if raw, err = l.newConn(); err != nil {
			return
		}
	}
	return &pooledConn{
		Conn:     raw, // wrap
		freelist: l,
	}, nil
}

// Len returns physical connection count.
func (l *freelist) Len() int {
	l.RLock()
	defer l.RUnlock()
	return len(l.freeCh)
}

func (l *freelist) newConn() (conn net.Conn, err error) {
	conn, err = Dial(l.target)
	if err != nil {
		err = errors.Wrap(err, "dialing new connection failed")
		return
	}
	return
}

// ConnPool is the struct type of connection pool.
type ConnPool struct {
	nodeFreeLists sync.Map // proto.NodeID -> freelist
}

func (p *ConnPool) loadFreeList(id proto.NodeID) (sess *freelist, ok bool) {
	var v interface{}
	if v, ok = p.nodeFreeLists.Load(id); ok {
		sess = v.(*freelist)
		return
	}
	v, ok = p.nodeFreeLists.LoadOrStore(id, &freelist{
		target: id,
		freeCh: make(chan net.Conn, conf.MaxRPCPoolPhysicalConnection),
	})
	sess = v.(*freelist)
	return
}

// Get returns existing freelist to the node, if not exist try best to create one.
func (p *ConnPool) Get(id proto.NodeID) (conn net.Conn, err error) {
	sess, _ := p.loadFreeList(id)
	return sess.Get()
}

// GetEx returns an one-off connection if it's anonymous, otherwise returns existing freelist
// with Get.
func (p *ConnPool) GetEx(id proto.NodeID, isAnonymous bool) (conn net.Conn, err error) {
	if isAnonymous {
		return DialEx(id, true)
	}
	return p.Get(id)
}

// Remove the node freelist in the pool.
func (p *ConnPool) Remove(id proto.NodeID) {
	v, ok := p.nodeFreeLists.Load(id)
	if ok {
		_ = v.(*freelist).Close()
		p.nodeFreeLists.Delete(id)
	}
	return
}

// Close closes all FreeLists in the pool.
func (p *ConnPool) Close() error {
	var errmsgs []string
	p.nodeFreeLists.Range(func(k, v interface{}) bool {
		if err := v.(*freelist).Close(); err != nil {
			errmsgs = append(errmsgs, err.Error())
		}
		return true
	})
	if len(errmsgs) > 0 {
		return errors.Wrap(errors.New(strings.Join(errmsgs, ", ")), "close connection pool")
	}
	return nil
}

// Len returns the connection count in the pool.
func (p *ConnPool) Len() (total int) {
	p.nodeFreeLists.Range(func(k, v interface{}) bool {
		total += v.(*freelist).Len()
		return true
	})
	return
}

var (
	defaultPool = &ConnPool{}
)
