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

package transport

import (
	"net"

	log "github.com/sirupsen/logrus"
)

// CryptoListener implement net.Listener
// testPass is used for JUST test
type CryptoListener struct {
	listener net.Listener
	testPass string
}

// NewCryptoListener return a new CryptoListener
func NewCryptoListener(network, addr, pass string) (*CryptoListener, error) {
	l, err := net.Listen(network, addr)
	if err != nil {
		return nil, err
	}
	return &CryptoListener{l, pass}, nil
}

// Accept waits for and returns the next connection to the listener.
func (l *CryptoListener) Accept() (net.Conn, error) {
	c, err := l.listener.Accept()
	if err != nil {
		log.Error(err)
		return nil, err
	}
	cipher := NewCipher([]byte(l.testPass))

	return NewConn(c, cipher), nil
}

// Close closes the listener.
// Any blocked Accept operations will be unblocked and return errors.
func (l *CryptoListener) Close() error {
	return l.listener.Close()
}

// Addr returns the listener's network address.
func (l *CryptoListener) Addr() net.Addr {
	return l.listener.Addr()
}
