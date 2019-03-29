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
	"sync"

	"github.com/pkg/errors"

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/proto"
)

type PoolConn struct {
	net.Conn
	session *Session
}

func (c *PoolConn) Close() error {
	return c.session.put(c)
}

// Session is the Session type of SessionPool.
type Session struct {
	sync.RWMutex
	target proto.NodeID
	sess   chan net.Conn
}

// Close closes the session.
func (s *Session) Close() {
	s.Lock()
	defer s.Unlock()
	close(s.sess)
	for s := range s.sess {
		_ = s.Close()
	}
}

// Get returns new connection from session.
func (s *Session) Get() (conn net.Conn, err error) {
	s.Lock()
	defer s.Unlock()

	if conn, ok := <-s.sess; ok {
		return &PoolConn{
			Conn:    conn,
			session: s,
		}, nil
	}
	return s.newConn()
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

func (s *Session) put(conn net.Conn) (err error) {
	s.Lock()
	defer s.Unlock()
	select {
	case s.sess <- conn:
	default:
		return conn.Close()
	}
	return nil
}

// SessionPool is the struct type of session pool.
type SessionPool struct {
	sync.RWMutex
	sessions map[proto.NodeID]*Session
}

func (p *SessionPool) getSession(id proto.NodeID) (sess *Session, loaded bool) {
	// NO Blocking operation in this function
	p.Lock()
	defer p.Unlock()
	sess, exist := p.sessions[id]
	if exist {
		//log.WithField("node", id).Debug("load session for target node")
		loaded = true
	} else {
		// new session
		sess = &Session{
			target: id,
			sess:   make(chan net.Conn, conf.MaxRPCPoolPhysicalConnection),
		}
		p.sessions[id] = sess
	}
	return
}

// Get returns existing session to the node, if not exist try best to create one.
func (p *SessionPool) Get(id proto.NodeID) (conn net.Conn, err error) {
	var sess *Session
	sess, _ = p.getSession(id)
	return sess.Get()
}

// Remove the node sessions in the pool.
func (p *SessionPool) Remove(id proto.NodeID) {
	p.Lock()
	defer p.Unlock()
	sess, exist := p.sessions[id]
	if exist {
		sess.Close()
		delete(p.sessions, id)
	}
	return
}

// Close closes all sessions in the pool.
func (p *SessionPool) Close() {
	p.Lock()
	defer p.Unlock()
	for _, s := range p.sessions {
		s.Close()
	}
	p.sessions = make(map[proto.NodeID]*Session)
}

// Len returns the session counts in the pool.
func (p *SessionPool) Len() (total int) {
	p.RLock()
	defer p.RUnlock()

	for _, s := range p.sessions {
		total += s.Len()
	}
	return
}

var (
	defaultPool = &SessionPool{
		sessions: make(map[proto.NodeID]*Session),
	}
)
