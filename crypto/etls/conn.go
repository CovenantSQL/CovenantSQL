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

// Package etls implements "Enhanced Transport Layer Security", but more efficient
// than TLS used in https.
// example can be found in test case
package etls

import (
	"io"
	"net"
	"time"

	log "github.com/sirupsen/logrus"
)

// CryptoConn implements net.Conn and Cipher interface
type CryptoConn struct {
	conn net.Conn
	*Cipher
}

// NewConn returns a new CryptoConn
func NewConn(c net.Conn, cipher *Cipher) *CryptoConn {
	return &CryptoConn{
		conn:   c,
		Cipher: cipher,
	}
}

// Dial connects to a address with a Cipher
// address should be in the form of host:port
func Dial(network, address string, cipher *Cipher) (c *CryptoConn, err error) {
	conn, err := net.Dial(network, address)
	if err != nil {
		log.Errorf("Connect to %s failed: %v", address, err)
		return
	}
	c = NewConn(conn, cipher)
	c.conn = conn
	return
}

// Read iv and Encrypted data
func (c *CryptoConn) Read(b []byte) (n int, err error) {
	if c.decStream == nil {
		iv := make([]byte, c.info.ivLen)
		if _, err = io.ReadFull(c.conn, iv); err != nil {
			log.Error(err)
			return
		}
		if err = c.initDecrypt(iv); err != nil {
			return
		}
		if len(c.iv) == 0 {
			c.iv = iv
		}
	}

	cipherData := make([]byte, len(b))

	n, err = c.conn.Read(cipherData)
	if err != nil {
		log.Error(err)
		return
	}
	if n > 0 {
		c.decrypt(b[0:n], cipherData[0:n])
	}
	return
}

// Write iv and Encrypted data
func (c *CryptoConn) Write(b []byte) (n int, err error) {
	var iv []byte
	if c.encStream == nil {
		iv, err = c.initEncrypt()
		if err != nil {
			return
		}
	}

	dataSize := len(b) + len(iv)
	cipherData := make([]byte, dataSize)

	if iv != nil {
		// Put initialization vector in buffer, do a single write to send both
		// iv and data.
		copy(cipherData, iv)
	}

	c.encrypt(cipherData[len(iv):], b)
	n, err = c.conn.Write(cipherData)
	return
}

// Close closes the connection.
// Any blocked Read or Write operations will be unblocked and return errors.
func (c *CryptoConn) Close() error {
	return c.conn.Close()
}

// LocalAddr returns the local network address.
func (c *CryptoConn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

// RemoteAddr returns the remote network address.
func (c *CryptoConn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

// SetDeadline sets the read and write deadlines associated with the connection.
// A zero value for t means Read and Write will not time out.
// After a Write has timed out, the TLS state is corrupt and all future writes
// will return the same error.
func (c *CryptoConn) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}

// SetReadDeadline sets the read deadline on the underlying connection.
// A zero value for t means Read will not time out.
func (c *CryptoConn) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

// SetWriteDeadline sets the write deadline on the underlying connection.
// A zero value for t means Write will not time out.
// After a Write has timed out, the TLS state is corrupt and all future writes
// will return the same error.
func (c *CryptoConn) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}
