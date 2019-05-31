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

package mux

import (
	"net"
	"strings"
	"sync"

	"github.com/pkg/errors"
	mux "github.com/xtaci/smux"

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/rpc"
)

// Session is the Session type of SessionPool.
type Session struct {
	sync.RWMutex
	target proto.NodeID
	sess   []*mux.Session
	offset int
}

// SessionPool is the struct type of session pool.
type SessionPool struct {
	sync.RWMutex
	sessions map[proto.NodeID]*Session
}

var (
	defaultPool = &SessionPool{
		sessions: make(map[proto.NodeID]*Session),
	}
)

// GetSessionPoolInstance return default SessionPool instance with rpc.DefaultDialer.
func GetSessionPoolInstance() *SessionPool {
	return defaultPool
}

// Close closes the session.
func (s *Session) Close() error {
	s.Lock()
	defer s.Unlock()
	var errmsgs []string
	for _, s := range s.sess {
		if err := s.Close(); err != nil {
			errmsgs = append(errmsgs, err.Error())
		}
	}
	s.sess = s.sess[:0]
	if len(errmsgs) > 0 {
		return errors.Wrapf(errors.New(strings.Join(errmsgs, ", ")), "close session %s", s.target)
	}
	return nil
}

// Get returns new connection from session.
func (s *Session) Get() (conn rpc.Client, err error) {
	s.Lock()
	defer s.Unlock()
	s.offset++
	s.offset %= conf.MaxRPCMuxPoolPhysicalConnection

	var (
		sess     *mux.Session
		stream   *mux.Stream
		sessions []*mux.Session
	)

	for {
		if len(s.sess) <= s.offset {
			// open new session
			sess, err = s.newSession()
			if err != nil {
				return
			}
			s.sess = append(s.sess, sess)
			s.offset = len(s.sess) - 1
		} else {
			sess = s.sess[s.offset]
		}

		// open connection
		stream, err = sess.OpenStream()
		if err != nil {
			// invalidate session
			sessions = nil
			sessions = append(sessions, s.sess[0:s.offset]...)
			sessions = append(sessions, s.sess[s.offset+1:]...)
			s.sess = sessions
			continue
		}

		return rpc.NewClient(stream), nil
	}
}

// Len returns physical connection count.
func (s *Session) Len() int {
	s.RLock()
	defer s.RUnlock()
	return len(s.sess)
}

func newSession(id proto.NodeID, isAnonymous bool) (sess *mux.Session, err error) {
	var conn net.Conn
	if conn, err = rpc.DialEx(id, isAnonymous); err != nil {
		err = errors.Wrap(err, "dialing new session connection failed")
		return
	}
	return mux.Client(conn, mux.DefaultConfig())
}

func (s *Session) newSession() (sess *mux.Session, err error) {
	return newSession(s.target, false)
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
		}
		p.sessions[id] = sess
	}
	return
}

// Get returns existing session to the node, if not exist try best to create one.
func (p *SessionPool) Get(id proto.NodeID) (conn rpc.Client, err error) {
	var sess *Session
	sess, _ = p.getSession(id)
	return sess.Get()
}

// oneOffMuxConn wraps a mux.Session to implement net.Conn.
type oneOffMuxConn struct {
	*mux.Stream
	sess *mux.Session
}

// Close closes mux.Stream and its underlying mux.Session.
func (c *oneOffMuxConn) Close() error {
	if err := c.Stream.Close(); err != nil {
		return err
	}
	return c.sess.Close()
}

// NewOneOffMuxConn wraps a raw conn as a mux.Stream to access a rpc/mux.Server.
//
// Combine this with rpc.NewClient:
//	Dial conn with a corresponding dialer of RPC server
// 	Wrap conn as stream with NewOneOffMuxConn
// 	Call rpc.NewClient(stream) to get client.
func NewOneOffMuxConn(conn net.Conn) (net.Conn, error) {
	sess, err := mux.Client(conn, mux.DefaultConfig())
	if err != nil {
		return nil, errors.Wrapf(err, "create session to %s", conn.RemoteAddr())
	}
	stream, err := sess.OpenStream()
	if err != nil {
		return nil, errors.Wrapf(err, "open stream to %s", conn.RemoteAddr())
	}
	return &oneOffMuxConn{
		Stream: stream,
		sess:   sess,
	}, nil
}

// GetEx returns an one-off connection if it's anonymous, otherwise returns existing session
// with Get.
func (p *SessionPool) GetEx(id proto.NodeID, isAnonymous bool) (conn rpc.Client, err error) {
	if isAnonymous {
		var (
			sess   *mux.Session
			stream *mux.Stream
		)
		if sess, err = newSession(id, true); err != nil {
			return
		}
		if stream, err = sess.OpenStream(); err != nil {
			err = errors.Wrapf(err, "open new session to %s failed", id)
			return
		}
		return rpc.NewClient(&oneOffMuxConn{
			sess:   sess,
			Stream: stream,
		}), nil
	}
	return p.Get(id)
}

// Remove the node sessions in the pool.
func (p *SessionPool) Remove(id proto.NodeID) {
	p.Lock()
	defer p.Unlock()
	sess, exist := p.sessions[id]
	if exist {
		_ = sess.Close()
		delete(p.sessions, id)
	}
	return
}

// Close closes all sessions in the pool.
func (p *SessionPool) Close() error {
	p.Lock()
	defer p.Unlock()
	var errmsgs []string
	for _, s := range p.sessions {
		if err := s.Close(); err != nil {
			errmsgs = append(errmsgs, err.Error())
		}
	}
	p.sessions = make(map[proto.NodeID]*Session)
	if len(errmsgs) > 0 {
		return errors.Wrap(errors.New(strings.Join(errmsgs, ", ")), "close session pool")
	}
	return nil
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
