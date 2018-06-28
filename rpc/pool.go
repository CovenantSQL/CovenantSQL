/*
 * Copyright 2018 The ThunderDB Authors.
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

	"github.com/hashicorp/yamux"
	log "github.com/sirupsen/logrus"
	"gitlab.com/thunderdb/ThunderDB/proto"
)

// RoleType define the remote server role.
type RoleType int

const (
	// UnknownRole is node role can't be determined for now
	UnknownRole = -1
	// BPLeader is a server that have the ability to organize committing requests.
	BPLeader RoleType = iota
	// BPFollower is a server that follow the leader log commits.
	BPFollower
	// DBMiner is a server that run sql database
	DBMiner
	// DBClient is a client that send sql query to database
	DBClient
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
	Role RoleType
	Sess *yamux.Session
}

// SessionPool is the struct type of session pool
type SessionPool struct {
	sessions   SessionMap
	nodeDialer NodeDialer
	sync.RWMutex
}

// Close closes the session
func (s *Session) Close() {
	s.Sess.Close()
}

// NewSessionPool creates a new SessionPool
func NewSessionPool(nd NodeDialer) *SessionPool {
	return &SessionPool{
		sessions:   make(SessionMap),
		nodeDialer: nd,
	}
}

// toSession wraps net.Conn to yamux.Session
func toSession(id proto.NodeID, conn net.Conn) (sess *Session, err error) {
	// create yamux session
	newSess, err := yamux.Client(conn, YamuxConfig)
	if err != nil {
		//log.Errorf("dial to new node %s failed: %s", id, err)  // no log in lock
		return
	}
	// Store it
	sess = &Session{
		ID:   id,
		Role: UnknownRole,
		Sess: newSess,
	}
	return
}

// LoadOrStore returns the existing Session for the node id if present. Otherwise, it stores and
// returns the given Session. The loaded result is true if the Session was loaded, false if stored.
func (p *SessionPool) LoadOrStore(id proto.NodeID, newSess *Session) (sess *Session, loaded bool) {
	// NO Blocking operation in this function
	p.Lock()
	defer p.Unlock()
	sess, exist := p.sessions[id]
	if exist {
		log.Debugf("load session for %s", id)
		loaded = true
	} else {
		sess = newSess
		p.sessions[id] = newSess
	}
	return
}

// Get returns existing session to the node, if not exist try best to create one
func (p *SessionPool) Get(id proto.NodeID) (conn net.Conn, err error) {
	// first try to get one session from pool
	p.Lock()
	cachedConn, ok := p.sessions[id]
	p.Unlock()
	if ok {
		return cachedConn.Sess.Open()
	}

	// Can't find existing Session, try to dial one
	newConn, err := p.nodeDialer(id)
	if err != nil {
		log.Errorf("dial to new node %s failed: %s", id, err)
		return
	}
	newSess, err := toSession(id, newConn)
	if err != nil {
		newConn.Close()
		log.Errorf("dial to new node %s failed: %s", id, err)
		return
	}
	sess, loaded := p.LoadOrStore(id, newSess)
	if loaded {
		newSess.Close()
	}
	return sess.Sess.Open()
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
	defer p.Unlock()
	delete(p.sessions, id)
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
