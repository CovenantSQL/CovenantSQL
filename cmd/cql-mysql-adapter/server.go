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

	my "github.com/go-mysql-org/go-mysql/mysql"
	mys "github.com/go-mysql-org/go-mysql/server"

	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

// mysqlServer handshakes like the go-mysql fork the adapter used before:
// MySQL 5.7.0, utf8_general_ci, mysql_native_password and no TLS. The
// upstream default server announces 8.0.11 and utf8mb4, and offers TLS with a
// self-signed certificate.
var mysqlServer = mys.NewServer("5.7.0", utf8GeneralCI, my.AUTH_NATIVE_PASSWORD, nil, nil)

// Server defines the main logic of mysql protocol adapter.
type Server struct {
	listenAddr string
	listener   net.Listener
	mysqlUser  string
	mysqlAuth  *mys.InMemoryAuthenticationHandler
}

// NewServer bind the service port and return a runnable adapter.
func NewServer(listenAddr string, user string, password string) (s *Server, err error) {
	s = &Server{
		listenAddr: listenAddr,
		mysqlUser:  user,
		// mysqlServer.NewConn would store the password for
		// caching_sha2_password, whose full authentication needs TLS or an RSA
		// key that mysqlServer doesn't have.
		mysqlAuth: mys.NewInMemoryAuthenticationHandler(my.AUTH_NATIVE_PASSWORD),
	}

	if err = s.mysqlAuth.AddUser(user, password); err != nil {
		return
	}

	if s.listener, err = net.Listen("tcp", listenAddr); err != nil {
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
	h, err := mysqlServer.NewCustomizedConn(conn, s.mysqlAuth, NewCursor(s))

	if err != nil {
		log.WithError(err).Error("process connection failed")
		return
	}

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
}
