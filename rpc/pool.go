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

	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	mux "github.com/xtaci/smux"
)

// SessPool is the session pool interface
type SessPool interface {
	Get(proto.NodeID) (net.Conn, error)
	Set(proto.NodeID, net.Conn) bool
	Remove(proto.NodeID)
	Close()
	Len() int
}

// NodeDialer is the dialer handler
type NodeDialer func(nodeID proto.NodeID) (net.Conn, error)

// SessionMap is the map from node id to session
type SessionMap map[proto.NodeID]*Session

// Session is the Session type of SessionPool
type Session struct {
	ID   proto.NodeID
	Sess *mux.Session
	conn net.Conn
}

// SessionPool is the struct type of session pool
type SessionPool struct {
	sessions   SessionMap
	nodeDialer NodeDialer
	sync.RWMutex
}

var (
	instance *SessionPool
	once     sync.Once
)

// Close closes the session
func (s *Session) Close() {
	s.Sess.Close()
}

// newSessionPool creates a new SessionPool
func newSessionPool(nd NodeDialer) *SessionPool {
	return &SessionPool{
		sessions:   make(SessionMap),
		nodeDialer: nd,
	}
}

// GetSessionPoolInstance return default SessionPool instance with rpc.DefaultDialer
func GetSessionPoolInstance() *SessionPool {
	once.Do(func() {
		instance = newSessionPool(DefaultDialer)
	})
	return instance
}

// toSession wraps net.Conn to mux.Session
func toSession(id proto.NodeID, conn net.Conn) (sess *Session, err error) {
	// create mux session
	newSess, err := mux.Client(conn, YamuxConfig)
	if err != nil {
		//log.Errorf("dial to new node %s failed: %s", id, err)  // no log in lock
		return
	}
	// Store it
	sess = &Session{
		ID:   id,
		Sess: newSess,
		conn: conn,
	}
	return
}

// LoadOrStore returns the existing Session for the node id if present. Otherwise, it stores and
// returns the given Session. The loaded result is true if the Session was loaded, false if stored.
func (p *SessionPool) LoadOrStore(id proto.NodeID, newSess *Session) (sess *Session, loaded bool) {
	// NO Blocking operation in this function
	p.Lock()
	sess, exist := p.sessions[id]
	if exist {
		p.Unlock()
		log.WithField("node", id).Debug("load session for target node")
		loaded = true
	} else {
		p.sessions[id] = newSess
		p.Unlock()
		sess = newSess
	}
	return
}

func (p *SessionPool) getSessionFromPool(id proto.NodeID) (sess *Session, ok bool) {
	sess, ok = p.sessions[id]
	return
}

// Get returns existing session to the node, if not exist try best to create one
func (p *SessionPool) Get(id proto.NodeID) (conn net.Conn, err error) {
	// first try to get one session from pool
	p.Lock()
	cachedConn, ok := p.getSessionFromPool(id)
	p.Unlock()
	if ok {
		conn, err = cachedConn.Sess.OpenStream()
		if err == nil {
			log.WithField("node", id).Debug("reusing session")
			return
		}
		log.WithField("target", id).WithError(err).Error("open session failed")
		p.Remove(id)
	}

	log.WithField("target", id).Debug("dialing new session")
	// Can't find existing Session, try to dial one
	newConn, err := p.nodeDialer(id)
	if err != nil {
		log.WithField("node", id).WithError(err).Error("dial new session failed")
		return
	}
	newSess, err := toSession(id, newConn)
	if err != nil {
		newConn.Close()
		log.WithField("node", id).WithError(err).Error("dial new session failed")
		return
	}
	sess, loaded := p.LoadOrStore(id, newSess)
	if loaded {
		newSess.Close()
	}
	return sess.Sess.OpenStream()
}

// Set tries to set a new connection to the pool, typically from Accept()
// if there is an existing one, just do nothing
func (p *SessionPool) Set(id proto.NodeID, conn net.Conn) (exist bool) {
	sess, err := toSession(id, conn)
	if err != nil {
		return
	}
	_, exist = p.LoadOrStore(id, sess)
	return
}

// Remove the node sessions in the pool
func (p *SessionPool) Remove(id proto.NodeID) {
	p.Lock()
	sess, ok := p.getSessionFromPool(id)
	if ok {
		delete(p.sessions, id)
		p.Unlock()
		sess.Close()
	} else {
		p.Unlock()
	}
}

// Close closes all sessions in the pool
func (p *SessionPool) Close() {
	p.Lock()
	defer p.Unlock()
	for _, s := range p.sessions {
		s.Close()
	}
	p.sessions = make(SessionMap)
}

// Len returns the session counts in the pool
func (p *SessionPool) Len() int {
	p.RLock()
	defer p.RUnlock()
	return len(p.sessions)
}
