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

package etls

import (
	"net"
)

// CipherHandler is the func type for converting net.Conn to CryptoConn
type CipherHandler func(conn net.Conn) (cryptoConn *CryptoConn, err error)

// CryptoListener implements net.Listener
type CryptoListener struct {
	net.Listener
	CHandler CipherHandler
}

// NewCryptoListener returns a new CryptoListener
func NewCryptoListener(network, addr string, handler CipherHandler) (*CryptoListener, error) {
	l, err := net.Listen(network, addr)
	if err != nil {
		return nil, err
	}
	return &CryptoListener{l, handler}, nil
}

// Accept waits for and returns the next connection to the listener.
func (l *CryptoListener) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	return &CryptoConn{
		Conn: c,
	}, nil
}

// Close closes the listener.
// Any blocked Accept operations will be unblocked and return errors.
func (l *CryptoListener) Close() error {
	return l.Listener.Close()
}

// Addr returns the listener's network address.
func (l *CryptoListener) Addr() net.Addr {
	return l.Listener.Addr()
}
