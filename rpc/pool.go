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
	session *Session
}

func (c *pooledConn) Close() error {
	return c.session.put(c.Conn) // unwrap
}

// Session is the Session type of SessionPool.
type Session struct {
	sync.RWMutex
	target proto.NodeID
	sess   chan net.Conn
}

// Close closes the session.
func (s *Session) Close() error {
	s.Lock()
	defer s.Unlock()
	close(s.sess)
	var errmsgs []string
	for s := range s.sess {
		if err := s.Close(); err != nil {
			errmsgs = append(errmsgs, err.Error())
		}
	}
	s.sess = nil // set to nil explicitly to force close in Session.put
	if len(errmsgs) > 0 {
		return errors.Wrapf(errors.New(strings.Join(errmsgs, ", ")), "close session %s", s.target)
	}
	return nil
}

func (s *Session) put(conn net.Conn) error {
	s.Lock()
	defer s.Unlock()
	select {
	case s.sess <- conn:
	default:
		return conn.Close()
	}
	return nil
}

func (s *Session) get() (conn net.Conn, ok bool) {
	s.RLock()
	defer s.RUnlock()
	conn, ok = <-s.sess
	return
}

// Get returns new connection from session.
func (s *Session) Get() (conn net.Conn, err error) {
	var (
		raw net.Conn
		ok  bool
	)
	if raw, ok = s.get(); !ok {
		if raw, err = s.newConn(); err != nil {
			return
		}
	}
	return &pooledConn{
		Conn:    raw, // wrap
		session: s,
	}, nil
}

// Len returns physical connection count.
func (s *Session) Len() int {
	s.RLock()
	defer s.RUnlock()
	return len(s.sess)
}

func (s *Session) newConn() (conn net.Conn, err error) {
	conn, err = Dial(s.target)
	if err != nil {
		err = errors.Wrap(err, "dialing new session connection failed")
		return
	}
	return
}

// SessionPool is the struct type of session pool.
type SessionPool struct {
	nodeSessions sync.Map // proto.NodeID -> Session
}

func (p *SessionPool) loadSession(id proto.NodeID) (sess *Session, ok bool) {
	var v interface{}
	if v, ok = p.nodeSessions.Load(id); ok {
		sess = v.(*Session)
		return
	}
	v, ok = p.nodeSessions.LoadOrStore(id, &Session{
		target: id,
		sess:   make(chan net.Conn, conf.MaxRPCPoolPhysicalConnection),
	})
	sess = v.(*Session)
	return
}

// Get returns existing session to the node, if not exist try best to create one.
func (p *SessionPool) Get(id proto.NodeID) (conn net.Conn, err error) {
	sess, _ := p.loadSession(id)
	return sess.Get()
}

// GetEx returns an one-off connection if it's anonymous, otherwise returns existing session
// with Get.
func (p *SessionPool) GetEx(id proto.NodeID, isAnonymous bool) (conn net.Conn, err error) {
	if isAnonymous {
		return DialEx(id, true)
	}
	return p.Get(id)
}

// Remove the node sessions in the pool.
func (p *SessionPool) Remove(id proto.NodeID) {
	v, ok := p.nodeSessions.Load(id)
	if ok {
		_ = v.(*Session).Close()
		p.nodeSessions.Delete(id)
	}
	return
}

// Close closes all sessions in the pool.
func (p *SessionPool) Close() error {
	var errmsgs []string
	p.nodeSessions.Range(func(k, v interface{}) bool {
		if err := v.(*Session).Close(); err != nil {
			errmsgs = append(errmsgs, err.Error())
		}
		return true
	})
	if len(errmsgs) > 0 {
		return errors.Wrap(errors.New(strings.Join(errmsgs, ", ")), "close session pool")
	}
	return nil
}

// Len returns the session counts in the pool.
func (p *SessionPool) Len() (total int) {
	p.nodeSessions.Range(func(k, v interface{}) bool {
		total += v.(*Session).Len()
		return true
	})
	return
}

var (
	defaultPool = &SessionPool{}
)
