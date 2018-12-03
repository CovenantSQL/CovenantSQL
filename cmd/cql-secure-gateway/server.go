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

package main

import (
	"net"

	"github.com/CovenantSQL/CovenantSQL/cmd/cql-mysql-adapter/cursor"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	mys "github.com/siddontang/go-mysql/server"
)

// Server defines the main logic of mysql protocol adapter.
type Server struct {
	listener net.Listener
	cfg      *sgConfig
	h        *Handler
}

// NewServer bind the service port and return a runnable adapter.
func NewServer(sc *sgConfig) (s *Server, err error) {
	s = &Server{
		cfg: sc,
		h:   NewHandler(sc.AuthConfig),
	}

	if s.listener, err = net.Listen("tcp", sc.ListenAddr); err != nil {
		return
	}

	return
}

// Serve starts the server.
func (s *Server) Serve() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}

		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	cur := cursor.NewCursor(s.h)
	h, err := mys.NewConnWithUsers(conn, s.cfg.AuthConfig.Users, cur)

	if err != nil {
		log.WithError(err).Error("process connection failed")
		return
	}

	// set connection user to cursor
	cur.SetUser(h.GetUser())

	for {
		err = h.HandleCommand()
		if err != nil {
			return
		}
	}
}

// Shutdown ends the server.
func (s *Server) Shutdown() {
	s.listener.Close()
	s.h.Close()
}
